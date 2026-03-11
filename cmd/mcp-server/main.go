package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"mcp-server-mock/internal/config"
	"mcp-server-mock/internal/mcp/tools"
	"mcp-server-mock/internal/mcp/transport"
	"mcp-server-mock/internal/observability"
	"mcp-server-mock/internal/viewport"
)

func main() {
	std := log.Default()
	cfg, err := config.Load()
	if err != nil {
		std.Fatalf("failed to load config: %v", err)
	}

	sanitizer := observability.NewLogSanitizer(cfg.Observability.LogMaxBodyLength)
	obsLogger := observability.NewLogger(std, cfg.Observability, sanitizer)
	registry, err := tools.NewRegistry(cfg.ToolsSpecLocationPattern, tools.BuiltinHandlers(cfg.Bash), std)
	if err != nil {
		std.Fatalf("failed to initialize tool registry: %v", err)
	}
	viewportRegistry, err := viewport.NewRegistry(
		cfg.ViewportsDir,
		time.Duration(cfg.ViewportRefreshIntervalMs)*time.Millisecond,
		registry.ViewportBindings(),
		std,
	)
	if err != nil {
		std.Fatalf("failed to initialize viewport registry: %v", err)
	}
	defer viewportRegistry.Close()
	controller := transport.NewController(registry, viewportRegistry, obsLogger, cfg.HTTPMaxBodyBytes)

	mux := http.NewServeMux()
	mux.Handle("/mcp", controller)

	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	std.Printf("event=server.start port=%d", cfg.ServerPort)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		std.Fatalf("server failed: %v", err)
	}
}
