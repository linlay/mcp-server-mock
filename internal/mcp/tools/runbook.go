package tools

import (
	"context"
	"fmt"
)

type RunbookHandler struct{}

func (RunbookHandler) Name() string {
	return ToolRunbook
}

func (RunbookHandler) Call(_ context.Context, args map[string]any) (map[string]any, error) {
	message := fmt.Sprint(orValue(args, "message", orValue(args, "query", "")))
	cityName := fmt.Sprint(orValue(args, "city", "Shanghai"))
	random := randomByArgs(args)
	command := "ls -la"
	if random.NextBool() {
		command = "df -h"
	}

	return map[string]any{
		"message":            message,
		"city":               cityName,
		"riskLevel":          riskLevels[random.NextInt(len(riskLevels))],
		"recommendedCommand": command,
		"steps": []string{
			"检查系统负载与磁盘利用率",
			"确认业务实例状态",
			"输出巡检摘要",
		},
		"mockTag": "idempotent-random-json",
	}, nil
}
