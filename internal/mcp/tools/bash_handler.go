package tools

import (
	"context"

	"mcp-server-mock/internal/config"
)

type BashHandler struct {
	Executor *BashExecutor
}

func NewBashHandler(cfg config.BashConfig) BashHandler {
	return BashHandler{Executor: NewBashExecutor(cfg)}
}

func (h BashHandler) Name() string {
	return ToolBash
}

func (h BashHandler) Call(ctx context.Context, call ToolCall) (map[string]any, error) {
	result := h.Executor.Execute(ctx, readString(call.Arguments, "command"), call.WorkDirectory, call.UserID)
	return map[string]any{
		"exitCode":         result.ExitCode,
		"workingDirectory": result.WorkingDirectory,
		"userId":           result.UserID,
		"stdout":           result.Stdout,
		"stderr":           result.Stderr,
		"text":             result.Text,
	}, nil
}
