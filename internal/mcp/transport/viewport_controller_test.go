package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"mcp-server-mock/internal/viewport"
)

func TestViewportControllerShouldReturnHtmlPayload(t *testing.T) {
	registry := mustViewportRegistry(t)
	defer registry.Close()

	request := httptest.NewRequest(http.MethodGet, "/api/ap/viewport?viewportKey=show_weather_card", nil)
	response := httptest.NewRecorder()
	NewViewportController(registry).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := decodeBody(t, response.Body.Bytes())
	data := body["data"].(map[string]any)
	if _, ok := data["html"].(string); !ok {
		t.Fatalf("expected html string, got %#v", data)
	}
}

func TestViewportControllerShouldReturnNotFound(t *testing.T) {
	registry := mustViewportRegistry(t)
	defer registry.Close()

	request := httptest.NewRequest(http.MethodGet, "/api/ap/viewport?viewportKey=missing", nil)
	response := httptest.NewRecorder()
	NewViewportController(registry).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
}

func TestViewportListControllerShouldReturnViewportSummaries(t *testing.T) {
	registry := mustViewportRegistry(t)
	defer registry.Close()

	request := httptest.NewRequest(http.MethodGet, "/api/ap/viewports", nil)
	response := httptest.NewRecorder()
	NewViewportListController(registry).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := decodeBody(t, response.Body.Bytes())
	data := body["data"].([]any)
	if len(data) == 0 {
		t.Fatal("expected non-empty summaries")
	}
}

func mustViewportRegistry(t *testing.T) *viewport.Registry {
	t.Helper()
	registry, err := viewport.NewRegistry(projectViewportsDir(t), 0, map[string][]string{
		"show_weather_card":     {"mock.weather.query"},
		"show_logistics_status": {"mock.logistics.status"},
		"show_transport_card":   {"mock.transport.schedule.query"},
		"show_todo_card":        {"mock.todo.tasks.list"},
	}, nil)
	if err != nil {
		t.Fatalf("create viewport registry: %v", err)
	}
	return registry
}

func projectViewportsDir(t *testing.T) string {
	t.Helper()
	return testRootDir(t) + "/viewports"
}

func testRootDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve runtime caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

func decodeBody(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return out
}
