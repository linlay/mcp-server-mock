package mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// ToolSpecRepository loads and caches tool specs from YAML files.
type ToolSpecRepository struct {
	pattern string
	std     *log.Logger

	mu    sync.RWMutex
	tools []map[string]any
}

func NewToolSpecRepository(specLocationPattern string, std *log.Logger) *ToolSpecRepository {
	if strings.TrimSpace(specLocationPattern) == "" {
		specLocationPattern = "./tools/*.yml"
	}
	if std == nil {
		std = log.Default()
	}
	repository := &ToolSpecRepository{
		pattern: specLocationPattern,
		std:     std,
		tools:   []map[string]any{},
	}
	repository.reload()
	return repository
}

func (r *ToolSpecRepository) ListTools() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return deepCopyTools(r.tools)
}

func (r *ToolSpecRepository) reload() {
	loaded := r.loadToolsSafely()
	r.mu.Lock()
	r.tools = loaded
	r.mu.Unlock()
}

func (r *ToolSpecRepository) loadToolsSafely() []map[string]any {
	paths, err := resolvePattern(r.pattern)
	if err != nil {
		r.std.Printf("event=tool.spec.load.failed pattern=%s error=%s", r.pattern, err.Error())
		return []map[string]any{}
	}
	if len(paths) == 0 {
		r.std.Printf("event=tool.spec.load.empty pattern=%s", r.pattern)
		return []map[string]any{}
	}

	sort.Slice(paths, func(i, j int) bool {
		left := strings.ToLower(filepath.Base(paths[i]))
		right := strings.ToLower(filepath.Base(paths[j]))
		if left == right {
			return paths[i] < paths[j]
		}
		return left < right
	})

	tools := make([]map[string]any, 0, len(paths))
	names := make(map[string]struct{}, len(paths))

	for _, path := range paths {
		tool, err := readTool(path)
		if err != nil {
			r.std.Printf("event=tool.spec.load.failed pattern=%s error=%s", r.pattern, err.Error())
			return []map[string]any{}
		}
		if err := validateToolSpec(tool, filepath.Base(path)); err != nil {
			r.std.Printf("event=tool.spec.load.failed pattern=%s error=%s", r.pattern, err.Error())
			return []map[string]any{}
		}

		name := strings.TrimSpace(toString(tool["name"]))
		if _, exists := names[name]; exists {
			r.std.Printf("event=tool.spec.load.failed pattern=%s error=%s", r.pattern, fmt.Sprintf("duplicate tool name: %s in %s", name, filepath.Base(path)))
			return []map[string]any{}
		}
		names[name] = struct{}{}
		tools = append(tools, tool)
	}

	r.std.Printf("event=tool.spec.load.success pattern=%s count=%d", r.pattern, len(tools))
	return tools
}

func resolvePattern(pattern string) ([]string, error) {
	resolved := strings.TrimSpace(pattern)
	if strings.HasPrefix(resolved, "file:") {
		resolved = strings.TrimPrefix(resolved, "file:")
	}
	return filepath.Glob(resolved)
}

func readTool(path string) (map[string]any, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(bytes))) == 0 {
		return nil, fmt.Errorf("empty yaml: %s", filepath.Base(path))
	}
	tool := make(map[string]any)
	if err := yaml.Unmarshal(bytes, &tool); err != nil {
		return nil, err
	}
	if len(tool) == 0 {
		return nil, fmt.Errorf("empty yaml: %s", filepath.Base(path))
	}
	return tool, nil
}

func validateToolSpec(tool map[string]any, filename string) error {
	if tool == nil {
		return fmt.Errorf("tool spec must be object: %s", filename)
	}
	if err := requireText(tool, "type", filename); err != nil {
		return err
	}
	if err := requireText(tool, "name", filename); err != nil {
		return err
	}
	if err := requireText(tool, "description", filename); err != nil {
		return err
	}
	inputSchema, ok := tool["inputSchema"].(map[string]any)
	if !ok || inputSchema == nil {
		return fmt.Errorf("inputSchema must be object: %s", filename)
	}
	return nil
}

func requireText(node map[string]any, field, filename string) error {
	value, exists := node[field]
	if !exists {
		return fmt.Errorf("%s is required: %s", field, filename)
	}
	text := strings.TrimSpace(toString(value))
	if text == "" {
		return fmt.Errorf("%s is required: %s", field, filename)
	}
	return nil
}

func deepCopyTools(source []map[string]any) []map[string]any {
	if len(source) == 0 {
		return []map[string]any{}
	}
	bytes, err := json.Marshal(source)
	if err != nil {
		return []map[string]any{}
	}
	copy := make([]map[string]any, 0, len(source))
	if err := json.Unmarshal(bytes, &copy); err != nil {
		return []map[string]any{}
	}
	return copy
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
