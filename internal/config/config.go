package config

import (
	"os"
	"strconv"
	"strings"
)

// ObservabilityConfig controls request/response logging behavior.
type ObservabilityConfig struct {
	LogEnabled        bool
	LogMaxBodyLength  int
	LogIncludeHeaders bool
}

// Config holds process-level server configuration.
type Config struct {
	ServerPort               int
	ToolsSpecLocationPattern string
	Observability            ObservabilityConfig
}

// Load returns configuration from env with safe defaults.
func Load() Config {
	return Config{
		ServerPort:               readIntEnv("SERVER_PORT", 19080),
		ToolsSpecLocationPattern: readStringEnv("MCP_TOOLS_SPEC_LOCATION_PATTERN", "./tools/*.yml"),
		Observability: ObservabilityConfig{
			LogEnabled:        readBoolEnv("MCP_OBSERVABILITY_LOG_ENABLED", true),
			LogMaxBodyLength:  readIntEnv("MCP_OBSERVABILITY_LOG_MAX_BODY_LENGTH", 2000),
			LogIncludeHeaders: readBoolEnv("MCP_OBSERVABILITY_LOG_INCLUDE_HEADERS", false),
		},
	}
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
