package tools

import (
	"encoding/json"
	"strings"
)

// ToolCallResult is the structured response for tools/call.
type ToolCallResult struct {
	StructuredContent map[string]any `json:"structuredContent,omitempty"`
	Content           []ContentBlock `json:"content"`
	IsError           bool           `json:"isError"`
	Error             string         `json:"error,omitempty"`
}

// ContentBlock represents a single content item in a tool response.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func SuccessResult(structured map[string]any) ToolCallResult {
	if structured == nil {
		structured = map[string]any{}
	}
	structuredText, err := json.Marshal(structured)
	if err != nil {
		structuredText = []byte("{}")
	}
	return ToolCallResult{
		StructuredContent: structured,
		Content: []ContentBlock{{
			Type: "text",
			Text: string(structuredText),
		}},
		IsError: false,
	}
}

func ErrorResult(message string) ToolCallResult {
	if strings.TrimSpace(message) == "" {
		message = "unknown error"
	}
	return ToolCallResult{
		IsError: true,
		Error:   message,
		Content: []ContentBlock{{
			Type: "text",
			Text: message,
		}},
	}
}
