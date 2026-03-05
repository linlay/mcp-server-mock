package mcp

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

	"mcp-server-mock-go/internal/config"
	"mcp-server-mock-go/internal/observability"
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
	tools := result["tools"].([]any)
	if len(tools) != 6 {
		t.Fatalf("expected 6 tools, got %d", len(tools))
	}

	names := make([]string, 0, len(tools))
	for _, item := range tools {
		tool := item.(map[string]any)
		names = append(names, tool["name"].(string))
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

	weather := findTool(t, tools, "mock.weather.query")
	logistics := findTool(t, tools, "mock.logistics.status")
	runbook := findTool(t, tools, "mock.ops.runbook.generate")
	sensitive := findTool(t, tools, "mock.sensitive-data.detect")
	todo := findTool(t, tools, "mock.todo.tasks.list")
	transport := findTool(t, tools, "mock.transport.schedule.query")

	assertEquals(t, weather["type"], "function")
	assertEquals(t, logistics["type"], "function")
	assertEquals(t, runbook["type"], "function")
	assertEquals(t, sensitive["type"], "function")
	assertEquals(t, todo["type"], "function")
	assertEquals(t, transport["type"], "function")

	assertContains(t, weather["description"].(string), "[MOCK] 根据 city 和 date 查询天气（伪造数据）。")
	assertContains(t, logistics["description"].(string), "[MOCK] 根据 trackingNo 查询物流状态（伪造数据）。")
	assertContains(t, runbook["description"].(string), "[MOCK] 根据 message 生成巡检 runbook（伪造数据）。")
	assertContains(t, sensitive["description"].(string), "[MOCK] 检测超长文本中是否包含敏感数据")
	assertContains(t, todo["description"].(string), "[MOCK] 生成待办任务列表（伪造数据）。")
	assertContains(t, transport["description"].(string), "[MOCK] 根据出发地、目的地和日期生成航班或高铁行程（伪造数据）。")

	assertContains(t, weather["afterCallHint"].(string), "show_weather_card")
	assertContains(t, logistics["afterCallHint"].(string), "show_logistics_status")
	assertContains(t, todo["afterCallHint"].(string), "show_todo_card")
	assertContains(t, transport["afterCallHint"].(string), "show_transport_card")
	if _, ok := runbook["afterCallHint"]; ok {
		t.Fatal("runbook should not have afterCallHint")
	}
	if _, ok := sensitive["afterCallHint"]; ok {
		t.Fatal("sensitive tool should not have afterCallHint")
	}

	assertAdditionalPropertiesFalse(t, weather)
	assertAdditionalPropertiesFalse(t, logistics)
	assertAdditionalPropertiesFalse(t, runbook)
	assertAdditionalPropertiesFalse(t, sensitive)
	assertAdditionalPropertiesFalse(t, todo)
	assertAdditionalPropertiesFalse(t, transport)

	weatherRequired := asStringSlice(weather["inputSchema"].(map[string]any)["required"].([]any))
	if !containsExactlyInAnyOrder(weatherRequired, []string{"city", "date"}) {
		t.Fatalf("unexpected weather required fields: %v", weatherRequired)
	}
	logisticsRequired := asStringSlice(logistics["inputSchema"].(map[string]any)["required"].([]any))
	if !containsExactlyInAnyOrder(logisticsRequired, []string{"trackingNo"}) {
		t.Fatalf("unexpected logistics required fields: %v", logisticsRequired)
	}
	if _, ok := runbook["inputSchema"].(map[string]any)["required"]; ok {
		t.Fatal("runbook required should not exist")
	}
	if _, ok := sensitive["inputSchema"].(map[string]any)["required"]; ok {
		t.Fatal("sensitive required should not exist")
	}
	if _, ok := todo["inputSchema"].(map[string]any)["required"]; ok {
		t.Fatal("todo required should not exist")
	}
	if _, ok := transport["inputSchema"].(map[string]any)["required"]; ok {
		t.Fatal("transport required should not exist")
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

func TestToolsCallShouldRejectLegacyToolAlias(t *testing.T) {
	handler := newMCPTestHandler(t, config.ObservabilityConfig{LogEnabled: true, LogMaxBodyLength: 2000})

	body, status, _ := postRPC(t, handler, rpc("3-alias", "tools/call", map[string]any{
		"name":      "mock_city_weather",
		"arguments": map[string]any{"city": "shanghai", "date": "2026-02-14"},
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

	raw, status, headers := postRawRPC(t, handler, rpc("4", "tools/list", map[string]any{}), "text/event-stream")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	assertContains(t, headers.Get("Content-Type"), "text/event-stream")
	assertContains(t, raw, "data:")
	assertContains(t, raw, `"jsonrpc":"2.0"`)
	assertContains(t, raw, `"tools"`)
	assertContains(t, raw, `"afterCallHint"`)
	assertContains(t, raw, "show_weather_card")
}

func newMCPTestHandler(t *testing.T, obs config.ObservabilityConfig) http.Handler {
	t.Helper()
	logger := log.New(io.Discard, "", 0)
	repo := NewToolSpecRepository(testToolsPattern(t), logger)
	obsLogger := observability.NewLogger(logger, obs, observability.NewLogSanitizer(obs.LogMaxBodyLength))
	service := NewToolService(repo, obsLogger)
	controller := NewController(service, obsLogger)
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

func findTool(t *testing.T, tools []any, name string) map[string]any {
	t.Helper()
	for _, item := range tools {
		tool := item.(map[string]any)
		if tool["name"] == name {
			return tool
		}
	}
	t.Fatalf("tool not found: %s", name)
	return nil
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

func assertAdditionalPropertiesFalse(t *testing.T, tool map[string]any) {
	t.Helper()
	inputSchema := tool["inputSchema"].(map[string]any)
	if inputSchema["additionalProperties"] != false {
		t.Fatalf("expected additionalProperties=false for tool %v", tool["name"])
	}
}

func containsExactlyInAnyOrder(actual []string, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	sortedActual := append([]string(nil), actual...)
	sortedExpected := append([]string(nil), expected...)
	sort.Strings(sortedActual)
	sort.Strings(sortedExpected)
	for i := range sortedActual {
		if sortedActual[i] != sortedExpected[i] {
			return false
		}
	}
	return true
}

func asStringSlice(values []any) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.(string))
	}
	return result
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
