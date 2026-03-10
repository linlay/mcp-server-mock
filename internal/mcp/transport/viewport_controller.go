package transport

import (
	"encoding/json"
	"net/http"
	"strings"

	"mcp-server-mock/internal/api"
	"mcp-server-mock/internal/viewport"
)

type ViewportController struct {
	registry *viewport.Registry
}

func NewViewportController(registry *viewport.Registry) *ViewportController {
	return &ViewportController{registry: registry}
}

func (c *ViewportController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	viewportKey := strings.TrimSpace(r.URL.Query().Get("viewportKey"))
	if viewportKey == "" {
		writeJSON(w, http.StatusBadRequest, api.Failure(http.StatusBadRequest, "viewportKey is required"))
		return
	}
	entry, ok := c.registry.Find(viewportKey)
	if !ok {
		writeJSON(w, http.StatusNotFound, api.Failure(http.StatusNotFound, "Viewport not found: "+viewportKey))
		return
	}
	data := entry.Payload
	if entry.ViewportType == "html" {
		data = map[string]any{"html": entry.Payload}
	}
	writeJSON(w, http.StatusOK, api.Success(data))
}

type ViewportListController struct {
	registry *viewport.Registry
}

func NewViewportListController(registry *viewport.Registry) *ViewportListController {
	return &ViewportListController{registry: registry}
}

func (c *ViewportListController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, api.Success(c.registry.ListSummaries()))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
