package tools

import "context"

const (
	ToolWeather   = "mock.weather.query"
	ToolLogistics = "mock.logistics.status"
	ToolRunbook   = "mock.ops.runbook.generate"
	ToolSensitive = "mock.sensitive-data.detect"
	ToolTodo      = "mock.todo.tasks.list"
	ToolTransport = "mock.transport.schedule.query"
)

type ToolHandler interface {
	Name() string
	Call(ctx context.Context, args map[string]any) (map[string]any, error)
}

func BuiltinHandlers() []ToolHandler {
	return []ToolHandler{
		WeatherHandler{},
		LogisticsHandler{},
		RunbookHandler{},
		SensitiveHandler{},
		TodoHandler{},
		TransportHandler{},
	}
}
