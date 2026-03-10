package viewport

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	ViewportKey  string
	ViewportType string
	Payload      any
}

type Summary struct {
	ViewportKey  string   `json:"viewportKey"`
	ViewportType string   `json:"viewportType"`
	ToolNames    []string `json:"toolNames"`
}

type Registry struct {
	dir                 string
	refreshInterval     time.Duration
	toolNamesByViewport map[string][]string
	logger              *log.Logger

	mu      sync.RWMutex
	entries map[string]Entry
	stopCh  chan struct{}
}

func NewRegistry(dir string, refreshInterval time.Duration, toolNamesByViewport map[string][]string, logger *log.Logger) (*Registry, error) {
	if strings.TrimSpace(dir) == "" {
		dir = "./viewports"
	}
	if logger == nil {
		logger = log.Default()
	}
	registry := &Registry{
		dir:                 dir,
		refreshInterval:     refreshInterval,
		toolNamesByViewport: normalizeToolNamesByViewport(toolNamesByViewport),
		logger:              logger,
		entries:             map[string]Entry{},
		stopCh:              make(chan struct{}),
	}
	if err := registry.Refresh(); err != nil {
		return nil, err
	}
	if refreshInterval > 0 {
		go registry.refreshLoop()
	}
	return registry, nil
}

func (r *Registry) Close() {
	if r == nil || r.stopCh == nil {
		return
	}
	select {
	case <-r.stopCh:
		return
	default:
		close(r.stopCh)
	}
}

func (r *Registry) Refresh() error {
	if r == nil {
		return nil
	}
	entries, err := r.loadEntries()
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.entries = entries
	r.mu.Unlock()
	return nil
}

func (r *Registry) Find(viewportKey string) (Entry, bool) {
	if r == nil {
		return Entry{}, false
	}
	normalized := normalizeKey(viewportKey)
	if normalized == "" {
		return Entry{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[normalized]
	return entry, ok
}

func (r *Registry) ListSummaries() []Summary {
	if r == nil {
		return []Summary{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, 0, len(r.entries))
	for key := range r.entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	summaries := make([]Summary, 0, len(keys))
	for _, key := range keys {
		entry := r.entries[key]
		summaries = append(summaries, Summary{
			ViewportKey:  entry.ViewportKey,
			ViewportType: entry.ViewportType,
			ToolNames:    append([]string(nil), r.toolNamesByViewport[key]...),
		})
	}
	return summaries
}

func (r *Registry) refreshLoop() {
	ticker := time.NewTicker(r.refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := r.Refresh(); err != nil {
				r.logger.Printf("event=viewport.registry.refresh_failed dir=%s err=%v", r.dir, err)
			}
		case <-r.stopCh:
			return
		}
	}
}

func (r *Registry) loadEntries() (map[string]Entry, error) {
	files, err := os.ReadDir(r.dir)
	if err != nil {
		return nil, fmt.Errorf("read viewports dir %s: %w", r.dir, err)
	}
	entries := make(map[string]Entry)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		entry, ok, err := loadEntry(filepath.Join(r.dir, file.Name()))
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if _, exists := entries[entry.ViewportKey]; exists {
			return nil, fmt.Errorf("duplicate viewport key: %s", entry.ViewportKey)
		}
		entries[entry.ViewportKey] = entry
	}
	for viewportKey, toolNames := range r.toolNamesByViewport {
		if _, ok := entries[viewportKey]; ok {
			continue
		}
		return nil, fmt.Errorf("missing viewport file for key %s referenced by tools %s", viewportKey, strings.Join(toolNames, ", "))
	}
	return entries, nil
}

func loadEntry(path string) (Entry, bool, error) {
	name := filepath.Base(path)
	suffix := strings.ToLower(filepath.Ext(name))
	if suffix != ".html" && suffix != ".qlc" {
		return Entry{}, false, nil
	}
	key := normalizeKey(strings.TrimSuffix(name, filepath.Ext(name)))
	if key == "" {
		return Entry{}, false, fmt.Errorf("empty viewport key for file %s", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Entry{}, false, fmt.Errorf("read viewport file %s: %w", path, err)
	}
	switch suffix {
	case ".html":
		return Entry{
			ViewportKey:  key,
			ViewportType: "html",
			Payload:      string(raw),
		}, true, nil
	case ".qlc":
		var payload any
		if err := json.Unmarshal(raw, &payload); err != nil {
			return Entry{}, false, fmt.Errorf("parse qlc viewport %s: %w", path, err)
		}
		return Entry{
			ViewportKey:  key,
			ViewportType: "qlc",
			Payload:      payload,
		}, true, nil
	default:
		return Entry{}, false, nil
	}
}

func normalizeKey(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func normalizeToolNamesByViewport(raw map[string][]string) map[string][]string {
	if len(raw) == 0 {
		return map[string][]string{}
	}
	normalized := make(map[string][]string, len(raw))
	for key, names := range raw {
		viewportKey := normalizeKey(key)
		if viewportKey == "" {
			continue
		}
		values := make([]string, 0, len(names))
		for _, name := range names {
			trimmed := strings.TrimSpace(name)
			if trimmed != "" {
				values = append(values, trimmed)
			}
		}
		sort.Strings(values)
		normalized[viewportKey] = values
	}
	return normalized
}
