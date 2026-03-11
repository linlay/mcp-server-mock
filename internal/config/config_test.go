package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadShouldUseEmbeddedDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.ServerPort != 8080 {
		t.Fatalf("expected embedded server port 8080, got %d", cfg.ServerPort)
	}
	if cfg.ToolsSpecLocationPattern != "./tools/*.yml" {
		t.Fatalf("unexpected tools pattern: %s", cfg.ToolsSpecLocationPattern)
	}
	if cfg.ViewportsDir != "./viewports" {
		t.Fatalf("unexpected viewports dir: %s", cfg.ViewportsDir)
	}
	if cfg.ViewportRefreshIntervalMs != 30000 {
		t.Fatalf("unexpected viewport refresh interval: %d", cfg.ViewportRefreshIntervalMs)
	}
	if cfg.HTTPMaxBodyBytes != 1048576 {
		t.Fatalf("unexpected http max body bytes: %d", cfg.HTTPMaxBodyBytes)
	}
	if !cfg.Observability.LogEnabled {
		t.Fatal("expected log enabled by default")
	}
	if cfg.Observability.LogMaxBodyLength != 2000 {
		t.Fatalf("unexpected log max body length: %d", cfg.Observability.LogMaxBodyLength)
	}
	if cfg.Observability.LogIncludeHeaders {
		t.Fatal("expected log include headers false by default")
	}
	if cfg.Bash.WorkingDirectory != "." {
		t.Fatalf("unexpected bash working directory: %s", cfg.Bash.WorkingDirectory)
	}
	if len(cfg.Bash.AllowedCommands) == 0 {
		t.Fatal("expected bash allowed commands by default")
	}
	if cfg.Bash.TimeoutMs != 10000 {
		t.Fatalf("unexpected bash timeout: %d", cfg.Bash.TimeoutMs)
	}
}

func TestLoadShouldOverlayExternalYAMLThenEnv(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.dev.yml")
	writeConfigFile(t, configPath, `serverPort: 9090
viewportsDir: ./custom-viewports
viewportRefreshIntervalMs: 12345
httpMaxBodyBytes: 2048
observability:
  logEnabled: false
  logMaxBodyLength: 512
bash:
  workingDirectory: ./custom-bash
  allowedRoots:
    - ./custom-bash
    - /tmp
  allowedCommands:
    - pwd
    - env
  timeoutMs: 5000
  maxCommandChars: 1000
  maxOutputChars: 2048
`)

	t.Setenv("CONFIG_PATH", configPath)
	t.Setenv("SERVER_PORT", "12000")
	t.Setenv("MCP_VIEWPORTS_DIR", "./env-viewports")
	t.Setenv("MCP_OBSERVABILITY_LOG_INCLUDE_HEADERS", "true")
	t.Setenv("MCP_BASH_ALLOWED_COMMANDS", "pwd,env,echo")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.ServerPort != 12000 {
		t.Fatalf("expected env server port 12000, got %d", cfg.ServerPort)
	}
	if cfg.HTTPMaxBodyBytes != 2048 {
		t.Fatalf("expected yaml http max body bytes 2048, got %d", cfg.HTTPMaxBodyBytes)
	}
	if cfg.ViewportsDir != "./env-viewports" {
		t.Fatalf("expected env viewport dir override, got %s", cfg.ViewportsDir)
	}
	if cfg.ViewportRefreshIntervalMs != 12345 {
		t.Fatalf("expected yaml viewport refresh interval 12345, got %d", cfg.ViewportRefreshIntervalMs)
	}
	if cfg.Observability.LogEnabled {
		t.Fatal("expected yaml to disable logging")
	}
	if cfg.Observability.LogMaxBodyLength != 512 {
		t.Fatalf("expected yaml max body 512, got %d", cfg.Observability.LogMaxBodyLength)
	}
	if !cfg.Observability.LogIncludeHeaders {
		t.Fatal("expected env override to enable header logging")
	}
	if cfg.ToolsSpecLocationPattern != "./tools/*.yml" {
		t.Fatalf("expected embedded default tools pattern, got %s", cfg.ToolsSpecLocationPattern)
	}
	if cfg.Bash.WorkingDirectory != "./custom-bash" {
		t.Fatalf("expected yaml bash working directory, got %s", cfg.Bash.WorkingDirectory)
	}
	if got := strings.Join(cfg.Bash.AllowedCommands, ","); got != "pwd,env,echo" {
		t.Fatalf("expected env bash allowed commands override, got %s", got)
	}
	if cfg.Bash.TimeoutMs != 5000 {
		t.Fatalf("expected yaml bash timeout 5000, got %d", cfg.Bash.TimeoutMs)
	}
}

func TestLoadShouldFailForUnknownExternalField(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.dev.yml")
	writeConfigFile(t, configPath, "unknownField: true\n")
	t.Setenv("CONFIG_PATH", configPath)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "field unknownField not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadShouldFailForMissingExternalConfigFile(t *testing.T) {
	t.Setenv("CONFIG_PATH", filepath.Join(t.TempDir(), "missing.yml"))

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "read config path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeConfigFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
}
