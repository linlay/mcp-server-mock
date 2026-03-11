package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed application.yml
var embeddedApplicationYAML []byte

// ObservabilityConfig controls request/response logging behavior.
type ObservabilityConfig struct {
	LogEnabled        bool `yaml:"logEnabled"`
	LogMaxBodyLength  int  `yaml:"logMaxBodyLength"`
	LogIncludeHeaders bool `yaml:"logIncludeHeaders"`
}

type BashConfig struct {
	WorkingDirectory string   `yaml:"workingDirectory"`
	AllowedRoots     []string `yaml:"allowedRoots"`
	AllowedCommands  []string `yaml:"allowedCommands"`
	TimeoutMs        int      `yaml:"timeoutMs"`
	MaxCommandChars  int      `yaml:"maxCommandChars"`
	MaxOutputChars   int      `yaml:"maxOutputChars"`
}

// Config holds process-level server configuration.
type Config struct {
	ServerPort                int                 `yaml:"serverPort"`
	ToolsSpecLocationPattern  string              `yaml:"toolsSpecLocationPattern"`
	ViewportsDir              string              `yaml:"viewportsDir"`
	ViewportRefreshIntervalMs int                 `yaml:"viewportRefreshIntervalMs"`
	HTTPMaxBodyBytes          int64               `yaml:"httpMaxBodyBytes"`
	Observability             ObservabilityConfig `yaml:"observability"`
	Bash                      BashConfig          `yaml:"bash"`
}

// Load returns configuration from code defaults, embedded YAML, optional external YAML, and env.
func Load() (Config, error) {
	cfg := Config{
		ServerPort:                8080,
		ToolsSpecLocationPattern:  "./tools/*.yml",
		ViewportsDir:              "./viewports",
		ViewportRefreshIntervalMs: 30000,
		HTTPMaxBodyBytes:          1024 * 1024,
		Observability: ObservabilityConfig{
			LogEnabled:        true,
			LogMaxBodyLength:  2000,
			LogIncludeHeaders: false,
		},
		Bash: BashConfig{
			WorkingDirectory: ".",
			AllowedRoots:     []string{".", "./tools", "./viewports", "/tmp"},
			AllowedCommands:  []string{"pwd", "ls", "cat", "head", "tail", "echo", "env", "find"},
			TimeoutMs:        10000,
			MaxCommandChars:  4000,
			MaxOutputChars:   8000,
		},
	}

	if err := mergeYAML("embedded application.yml", embeddedApplicationYAML, &cfg); err != nil {
		return Config{}, err
	}

	configPath := strings.TrimSpace(os.Getenv("CONFIG_PATH"))
	if configPath != "" {
		raw, err := os.ReadFile(configPath)
		if err != nil {
			return Config{}, fmt.Errorf("read config path %q: %w", configPath, err)
		}
		if err := mergeYAML(configPath, raw, &cfg); err != nil {
			return Config{}, err
		}
	}

	applyEnvOverrides(&cfg)
	return cfg, nil
}

func mergeYAML(source string, raw []byte, target *Config) error {
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", source, err)
	}
	return nil
}

func applyEnvOverrides(cfg *Config) {
	cfg.ServerPort = readIntEnv("SERVER_PORT", cfg.ServerPort)
	cfg.ToolsSpecLocationPattern = readStringEnv("MCP_TOOLS_SPEC_LOCATION_PATTERN", cfg.ToolsSpecLocationPattern)
	cfg.ViewportsDir = readStringEnv("MCP_VIEWPORTS_DIR", cfg.ViewportsDir)
	cfg.ViewportRefreshIntervalMs = readIntEnv("MCP_VIEWPORT_REFRESH_INTERVAL_MS", cfg.ViewportRefreshIntervalMs)
	cfg.HTTPMaxBodyBytes = readInt64Env("MCP_HTTP_MAX_BODY_BYTES", cfg.HTTPMaxBodyBytes)
	cfg.Observability.LogEnabled = readBoolEnv("MCP_OBSERVABILITY_LOG_ENABLED", cfg.Observability.LogEnabled)
	cfg.Observability.LogMaxBodyLength = readIntEnv("MCP_OBSERVABILITY_LOG_MAX_BODY_LENGTH", cfg.Observability.LogMaxBodyLength)
	cfg.Observability.LogIncludeHeaders = readBoolEnv("MCP_OBSERVABILITY_LOG_INCLUDE_HEADERS", cfg.Observability.LogIncludeHeaders)
	cfg.Bash.WorkingDirectory = readStringEnv("MCP_BASH_WORKING_DIRECTORY", cfg.Bash.WorkingDirectory)
	cfg.Bash.AllowedRoots = readListEnv("MCP_BASH_ALLOWED_ROOTS", cfg.Bash.AllowedRoots)
	cfg.Bash.AllowedCommands = readListEnv("MCP_BASH_ALLOWED_COMMANDS", cfg.Bash.AllowedCommands)
	cfg.Bash.TimeoutMs = readIntEnv("MCP_BASH_TIMEOUT_MS", cfg.Bash.TimeoutMs)
	cfg.Bash.MaxCommandChars = readIntEnv("MCP_BASH_MAX_COMMAND_CHARS", cfg.Bash.MaxCommandChars)
	cfg.Bash.MaxOutputChars = readIntEnv("MCP_BASH_MAX_OUTPUT_CHARS", cfg.Bash.MaxOutputChars)
}

func readStringEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func readIntEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func readBoolEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func readInt64Env(key string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func readListEnv(key string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	values := make([]string, 0)
	for _, part := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		values = append(values, trimmed)
	}
	if len(values) == 0 {
		return fallback
	}
	return values
}
