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

// Config holds process-level server configuration.
type Config struct {
	ServerPort               int                 `yaml:"serverPort"`
	ToolsSpecLocationPattern string              `yaml:"toolsSpecLocationPattern"`
	HTTPMaxBodyBytes         int64               `yaml:"httpMaxBodyBytes"`
	Observability            ObservabilityConfig `yaml:"observability"`
}

// Load returns configuration from code defaults, embedded YAML, optional external YAML, and env.
func Load() (Config, error) {
	cfg := Config{
		ServerPort:               8080,
		ToolsSpecLocationPattern: "./tools/*.yml",
		HTTPMaxBodyBytes:         1024 * 1024,
		Observability: ObservabilityConfig{
			LogEnabled:        true,
			LogMaxBodyLength:  2000,
			LogIncludeHeaders: false,
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
	cfg.HTTPMaxBodyBytes = readInt64Env("MCP_HTTP_MAX_BODY_BYTES", cfg.HTTPMaxBodyBytes)
	cfg.Observability.LogEnabled = readBoolEnv("MCP_OBSERVABILITY_LOG_ENABLED", cfg.Observability.LogEnabled)
	cfg.Observability.LogMaxBodyLength = readIntEnv("MCP_OBSERVABILITY_LOG_MAX_BODY_LENGTH", cfg.Observability.LogMaxBodyLength)
	cfg.Observability.LogIncludeHeaders = readBoolEnv("MCP_OBSERVABILITY_LOG_INCLUDE_HEADERS", cfg.Observability.LogIncludeHeaders)
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
