package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mcp-server-mock/internal/config"
)

func TestBashExecutorShouldExecuteAllowedCommand(t *testing.T) {
	root := t.TempDir()
	executor := NewBashExecutor(config.BashConfig{
		WorkingDirectory: root,
		AllowedRoots:     []string{root},
		AllowedCommands:  []string{"pwd"},
		TimeoutMs:        10000,
		MaxCommandChars:  4000,
		MaxOutputChars:   8000,
	})

	result := executor.Execute(context.Background(), "pwd", "", "")
	if result.ExitCode != 0 {
		t.Fatalf("expected exitCode 0, got %d (stderr=%s)", result.ExitCode, result.Stderr)
	}
	if strings.TrimSpace(result.Stdout) != root {
		t.Fatalf("expected stdout %s, got %q", root, result.Stdout)
	}
}

func TestBashExecutorShouldRejectCommandOutsideWhitelist(t *testing.T) {
	root := t.TempDir()
	executor := NewBashExecutor(config.BashConfig{
		WorkingDirectory: root,
		AllowedRoots:     []string{root},
		AllowedCommands:  []string{"pwd"},
	})

	result := executor.Execute(context.Background(), "env", "", "")
	if result.ExitCode != -1 {
		t.Fatalf("expected exitCode -1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Stderr, "Command not allowed") {
		t.Fatalf("unexpected stderr: %s", result.Stderr)
	}
}

func TestBashExecutorShouldRespectMetaWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "nested")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	executor := NewBashExecutor(config.BashConfig{
		WorkingDirectory: root,
		AllowedRoots:     []string{root},
		AllowedCommands:  []string{"pwd"},
	})

	result := executor.Execute(context.Background(), "pwd", "./nested", "")
	if result.ExitCode != 0 {
		t.Fatalf("expected exitCode 0, got %d (stderr=%s)", result.ExitCode, result.Stderr)
	}
	if strings.TrimSpace(result.Stdout) != child {
		t.Fatalf("expected stdout %s, got %q", child, result.Stdout)
	}
}

func TestBashExecutorShouldRejectMetaWorkingDirectoryOutsideAllowedRoots(t *testing.T) {
	root := t.TempDir()
	executor := NewBashExecutor(config.BashConfig{
		WorkingDirectory: root,
		AllowedRoots:     []string{root},
		AllowedCommands:  []string{"pwd"},
	})

	result := executor.Execute(context.Background(), "pwd", "../outside", "")
	if result.ExitCode != -1 {
		t.Fatalf("expected exitCode -1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Stderr, "Working directory not allowed") {
		t.Fatalf("unexpected stderr: %s", result.Stderr)
	}
}

func TestBashExecutorShouldInjectUserIDIntoEnvironment(t *testing.T) {
	root := t.TempDir()
	executor := NewBashExecutor(config.BashConfig{
		WorkingDirectory: root,
		AllowedRoots:     []string{root},
		AllowedCommands:  []string{"env"},
	})

	result := executor.Execute(context.Background(), "env", "", "rena-user-1")
	if result.ExitCode != 0 {
		t.Fatalf("expected exitCode 0, got %d (stderr=%s)", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "MCP_USER_ID=rena-user-1") {
		t.Fatalf("expected env output to include MCP_USER_ID, got %q", result.Stdout)
	}
	if result.UserID != "rena-user-1" {
		t.Fatalf("expected userId rena-user-1, got %q", result.UserID)
	}
}
