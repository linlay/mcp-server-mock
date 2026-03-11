package tools

import (
	"context"
	"strings"
)

type SensitiveHandler struct{}

func (SensitiveHandler) Name() string {
	return ToolSensitive
}

func (SensitiveHandler) Call(_ context.Context, call ToolCall) (map[string]any, error) {
	args := call.Arguments
	text := firstNonBlank(
		readAny(args, "text"),
		readAny(args, "content"),
		readAny(args, "message"),
		readAny(args, "query"),
		readAny(args, "document"),
		readAny(args, "input"),
	)

	if strings.TrimSpace(text) == "" {
		return map[string]any{
			"hasSensitiveData": false,
			"result":           "没有敏感数据",
			"description":      "未检测到可分析文本。",
		}, nil
	}

	for _, rule := range sensitiveRules {
		if rule.pattern.MatchString(text) {
			return map[string]any{
				"hasSensitiveData": true,
				"result":           "有敏感数据",
				"description":      "检测到疑似" + rule.label + "信息，建议脱敏后再传输。",
			}, nil
		}
	}

	return map[string]any{
		"hasSensitiveData": false,
		"result":           "没有敏感数据",
		"description":      "未发现明显敏感字段特征。",
	}, nil
}
