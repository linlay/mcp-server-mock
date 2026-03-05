package observability

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"mcp-server-mock/internal/config"
)

// MCPLogger defines the logging interface used by the transport layer.
type MCPLogger interface {
	LogMCPRequest(id any, method string, params any, acceptHeader string, stream bool, headers map[string]string)
	LogMCPResponse(id any, method string, response any, duration time.Duration, contentType string)
	LogMCPError(id any, method string, duration time.Duration, errorType, errorMessage string)
	LogToolRequest(rawToolName, toolName string, args map[string]any)
	LogToolResponse(toolName string, result any, duration time.Duration)
	LogToolError(rawToolName, toolName string, duration time.Duration, err string)
}

// Logger emits structured observability events matching the Java mock server format.
type Logger struct {
	std       *log.Logger
	props     config.ObservabilityConfig
	sanitizer *LogSanitizer
}

// Compile-time check that Logger implements MCPLogger.
var _ MCPLogger = (*Logger)(nil)

func NewLogger(std *log.Logger, props config.ObservabilityConfig, sanitizer *LogSanitizer) *Logger {
	if std == nil {
		std = log.Default()
	}
	if sanitizer == nil {
		sanitizer = NewLogSanitizer(props.LogMaxBodyLength)
	}
	return &Logger{std: std, props: props, sanitizer: sanitizer}
}

func (l *Logger) LogMCPRequest(id any, method string, params any, acceptHeader string, stream bool, headers map[string]string) {
	if !l.props.LogEnabled {
		return
	}
	if l.props.LogIncludeHeaders {
		l.std.Printf(
			"event=mcp.request rpcId=%s method=%s accept=%s isSse=%t paramsSummary=%s headersSummary=%s",
			nodeText(id),
			emptyIfBlank(method),
			emptyIfBlank(acceptHeader),
			stream,
			l.sanitizer.SummarizeJSON(params),
			l.sanitizer.SummarizeJSON(headers),
		)
		return
	}
	l.std.Printf(
		"event=mcp.request rpcId=%s method=%s accept=%s isSse=%t paramsSummary=%s",
		nodeText(id),
		emptyIfBlank(method),
		emptyIfBlank(acceptHeader),
		stream,
		l.sanitizer.SummarizeJSON(params),
	)
}

func (l *Logger) LogMCPResponse(id any, method string, response any, duration time.Duration, contentType string) {
	if !l.props.LogEnabled {
		return
	}
	l.std.Printf(
		"event=mcp.response rpcId=%s method=%s success=%t durationMs=%d contentType=%s responseSummary=%s",
		nodeText(id),
		emptyIfBlank(method),
		isResponseSuccess(response),
		duration.Milliseconds(),
		contentType,
		l.sanitizer.SummarizeJSON(response),
	)
}

func (l *Logger) LogMCPError(id any, method string, duration time.Duration, errorType, errorMessage string) {
	if !l.props.LogEnabled {
		return
	}
	l.std.Printf(
		"event=mcp.error rpcId=%s method=%s durationMs=%d errorType=%s errorMessage=%s",
		nodeText(id),
		emptyIfBlank(method),
		duration.Milliseconds(),
		emptyIfBlank(errorType),
		emptyIfBlank(errorMessage),
	)
}

func (l *Logger) LogToolRequest(rawToolName, toolName string, args map[string]any) {
	if !l.props.LogEnabled {
		return
	}
	l.std.Printf(
		"event=tool.call.request toolRawName=%s toolCanonicalName=%s argsSummary=%s",
		emptyIfBlank(rawToolName),
		emptyIfBlank(toolName),
		l.sanitizer.SummarizeJSON(args),
	)
}

func (l *Logger) LogToolResponse(toolName string, result any, duration time.Duration) {
	if !l.props.LogEnabled {
		return
	}
	structured := map[string]any{}
	isError := false
	if resultMap, ok := toMap(result); ok {
		if raw, ok := resultMap["structuredContent"]; ok {
			if parsed, ok := raw.(map[string]any); ok {
				structured = parsed
			}
		}
		isError = asBool(resultMap["isError"])
	}
	l.std.Printf(
		"event=tool.call.response toolCanonicalName=%s isError=%t durationMs=%d structuredSummary=%s",
		emptyIfBlank(toolName),
		isError,
		duration.Milliseconds(),
		l.sanitizer.SummarizeJSON(structured),
	)
}

func (l *Logger) LogToolError(rawToolName, toolName string, duration time.Duration, err string) {
	if !l.props.LogEnabled {
		return
	}
	l.std.Printf(
		"event=tool.call.error toolRawName=%s toolCanonicalName=%s durationMs=%d error=%s",
		emptyIfBlank(rawToolName),
		emptyIfBlank(toolName),
		duration.Milliseconds(),
		emptyIfBlank(err),
	)
}

func nodeText(value any) string {
	if value == nil {
		return "null"
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func emptyIfBlank(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return value
}

func isResponseSuccess(response any) bool {
	root, ok := response.(map[string]any)
	if !ok {
		return false
	}
	if _, hasError := root["error"]; hasError {
		return false
	}
	rawResult, hasResult := root["result"]
	if !hasResult {
		return false
	}
	result, ok := rawResult.(map[string]any)
	if !ok {
		return true
	}
	if rawIsError, hasIsError := result["isError"]; hasIsError {
		return !asBool(rawIsError)
	}
	return true
}

func asBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func toMap(value any) (map[string]any, bool) {
	if value == nil {
		return nil, false
	}
	if m, ok := value.(map[string]any); ok {
		return m, true
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return nil, false
	}
	return m, true
}

// NopLogger is a no-op implementation of MCPLogger.
type NopLogger struct{}

var _ MCPLogger = NopLogger{}

func (NopLogger) LogMCPRequest(any, string, any, string, bool, map[string]string) {}
func (NopLogger) LogMCPResponse(any, string, any, time.Duration, string)           {}
func (NopLogger) LogMCPError(any, string, time.Duration, string, string)           {}
func (NopLogger) LogToolRequest(string, string, map[string]any)                    {}
func (NopLogger) LogToolResponse(string, any, time.Duration)                       {}
func (NopLogger) LogToolError(string, string, time.Duration, string)               {}
