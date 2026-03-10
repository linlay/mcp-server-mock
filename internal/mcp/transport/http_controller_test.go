package transport

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"mcp-server-mock/internal/config"
	"mcp-server-mock/internal/mcp/tools"
	"mcp-server-mock/internal/observability"
)

func TestInitializeShouldReturnProtocolVersion(t *testing.T) {
	handler := newMCPTestHandler(t, config.ObservabilityConfig{LogEnabled: true, LogMaxBodyLength: 2000})

	body, status, _ := postRPC(t, handler, rpc("1", "initialize", map[string]any{
		"protocolVersion": "2025-06",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "runner", "version": "0.0.1"},
	}), "")

	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	assertEquals(t, body["jsonrpc"], "2.0")
	assertEquals(t, body["id"], "1")
	result := body["result"].(map[string]any)
	assertEquals(t, result["protocolVersion"], "2025-06")
	serverInfo := result["serverInfo"].(map[string]any)
	assertEquals(t, serverInfo["name"], "mcp-server-mock")
}

func TestToolsListShouldReturnSixCanonicalTools(t *testing.T) {
	handler := newMCPTestHandler(t, config.ObservabilityConfig{LogEnabled: true, LogMaxBodyLength: 2000})

	body, status, _ := postRPC(t, handler, rpc("2", "tools/list", map[string]any{}), "")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	result := body["result"].(map[string]any)
	items := result["tools"].([]any)
	if len(items) != 6 {
		t.Fatalf("expected 6 tools, got %d", len(items))
	}

	names := make([]string, 0, len(items))
	labels := map[string]string{}
	for _, item := range items {
		tool := item.(map[string]any)
		name := tool["name"].(string)
		names = append(names, name)
		label, ok := tool["label"].(string)
		if !ok || label == "" {
			t.Fatalf("expected non-empty label for %s, got %#v", name, tool["label"])
		}
		labels[name] = label
	}
	sort.Strings(names)
	expected := []string{
		"mock.logistics.status",
		"mock.ops.runbook.generate",
		"mock.sensitive-data.detect",
		"mock.todo.tasks.list",
		"mock.transport.schedule.query",
		"mock.weather.query",
	}
	for i := range expected {
		if names[i] != expected[i] {
			t.Fatalf("expected tool %s, got %s", expected[i], names[i])
		}
	}
	expectedLabels := map[string]string{
		"mock.logistics.status":         "物流状态查询",
		"mock.ops.runbook.generate":     "巡检 Runbook 生成",
		"mock.sensitive-data.detect":    "敏感信息检测",
		"mock.todo.tasks.list":          "待办任务列表",
		"mock.transport.schedule.query": "出行班次查询",
		"mock.weather.query":            "天气查询",
	}
	for name, want := range expectedLabels {
		if got := labels[name]; got != want {
			t.Fatalf("expected label %s for %s, got %s", want, name, got)
		}
	}
}

func TestToolsCallShouldReturnStructuredWeatherContent(t *testing.T) {
	handler := newMCPTestHandler(t, config.ObservabilityConfig{LogEnabled: true, LogMaxBodyLength: 2000})

	body, status, _ := postRPC(t, handler, rpc("3", "tools/call", map[string]any{
		"name":      "mock.weather.query",
		"arguments": map[string]any{"city": "shanghai", "date": "2026-02-14"},
	}), "")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	result := body["result"].(map[string]any)
	assertEquals(t, result["isError"], false)
	structured := result["structuredContent"].(map[string]any)
	assertEquals(t, structured["city"], "上海")
	assertEquals(t, structured["date"], "2026-02-14")
	if _, ok := structured["temperatureC"].(float64); !ok {
		t.Fatal("temperatureC should be number")
	}
	assertEquals(t, structured["mockTag"], "幂等随机数据")
}

func TestToolsCallShouldRejectInvalidParamsBySchema(t *testing.T) {
	handler := newMCPTestHandler(t, config.ObservabilityConfig{LogEnabled: true, LogMaxBodyLength: 2000})

	body, status, _ := postRPC(t, handler, rpc("4", "tools/call", map[string]any{
		"name":      "mock.weather.query",
		"arguments": map[string]any{"city": "shanghai"},
	}), "")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	errorNode := body["error"].(map[string]any)
	assertEquals(t, int(errorNode["code"].(float64)), -32602)
	assertContains(t, errorNode["message"].(string), "invalid params")
}

func TestToolsCallShouldRejectAdditionalPropertiesBySchema(t *testing.T) {
	handler := newMCPTestHandler(t, config.ObservabilityConfig{LogEnabled: true, LogMaxBodyLength: 2000})

	body, status, _ := postRPC(t, handler, rpc("5", "tools/call", map[string]any{
		"name": "mock.weather.query",
		"arguments": map[string]any{
			"city":  "shanghai",
			"date":  "2026-02-14",
			"extra": "oops",
		},
	}), "")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	errorNode := body["error"].(map[string]any)
	assertEquals(t, int(errorNode["code"].(float64)), -32602)
	assertContains(t, errorNode["message"].(string), "invalid params")
}

func TestToolsCallShouldReturnToolErrorForUnknownTool(t *testing.T) {
	handler := newMCPTestHandler(t, config.ObservabilityConfig{LogEnabled: true, LogMaxBodyLength: 2000})

	body, status, _ := postRPC(t, handler, rpc("6", "tools/call", map[string]any{
		"name":      "mock.unknown.tool",
		"arguments": map[string]any{"text": "abc"},
	}), "")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	result := body["result"].(map[string]any)
	assertEquals(t, result["isError"], true)
	assertContains(t, result["error"].(string), "unknown tool")
}

func TestToolsListShouldSupportSSEResponse(t *testing.T) {
	handler := newMCPTestHandler(t, config.ObservabilityConfig{LogEnabled: true, LogMaxBodyLength: 2000})

	raw, status, headers := postRawRPC(t, handler, rpc("7", "tools/list", map[string]any{}), "text/event-stream")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	assertContains(t, headers.Get("Content-Type"), "text/event-stream")
	assertContains(t, raw, "data:")
	assertContains(t, raw, `"jsonrpc":"2.0"`)
	assertContains(t, raw, `"tools"`)
	assertContains(t, raw, `"label"`)
	assertContains(t, raw, `"afterCallHint"`)
	assertContains(t, raw, `"toolType"`)
	assertContains(t, raw, `"viewportKey"`)
}

func newMCPTestHandler(t *testing.T, obs config.ObservabilityConfig) http.Handler {
	t.Helper()
	logger := log.New(io.Discard, "", 0)
	registry, err := tools.NewRegistry(testToolsPattern(t), tools.BuiltinHandlers(), logger)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}
	obsLogger := observability.NewLogger(logger, obs, observability.NewLogSanitizer(obs.LogMaxBodyLength))
	controller := NewController(registry, obsLogger, 1024*1024)
	mux := http.NewServeMux()
	mux.Handle("/mcp", controller)
	return mux
}

func postRPC(t *testing.T, handler http.Handler, payload map[string]any, accept string) (map[string]any, int, http.Header) {
	t.Helper()
	raw, status, headers := postRawRPC(t, handler, payload, accept)
	body := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("failed to decode response: %v; body=%s", err, raw)
	}
	return body, status, headers
}

func postRawRPC(t *testing.T, handler http.Handler, payload map[string]any, accept string) (string, int, http.Header) {
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
	return response.Body.String(), response.Code, response.Header()
}

func rpc(id, method string, params map[string]any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
}

func assertEquals(t *testing.T, got any, expected any) {
	t.Helper()
	if got != expected {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func assertContains(t *testing.T, value, expected string) {
	t.Helper()
	if !bytes.Contains([]byte(value), []byte(expected)) {
		t.Fatalf("expected %q to contain %q", value, expected)
	}
}

func testToolsPattern(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve runtime caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	return filepath.Join(root, "tools", "*.yml")
}
