package viewport

import (
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRegistryShouldLoadHtmlAndQlc(t *testing.T) {
	dir := t.TempDir()
	writeViewportFile(t, filepath.Join(dir, "show_weather_card.html"), "<div>ok</div>")
	writeViewportFile(t, filepath.Join(dir, "todo_form.qlc"), `{"schema":{"type":"object"}}`)

	registry, err := NewRegistry(dir, 0, map[string][]string{
		"show_weather_card": {"mock.weather.query"},
	}, log.New(os.Stdout, "", 0))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer registry.Close()

	html, ok := registry.Find("show_weather_card")
	if !ok {
		t.Fatal("expected html viewport")
	}
	if html.ViewportType != "html" {
		t.Fatalf("expected html type, got %s", html.ViewportType)
	}
	qlc, ok := registry.Find("todo_form")
	if !ok {
		t.Fatal("expected qlc viewport")
	}
	if qlc.ViewportType != "qlc" {
		t.Fatalf("expected qlc type, got %s", qlc.ViewportType)
	}
}

func TestRegistryShouldFailWhenReferencedViewportMissing(t *testing.T) {
	dir := t.TempDir()
	writeViewportFile(t, filepath.Join(dir, "show_weather_card.html"), "<div>ok</div>")

	_, err := NewRegistry(dir, 0, map[string][]string{
		"show_todo_card": {"mock.todo.tasks.list"},
	}, log.New(os.Stdout, "", 0))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistryRefreshShouldKeepPreviousSnapshotOnFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "show_weather_card.html")
	writeViewportFile(t, path, "<div>ok</div>")

	registry, err := NewRegistry(dir, time.Millisecond, map[string][]string{
		"show_weather_card": {"mock.weather.query"},
	}, log.New(os.Stdout, "", 0))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer registry.Close()

	writeViewportFile(t, path, "")
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove file: %v", err)
	}
	if err := registry.Refresh(); err == nil {
		t.Fatal("expected refresh error")
	}
	entry, ok := registry.Find("show_weather_card")
	if !ok || entry.ViewportType != "html" {
		t.Fatalf("expected previous snapshot to be retained, got %#v", entry)
	}
}

func writeViewportFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write viewport file: %v", err)
	}
}
