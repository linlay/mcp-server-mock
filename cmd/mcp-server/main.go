package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"mcp-server-mock/internal/config"
	"mcp-server-mock/internal/mcp/tools"
	"mcp-server-mock/internal/mcp/transport"
	"mcp-server-mock/internal/observability"
)

func main() {
	cfg := config.Load()
	std := log.Default()

	sanitizer := observability.NewLogSanitizer(cfg.Observability.LogMaxBodyLength)
	obsLogger := observability.NewLogger(std, cfg.Observability, sanitizer)
	registry, err := tools.NewRegistry(cfg.ToolsSpecLocationPattern, tools.BuiltinHandlers(), std)
	if err != nil {
		std.Fatalf("failed to initialize tool registry: %v", err)
	}
	controller := transport.NewController(registry, obsLogger, cfg.HTTPMaxBodyBytes)

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
