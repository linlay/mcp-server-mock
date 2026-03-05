package transport

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"mcp-server-mock/internal/mcp/protocol"
	"mcp-server-mock/internal/mcp/schema"
	"mcp-server-mock/internal/mcp/tools"
	"mcp-server-mock/internal/observability"
)

type Controller struct {
	registry     *tools.Registry
	logger       *observability.Logger
	maxBodyBytes int64
}

func NewController(registry *tools.Registry, logger *observability.Logger, maxBodyBytes int64) *Controller {
	if maxBodyBytes <= 0 {
		maxBodyBytes = 1024 * 1024
	}
	return &Controller{registry: registry, logger: logger, maxBodyBytes: maxBodyBytes}
}

func (c *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, c.maxBodyBytes))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		c.writeResponse(w, r, protocol.NewError(nil, protocol.ErrCodeInvalidRequest, "invalid request: empty body"), start, "")
		return
	}

	req, err := protocol.DecodeRequest(body)
	if err != nil {
		c.writeResponse(w, r, protocol.NewError(nil, protocol.ErrCodeParseError, "parse error: invalid json"), start, "")
		return
	}
	if err := protocol.ValidateRequest(req); err != nil {
		c.writeResponse(w, r, protocol.NewError(req.ID, protocol.ErrCodeInvalidRequest, "invalid request: "+err.Error()), start, req.Method)
		return
	}

	accept := r.Header.Get("Accept")
	stream := wantsStream(accept)
	params := decodeParamsSummary(req.Params)
	headers := singleValueHeaders(r.Header)
	if c.logger != nil {
		c.logger.LogMCPRequest(req.ID, req.Method, params, accept, stream, headers)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			if c.logger != nil {
				c.logger.LogMCPError(req.ID, req.Method, time.Since(start), "panic", fmt.Sprint(recovered))
			}
			c.writeResponse(w, r, protocol.NewError(req.ID, protocol.ErrCodeInternal, "internal server error"), start, req.Method)
		}
	}()

	resp := c.dispatch(r, req, start)
	c.writeResponse(w, r, resp, start, req.Method)
}

func (c *Controller) dispatch(r *http.Request, req protocol.RPCRequest, start time.Time) protocol.RPCResponse {
	switch req.Method {
	case "initialize":
		return protocol.NewSuccess(req.ID, map[string]any{
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
		if c.registry == nil {
			return protocol.NewSuccess(req.ID, map[string]any{"tools": []map[string]any{}})
		}
		return protocol.NewSuccess(req.ID, map[string]any{"tools": c.registry.ListTools()})
	case "tools/call":
		return c.dispatchToolsCall(r, req, start)
	default:
		return protocol.NewError(req.ID, protocol.ErrCodeMethodNotFound, "method not found: "+req.Method)
	}
}

func (c *Controller) dispatchToolsCall(r *http.Request, req protocol.RPCRequest, start time.Time) protocol.RPCResponse {
	params := protocol.ToolsCallParams{}
	if err := protocol.DecodeParams(req.Params, &params); err != nil {
		return protocol.NewError(req.ID, protocol.ErrCodeInvalidParams, "invalid params: expected object")
	}
	if strings.TrimSpace(params.Name) == "" {
		return protocol.NewError(req.ID, protocol.ErrCodeInvalidParams, "invalid params: name is required")
	}
	if params.Arguments == nil {
		params.Arguments = map[string]any{}
	}

	item, ok := c.registry.Find(params.Name)
	canonicalName := ""
	if ok {
		canonicalName = item.Spec.Name
	}
	if c.logger != nil {
		c.logger.LogToolRequest(params.Name, canonicalName, params.Arguments)
	}

	if !ok {
		result := tools.ErrorResult("unknown tool: " + strings.TrimSpace(params.Name))
		if c.logger != nil {
			c.logger.LogToolError(params.Name, canonicalName, time.Since(start), result["error"].(string))
		}
		return protocol.NewSuccess(req.ID, result)
	}

	if err := schema.Validate(item.CompiledSchema, params.Arguments); err != nil {
		if c.logger != nil {
			c.logger.LogToolError(params.Name, item.Spec.Name, time.Since(start), err.Error())
		}
		return protocol.NewError(req.ID, protocol.ErrCodeInvalidParams, "invalid params: "+err.Error())
	}

	structured, err := item.Handler.Call(r.Context(), params.Arguments)
	if err != nil {
		if c.logger != nil {
			c.logger.LogToolError(params.Name, item.Spec.Name, time.Since(start), err.Error())
		}
		return protocol.NewSuccess(req.ID, tools.ErrorResult(err.Error()))
	}

	result := tools.SuccessResult(structured)
	if c.logger != nil {
		c.logger.LogToolResponse(item.Spec.Name, result, time.Since(start))
	}
	return protocol.NewSuccess(req.ID, result)
}

func (c *Controller) writeResponse(w http.ResponseWriter, r *http.Request, response protocol.RPCResponse, start time.Time, method string) {
	encoded, err := json.Marshal(response)
	if err != nil {
		if c.logger != nil {
			c.logger.LogMCPError(response.ID, method, time.Since(start), "marshal_error", err.Error())
		}
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	stream := wantsStream(r.Header.Get("Accept"))
	if stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: " + string(encoded) + "\n\n"))
		if c.logger != nil {
			c.logger.LogMCPResponse(response.ID, method, responseToMap(response), time.Since(start), "text/event-stream")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(encoded)
	if c.logger != nil {
		c.logger.LogMCPResponse(response.ID, method, responseToMap(response), time.Since(start), "application/json")
	}
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

func decodeParamsSummary(raw json.RawMessage) any {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func responseToMap(response protocol.RPCResponse) map[string]any {
	payload, err := json.Marshal(response)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal(payload, &out); err != nil {
		return map[string]any{}
	}
	return out
}
