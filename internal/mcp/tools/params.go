package tools

import (
	"fmt"
	"strings"
)

func readText(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	value, exists := args[key]
	if !exists || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func readAny(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	value, exists := args[key]
	if !exists || value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func formatHM(hour, minute int) string {
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

func city(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "上海"
	}
	mapped, ok := cityMap[strings.ToLower(normalized)]
	if ok {
		return mapped
	}
	return normalized
}

func orValue(args map[string]any, key string, fallback any) any {
	if args == nil {
		return fallback
	}
	value, exists := args[key]
	if !exists {
		return fallback
	}
	return value
}
