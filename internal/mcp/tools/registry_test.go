package tools

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistryShouldLoadAndListTools(t *testing.T) {
	r, err := NewRegistry(projectToolsPattern(t), BuiltinHandlers(), log.New(os.Stdout, "", 0))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	listed := r.ListTools()
	if len(listed) != 6 {
		t.Fatalf("expected 6 tools, got %d", len(listed))
	}
}

func TestRegistryShouldFailWhenInputSchemaInvalid(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "invalid.yml"), `type: function
name: mock.weather.query
description: test
inputSchema:
  type: 123
`)

	_, err := NewRegistry(filePattern(dir), []ToolHandler{stubHandler{name: "mock.weather.query"}}, log.New(os.Stdout, "", 0))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "compile inputSchema") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegistryShouldFailWhenDuplicateToolName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.yml"), validTool("mock.weather.query"))
	writeFile(t, filepath.Join(dir, "b.yml"), validTool("mock.weather.query"))

	_, err := NewRegistry(filePattern(dir), []ToolHandler{stubHandler{name: "mock.weather.query"}}, log.New(os.Stdout, "", 0))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "duplicate tool name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegistryShouldFailWhenSpecHasNoHandler(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.yml"), validTool("mock.weather.query"))

	_, err := NewRegistry(filePattern(dir), []ToolHandler{}, log.New(os.Stdout, "", 0))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "has no handler") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegistryShouldFailWhenHandlerHasNoSpec(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.yml"), validTool("mock.weather.query"))

	_, err := NewRegistry(filePattern(dir), []ToolHandler{stubHandler{name: "mock.weather.query"}, stubHandler{name: "mock.todo.tasks.list"}}, log.New(os.Stdout, "", 0))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "handlers without tool spec") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type stubHandler struct {
	name string
}

func (s stubHandler) Name() string {
	return s.name
}

func (s stubHandler) Call(_ context.Context, _ map[string]any) (map[string]any, error) {
	return map[string]any{"ok": true}, nil
}

func validTool(name string) string {
	return "type: function\n" +
		"name: " + name + "\n" +
		"description: test\n" +
		"inputSchema:\n" +
		"  type: object\n" +
		"  properties:\n" +
		"    city:\n" +
		"      type: string\n" +
		"  additionalProperties: false\n"
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
}

func filePattern(dir string) string {
	return "file:" + filepath.ToSlash(filepath.Join(dir, "*.yml"))
}

func projectToolsPattern(t *testing.T) string {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	return filepath.Join(root, "tools", "*.yml")
}
