package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"mcp-server-mock-go/internal/observability"
)

// Controller handles the MCP JSON-RPC endpoint.
type Controller struct {
	toolService *ToolService
	logger      *observability.Logger
}

func NewController(toolService *ToolService, logger *observability.Logger) *Controller {
	return &Controller{toolService: toolService, logger: logger}
}

func (c *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	request := map[string]any{}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(string(bodyBytes)) != "" {
		if err := json.Unmarshal(bodyBytes, &request); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
	}

	id := normalizeID(request)
	method := text(request["method"])
	stream := wantsStream(r.Header.Get("Accept"))
	params := request["params"]

	headers := singleValueHeaders(r.Header)
	if c.logger != nil {
		c.logger.LogMCPRequest(id, method, params, r.Header.Get("Accept"), stream, headers)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			if c.logger != nil {
				c.logger.LogMCPError(id, method, time.Since(start), "panic", fmt.Sprint(recovered))
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}()

	response := c.dispatch(request)
	encoded, err := json.Marshal(response)
	if err != nil {
		if c.logger != nil {
			c.logger.LogMCPError(id, method, time.Since(start), "marshal_error", err.Error())
		}
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	if stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: " + string(encoded) + "\n\n"))
		if c.logger != nil {
			c.logger.LogMCPResponse(id, method, response, time.Since(start), "text/event-stream")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(encoded)
	if c.logger != nil {
		c.logger.LogMCPResponse(id, method, response, time.Since(start), "application/json")
	}
}

func (c *Controller) dispatch(request map[string]any) map[string]any {
	id := normalizeID(request)
	method := text(request["method"])
	params, _ := request["params"].(map[string]any)

	switch method {
	case "initialize":
		return rpcSuccess(id, map[string]any{
			"protocolVersion": "2025-06",
			"serverInfo": map[string]any{
				"name":    "mcp-server-mock",
				"version": "0.0.1",
			},
			"capabilities": map[string]any{
				"tools": map[string]any{
					"listChanged": false,
				},
			},
		})
	case "tools/list":
		tools := []map[string]any{}
		if c.toolService != nil {
			tools = c.toolService.ListTools()
		}
		return rpcSuccess(id, map[string]any{"tools": tools})
	case "tools/call":
		toolName := ""
		args := map[string]any{}
		if params != nil {
			toolName = text(params["name"])
			if rawArgs, ok := params["arguments"].(map[string]any); ok && rawArgs != nil {
				args = rawArgs
			}
		}
		result := map[string]any{}
		if c.toolService != nil {
			result = c.toolService.CallTool(toolName, args)
		}
		return rpcSuccess(id, result)
	default:
		return rpcError(id, -32601, "method not found: "+method)
	}
}

func normalizeID(request map[string]any) any {
	if request == nil {
		return nil
	}
	id, exists := request["id"]
	if !exists {
		return nil
	}
	return id
}

func text(value any) string {
	if value == nil {
		return ""
	}
	raw := strings.TrimSpace(fmt.Sprint(value))
	if raw == "" {
		return ""
	}
	return raw
}

func wantsStream(acceptHeader string) bool {
	if strings.TrimSpace(acceptHeader) == "" {
		return false
	}
	return strings.Contains(strings.ToLower(acceptHeader), "text/event-stream")
}

func singleValueHeaders(header http.Header) map[string]string {
	single := make(map[string]string, len(header))
	for key, values := range header {
		if len(values) == 0 {
			single[key] = ""
			continue
		}
		single[key] = values[0]
	}
	return single
}
