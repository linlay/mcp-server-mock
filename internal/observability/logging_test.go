package observability_test

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"mcp-server-mock/internal/config"
	"mcp-server-mock/internal/mcp/tools"
	"mcp-server-mock/internal/mcp/transport"
	"mcp-server-mock/internal/observability"
)

func TestShouldLogRequestAndResponseByDefault(t *testing.T) {
	handler, logs := newLoggingTestHandler(t, config.ObservabilityConfig{LogEnabled: true, LogMaxBodyLength: 2000})

	postRPC(t, handler, rpc("log-1", "tools/list", map[string]any{}), "")

	output := logs.String()
	assertContains(t, output, "event=mcp.request")
	assertContains(t, output, "event=mcp.response")
	assertContains(t, output, "rpcId=log-1")
}

func TestShouldLogToolCallRequestAndResponse(t *testing.T) {
	handler, logs := newLoggingTestHandler(t, config.ObservabilityConfig{LogEnabled: true, LogMaxBodyLength: 2000})

	postRPC(t, handler, rpc("log-2", "tools/call", map[string]any{
		"name":      "mock.weather.query",
		"arguments": map[string]any{"city": "shanghai", "date": "2026-02-14"},
	}), "")

	output := logs.String()
	assertContains(t, output, "event=tool.call.request")
	assertContains(t, output, "event=tool.call.response")
	assertContains(t, output, "toolCanonicalName=mock.weather.query")
}

func TestShouldMaskSensitiveValuesInLogs(t *testing.T) {
	handler, logs := newLoggingTestHandler(t, config.ObservabilityConfig{LogEnabled: true, LogMaxBodyLength: 2000})

	postRPC(t, handler, rpc("log-3", "tools/call", map[string]any{
		"name": "mock.weather.query",
		"arguments": map[string]any{
			"city":     "shanghai",
			"password": "my-secret-password",
			"apiKey":   "sk-abc-123456",
		},
	}), "")

	output := logs.String()
	assertContains(t, output, `"password":"***"`)
	assertContains(t, output, `"apiKey":"***"`)
	assertNotContains(t, output, "my-secret-password")
	assertNotContains(t, output, "sk-abc-123456")
}

func TestShouldLogUnknownToolAsError(t *testing.T) {
	handler, logs := newLoggingTestHandler(t, config.ObservabilityConfig{LogEnabled: true, LogMaxBodyLength: 2000})

	postRPC(t, handler, rpc("log-4", "tools/call", map[string]any{
		"name":      "mock.unknown.tool",
		"arguments": map[string]any{"text": "abc"},
	}), "")

	output := logs.String()
	assertContains(t, output, "event=tool.call.error")
	assertContains(t, output, "event=mcp.response")
	assertContains(t, output, "success=false")
}

func TestShouldNotLogObservabilityEventsWhenDisabled(t *testing.T) {
	handler, logs := newLoggingTestHandler(t, config.ObservabilityConfig{LogEnabled: false, LogMaxBodyLength: 2000})

	postRPC(t, handler, rpc("disabled-1", "tools/list", map[string]any{}), "")

	output := logs.String()
	assertNotContains(t, output, "event=mcp.request")
	assertNotContains(t, output, "event=mcp.response")
	assertNotContains(t, output, "event=tool.call.request")
	assertNotContains(t, output, "event=tool.call.response")
}

func TestShouldTruncateLongLogSummary(t *testing.T) {
	handler, logs := newLoggingTestHandler(t, config.ObservabilityConfig{LogEnabled: true, LogMaxBodyLength: 120})

	postRPC(t, handler, rpc("truncate-1", "tools/call", map[string]any{
		"name":      "mock.sensitive-data.detect",
		"arguments": map[string]any{"text": strings.Repeat("X", 2000)},
	}), "")

	assertContains(t, logs.String(), "...(truncated)")
}

func newLoggingTestHandler(t *testing.T, obs config.ObservabilityConfig) (http.Handler, *bytes.Buffer) {
	t.Helper()
	buffer := &bytes.Buffer{}
	logger := log.New(buffer, "", 0)
	registry, err := tools.NewRegistry(testToolsPattern(t), tools.BuiltinHandlers(), logger)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}
	sanitizer := observability.NewLogSanitizer(obs.LogMaxBodyLength)
	obsLogger := observability.NewLogger(logger, obs, sanitizer)
	controller := transport.NewController(registry, obsLogger, 1024*1024)
	mux := http.NewServeMux()
	mux.Handle("/mcp", controller)
	return mux, buffer
}

func postRPC(t *testing.T, handler http.Handler, payload map[string]any, accept string) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if accept != "" {
		request.Header.Set("Accept", accept)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d, body=%s", response.Code, response.Body.String())
	}
}

func rpc(id, method string, params map[string]any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
}

func testToolsPattern(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve runtime caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	return filepath.Join(root, "tools", "*.yml")
}

func assertContains(t *testing.T, value, expected string) {
	t.Helper()
	if !strings.Contains(value, expected) {
		t.Fatalf("expected logs to contain %q\nlogs=%s", expected, value)
	}
}

func assertNotContains(t *testing.T, value, expected string) {
	t.Helper()
	if strings.Contains(value, expected) {
		t.Fatalf("expected logs to not contain %q\nlogs=%s", expected, value)
	}
}
