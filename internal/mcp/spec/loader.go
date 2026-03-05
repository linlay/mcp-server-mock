package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadFromPattern(pattern string) ([]ToolSpec, error) {
	paths, err := resolvePattern(pattern)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return []ToolSpec{}, nil
	}

	sort.Slice(paths, func(i, j int) bool {
		left := strings.ToLower(filepath.Base(paths[i]))
		right := strings.ToLower(filepath.Base(paths[j]))
		if left == right {
			return paths[i] < paths[j]
		}
		return left < right
	})

	specs := make([]ToolSpec, 0, len(paths))
	for _, path := range paths {
		item, err := readToolSpec(path)
		if err != nil {
			return nil, err
		}
		specs = append(specs, item)
	}
	return specs, nil
}

func resolvePattern(pattern string) ([]string, error) {
	resolved := strings.TrimSpace(pattern)
	if strings.HasPrefix(resolved, "file:") {
		resolved = strings.TrimPrefix(resolved, "file:")
	}
	return filepath.Glob(resolved)
}

func readToolSpec(path string) (ToolSpec, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return ToolSpec{}, err
	}
	if len(strings.TrimSpace(string(bytes))) == 0 {
		return ToolSpec{}, fmt.Errorf("empty yaml: %s", filepath.Base(path))
	}

	raw := map[string]any{}
	if err := yaml.Unmarshal(bytes, &raw); err != nil {
		return ToolSpec{}, fmt.Errorf("invalid yaml %s: %w", filepath.Base(path), err)
	}
	if len(raw) == 0 {
		return ToolSpec{}, fmt.Errorf("empty yaml: %s", filepath.Base(path))
	}

	normalizedRaw, err := normalize(raw)
	if err != nil {
		return ToolSpec{}, fmt.Errorf("normalize yaml %s: %w", filepath.Base(path), err)
	}

	spec := ToolSpec{}
	if err := yaml.Unmarshal(bytes, &spec); err != nil {
		return ToolSpec{}, fmt.Errorf("decode spec %s: %w", filepath.Base(path), err)
	}
	spec.Raw = normalizedRaw
	spec.Source = filepath.Base(path)
	return spec, nil
}

func normalize(value any) (map[string]any, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}
