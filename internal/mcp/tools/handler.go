package tools

import (
	"context"

	"mcp-server-mock/internal/config"
)

const (
	ToolBash      = "bash"
	ToolWeather   = "mock.weather.query"
	ToolLogistics = "mock.logistics.status"
	ToolRunbook   = "mock.ops.runbook.generate"
	ToolSensitive = "mock.sensitive-data.detect"
	ToolTodo      = "mock.todo.tasks.list"
	ToolTransport = "mock.transport.schedule.query"
)

type ToolCall struct {
	Arguments     map[string]any
	Meta          map[string]any
	WorkDirectory string
	UserID        string
}

type ToolHandler interface {
	Name() string
	Call(ctx context.Context, call ToolCall) (map[string]any, error)
}

func NewToolCall(arguments map[string]any, meta map[string]any) ToolCall {
	if arguments == nil {
		arguments = map[string]any{}
	}
	if meta == nil {
		meta = map[string]any{}
	}
	return ToolCall{
		Arguments:     arguments,
		Meta:          meta,
		WorkDirectory: readText(meta, "workDirectory"),
		UserID:        readText(meta, "userId"),
	}
}

func BuiltinHandlers(cfg config.BashConfig) []ToolHandler {
	return []ToolHandler{
		NewBashHandler(cfg),
		WeatherHandler{},
		LogisticsHandler{},
		RunbookHandler{},
		SensitiveHandler{},
		TodoHandler{},
		TransportHandler{},
	}
}
