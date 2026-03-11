package tools

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mcp-server-mock/internal/config"
	"mcp-server-mock/internal/mcp/spec"
)

func TestRegistryShouldLoadAndListTools(t *testing.T) {
	r, err := NewRegistry(projectToolsPattern(t), BuiltinHandlers(config.BashConfig{}), log.New(os.Stdout, "", 0))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	listed := r.ListTools()
	if len(listed) != 7 {
		t.Fatalf("expected 7 tools, got %d", len(listed))
	}
	byName := map[string]map[string]any{}
	for _, item := range listed {
		name, _ := item["name"].(string)
		byName[name] = item
	}
	if got := byName["mock.weather.query"]["label"]; got != "天气查询" {
		t.Fatalf("expected weather label, got %#v", got)
	}
	if got := byName["mock.sensitive-data.detect"]["label"]; got != "敏感信息检测" {
		t.Fatalf("expected sensitive-data label, got %#v", got)
	}
	bindings := r.ViewportBindings()
	if len(bindings) != 0 {
		t.Fatalf("expected no viewport bindings for default mock tools, got %#v", bindings)
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

func TestSpecToMapShouldIncludeExtendedMetadata(t *testing.T) {
	frontend := spec.SpecToMap(spec.ToolSpec{
		Type:        "function",
		Name:        "mock.frontend.dialog",
		Label:       "确认对话框",
		Description: "frontend",
		InputSchema: map[string]any{"type": "object"},
		ToolType:    "html",
		ViewportKey: "confirm_dialog",
	})
	if got := frontend["label"]; got != "确认对话框" {
		t.Fatalf("expected label 确认对话框, got %#v", got)
	}
	if got := frontend["toolType"]; got != "html" {
		t.Fatalf("expected toolType html, got %#v", got)
	}
	if got := frontend["viewportKey"]; got != "confirm_dialog" {
		t.Fatalf("expected viewportKey confirm_dialog, got %#v", got)
	}

	action := spec.SpecToMap(spec.ToolSpec{
		Type:        "function",
		Name:        "mock.action.launch",
		Description: "action",
		InputSchema: map[string]any{"type": "object"},
		ToolAction:  true,
	})
	if got := action["toolAction"]; got != true {
		t.Fatalf("expected toolAction true, got %#v", got)
	}
}

func TestRegistryShouldReturnExplicitViewportMetadataOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "frontend.yml"), `type: function
name: mock.frontend.dialog
label: 确认对话框
description: frontend
toolType: html
viewportKey: confirm_dialog
inputSchema:
  type: object
`)

	r, err := NewRegistry(filePattern(dir), []ToolHandler{stubHandler{name: "mock.frontend.dialog"}}, log.New(os.Stdout, "", 0))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	listed := r.ListTools()
	if len(listed) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(listed))
	}
	if got := listed[0]["toolType"]; got != "html" {
		t.Fatalf("expected toolType html, got %#v", got)
	}
	if got := listed[0]["viewportKey"]; got != "confirm_dialog" {
		t.Fatalf("expected viewportKey confirm_dialog, got %#v", got)
	}

	bindings := r.ViewportBindings()
	if got := bindings["confirm_dialog"]; len(got) != 1 || got[0] != "mock.frontend.dialog" {
		t.Fatalf("unexpected viewport bindings: %#v", bindings)
	}
}

func TestRegistryShouldFailWhenLabelIsBlank(t *testing.T) {
	cases := []struct {
		name       string
		content    string
		wantErrMsg string
	}{
		{
			name: "empty-string",
			content: `type: function
name: mock.weather.query
label: ""
description: test
inputSchema:
  type: object
`,
			wantErrMsg: "$.label length must be >= 1",
		},
		{
			name: "whitespace-only",
			content: `type: function
name: mock.weather.query
label: "   "
description: test
inputSchema:
  type: object
`,
			wantErrMsg: "label must be a non-empty string",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "a.yml"), tc.content)

			_, err := NewRegistry(filePattern(dir), []ToolHandler{stubHandler{name: "mock.weather.query"}}, log.New(os.Stdout, "", 0))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantErrMsg) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRegistryShouldFailWhenToolModeMetadataIsInvalid(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.yml"), `type: function
name: mock.weather.query
description: test
toolType: html
inputSchema:
  type: object
`)

	_, err := NewRegistry(filePattern(dir), []ToolHandler{stubHandler{name: "mock.weather.query"}}, log.New(os.Stdout, "", 0))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "toolType and viewportKey must be declared together") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegistryShouldFailWhenActionToolAlsoDeclaresFrontendFields(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.yml"), `type: function
name: mock.weather.query
description: test
toolAction: true
toolType: html
viewportKey: confirm_dialog
inputSchema:
  type: object
`)

	_, err := NewRegistry(filePattern(dir), []ToolHandler{stubHandler{name: "mock.weather.query"}}, log.New(os.Stdout, "", 0))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "toolAction=true cannot be combined") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type stubHandler struct {
	name string
}

func (s stubHandler) Name() string {
	return s.name
}

func (s stubHandler) Call(_ context.Context, _ ToolCall) (map[string]any, error) {
	return map[string]any{"ok": true}, nil
}

func validTool(name string) string {
	return "type: function\n" +
		"name: " + name + "\n" +
		"label: 示例工具\n" +
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
