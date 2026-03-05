package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"mcp-server-mock-go/internal/config"
	"mcp-server-mock-go/internal/mcp"
	"mcp-server-mock-go/internal/observability"
)

func main() {
	cfg := config.Load()
	std := log.Default()

	repository := mcp.NewToolSpecRepository(cfg.ToolsSpecLocationPattern, std)
	sanitizer := observability.NewLogSanitizer(cfg.Observability.LogMaxBodyLength)
	obsLogger := observability.NewLogger(std, cfg.Observability, sanitizer)
	toolService := mcp.NewToolService(repository, obsLogger)
	controller := mcp.NewController(toolService, obsLogger)

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
