package mcp

import (
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestShouldReturnEmptyWhenPatternHasNoFiles(t *testing.T) {
	dir := t.TempDir()
	repository := NewToolSpecRepository(filePattern(dir), log.New(os.Stdout, "", 0))
	if len(repository.ListTools()) != 0 {
		t.Fatalf("expected empty tools when no yaml files")
	}
}

func TestShouldReturnEmptyWhenYAMLInvalid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "invalid.yml"), []byte("name: mock.weather.query\ninputSchema: ["), 0o644); err != nil {
		t.Fatalf("failed to write invalid yaml: %v", err)
	}
	repository := NewToolSpecRepository(filePattern(dir), log.New(os.Stdout, "", 0))
	if len(repository.ListTools()) != 0 {
		t.Fatalf("expected empty tools when yaml is invalid")
	}
}

func TestShouldReturnEmptyWhenToolNamesDuplicated(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.yml"), []byte(validTool("mock.weather.query", "desc-a")), 0o644); err != nil {
		t.Fatalf("failed to write a.yml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.yml"), []byte(validTool("mock.weather.query", "desc-b")), 0o644); err != nil {
		t.Fatalf("failed to write b.yml: %v", err)
	}
	repository := NewToolSpecRepository(filePattern(dir), log.New(os.Stdout, "", 0))
	if len(repository.ListTools()) != 0 {
		t.Fatalf("expected empty tools when names are duplicated")
	}
}

func filePattern(dir string) string {
	return "file:" + filepath.ToSlash(filepath.Join(dir, "*.yml"))
}

func validTool(name, description string) string {
	return "type: function\n" +
		"name: " + name + "\n" +
		"description: \"" + description + "\"\n" +
		"inputSchema:\n" +
		"  type: object\n" +
		"  properties:\n" +
		"    value:\n" +
		"      type: string\n" +
		"  additionalProperties: false\n"
}
