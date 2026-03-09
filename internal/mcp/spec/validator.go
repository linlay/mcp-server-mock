package spec

import (
	"fmt"
	"strings"

	"mcp-server-mock/internal/mcp/schema"
)

func ValidateDefinitions(specs []ToolSpec) error {
	meta, err := schema.Compile("tool-meta-schema", MetaSchema())
	if err != nil {
		return fmt.Errorf("compile tool meta schema: %w", err)
	}

	names := make(map[string]string, len(specs))
	for _, item := range specs {
		raw := SpecToMap(item)
		if err := schema.Validate(meta, raw); err != nil {
			return fmt.Errorf("invalid tool spec %s: %w", item.Source, err)
		}
		if err := validateToolMode(item); err != nil {
			return fmt.Errorf("invalid tool spec %s: %w", item.Source, err)
		}

		normalized := strings.ToLower(strings.TrimSpace(item.Name))
		if normalized == "" {
			return fmt.Errorf("tool name is required: %s", item.Source)
		}
		if first, exists := names[normalized]; exists {
			return fmt.Errorf("duplicate tool name: %s in %s and %s", item.Name, first, item.Source)
		}
		names[normalized] = item.Source
	}
	return nil
}

func validateToolMode(item ToolSpec) error {
	hasToolType := strings.TrimSpace(item.ToolType) != ""
	hasViewportKey := strings.TrimSpace(item.ViewportKey) != ""

	if item.ToolAction && (hasToolType || hasViewportKey) {
		return fmt.Errorf("toolAction=true cannot be combined with toolType or viewportKey")
	}
	if hasToolType != hasViewportKey {
		return fmt.Errorf("toolType and viewportKey must be declared together")
	}
	return nil
}

// SpecToMap builds a map representation of a ToolSpec from its structured fields.
func SpecToMap(s ToolSpec) map[string]any {
	m := map[string]any{
		"type":        s.Type,
		"name":        s.Name,
		"description": s.Description,
		"inputSchema": s.InputSchema,
	}
	if s.AfterCallHint != "" {
		m["afterCallHint"] = s.AfterCallHint
	}
	if s.ToolAction {
		m["toolAction"] = true
	}
	if strings.TrimSpace(s.ToolType) != "" {
		m["toolType"] = strings.TrimSpace(s.ToolType)
	}
	if strings.TrimSpace(s.ViewportKey) != "" {
		m["viewportKey"] = strings.TrimSpace(s.ViewportKey)
	}
	return m
}
