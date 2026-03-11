package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"mcp-server-mock/internal/config"
)

const (
	defaultBashTimeoutMs     = 10000
	defaultBashMaxCommandLen = 4000
	defaultBashMaxOutputLen  = 8000
)

var pathCheckedCommands = map[string]struct{}{
	"cat":  {},
	"find": {},
	"head": {},
	"ls":   {},
	"tail": {},
}

type BashExecutor struct {
	workingDirectory string
	allowedRoots     []string
	allowedCommands  map[string]struct{}
	timeout          time.Duration
	maxCommandChars  int
	maxOutputChars   int
}

type BashResult struct {
	ExitCode         int    `json:"exitCode"`
	WorkingDirectory string `json:"workingDirectory"`
	UserID           string `json:"userId"`
	Stdout           string `json:"stdout"`
	Stderr           string `json:"stderr"`
	Text             string `json:"text"`
}

func NewBashExecutor(cfg config.BashConfig) *BashExecutor {
	workingDirectory := normalizeAbsolutePath(mustGetwd(), cfg.WorkingDirectory)
	allowedRoots := normalizeAllowedRoots(workingDirectory, cfg.AllowedRoots)
	if len(allowedRoots) == 0 {
		allowedRoots = []string{workingDirectory}
	}

	timeoutMs := cfg.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = defaultBashTimeoutMs
	}
	maxCommandChars := cfg.MaxCommandChars
	if maxCommandChars <= 0 {
		maxCommandChars = defaultBashMaxCommandLen
	}
	maxOutputChars := cfg.MaxOutputChars
	if maxOutputChars <= 0 {
		maxOutputChars = defaultBashMaxOutputLen
	}

	return &BashExecutor{
		workingDirectory: workingDirectory,
		allowedRoots:     allowedRoots,
		allowedCommands:  normalizeCommandSet(cfg.AllowedCommands),
		timeout:          time.Duration(timeoutMs) * time.Millisecond,
		maxCommandChars:  maxCommandChars,
		maxOutputChars:   maxOutputChars,
	}
}

func (e *BashExecutor) Execute(ctx context.Context, command string, workDirectory string, userID string) BashResult {
	if strings.TrimSpace(command) == "" {
		return e.errorResult("Missing argument: command", "", userID)
	}
	if len(command) > e.maxCommandChars {
		return e.errorResult(
			fmt.Sprintf("Command is too long. Maximum length is %d characters.", e.maxCommandChars),
			"",
			userID,
		)
	}

	tokens, err := tokenizeCommand(command)
	if err != nil {
		return e.errorResult(err.Error(), "", userID)
	}
	if len(tokens) == 0 {
		return e.errorResult("Cannot parse command", "", userID)
	}
	if len(e.allowedCommands) == 0 {
		return e.errorResult("Bash command whitelist is empty. Configure MCP_BASH_ALLOWED_COMMANDS", "", userID)
	}
	if _, ok := e.allowedCommands[tokens[0]]; !ok {
		return e.errorResult("Command not allowed: "+tokens[0], "", userID)
	}

	resolvedDir, err := e.resolveWorkingDirectory(workDirectory)
	if err != nil {
		return e.errorResult(err.Error(), resolvedDir, userID)
	}
	if err := e.validatePathArgs(tokens, resolvedDir); err != nil {
		return e.errorResult(err.Error(), resolvedDir, userID)
	}

	runCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	commandArgs := append([]string{"-lc", `exec "$@"`, "bash"}, tokens...)
	cmd := exec.CommandContext(runCtx, "bash", commandArgs...)
	cmd.Dir = resolvedDir
	if strings.TrimSpace(userID) != "" {
		cmd.Env = append(os.Environ(), "MCP_USER_ID="+strings.TrimSpace(userID))
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return e.errorResult(err.Error(), resolvedDir, userID)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return e.errorResult(err.Error(), resolvedDir, userID)
	}
	if err := cmd.Start(); err != nil {
		return e.errorResult(err.Error(), resolvedDir, userID)
	}

	stdout := readCapped(stdoutPipe, e.maxOutputChars)
	stderr := readCapped(stderrPipe, e.maxOutputChars)
	waitErr := cmd.Wait()
	if runCtx.Err() == context.DeadlineExceeded {
		if strings.TrimSpace(stderr) != "" {
			stderr += "\n"
		}
		stderr += "Command timed out"
		return e.result(-1, resolvedDir, userID, stdout, stderr)
	}

	exitCode := 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
			if strings.TrimSpace(stderr) == "" {
				stderr = waitErr.Error()
			}
		}
	}
	return e.result(exitCode, resolvedDir, userID, stdout, stderr)
}

func (e *BashExecutor) resolveWorkingDirectory(override string) (string, error) {
	base := e.workingDirectory
	if strings.TrimSpace(override) != "" {
		base = normalizeAbsolutePath(e.workingDirectory, override)
	}
	if !isAllowedPath(base, e.allowedRoots) {
		return base, fmt.Errorf("Working directory not allowed outside authorized directories: %s", strings.TrimSpace(override))
	}
	return base, nil
}

func (e *BashExecutor) validatePathArgs(tokens []string, workDirectory string) error {
	if _, ok := pathCheckedCommands[tokens[0]]; !ok {
		return nil
	}
	for _, token := range tokens[1:] {
		if strings.TrimSpace(token) == "" || strings.HasPrefix(token, "-") {
			continue
		}
		resolved := normalizeAbsolutePath(workDirectory, token)
		if !isAllowedPath(resolved, e.allowedRoots) {
			return fmt.Errorf("Path not allowed outside authorized directories: %s", token)
		}
	}
	return nil
}

func (e *BashExecutor) result(exitCode int, workDirectory string, userID string, stdout string, stderr string) BashResult {
	text := strings.TrimSpace(stdout)
	if text == "" {
		text = strings.TrimSpace(stderr)
	}
	return BashResult{
		ExitCode:         exitCode,
		WorkingDirectory: workDirectory,
		UserID:           strings.TrimSpace(userID),
		Stdout:           stdout,
		Stderr:           stderr,
		Text:             text,
	}
}

func (e *BashExecutor) errorResult(message string, workDirectory string, userID string) BashResult {
	return e.result(-1, workDirectory, userID, "", message)
}

func tokenizeCommand(raw string) ([]string, error) {
	tokens := make([]string, 0)
	current := strings.Builder{}
	inSingle := false
	inDouble := false
	escaped := false

	for _, ch := range raw {
		if escaped {
			current.WriteRune(ch)
			escaped = false
			continue
		}

		switch {
		case inSingle:
			if ch == '\'' {
				inSingle = false
			} else {
				current.WriteRune(ch)
			}
		case inDouble:
			if ch == '"' {
				inDouble = false
			} else if ch == '\\' {
				escaped = true
			} else {
				current.WriteRune(ch)
			}
		default:
			switch ch {
			case '\\':
				escaped = true
			case '\'':
				inSingle = true
			case '"':
				inDouble = true
			case '\n', '\r', ';', '|', '&', '<', '>', '`':
				return nil, fmt.Errorf("Unsupported syntax for bash: %c", ch)
			case ' ', '\t':
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
			default:
				current.WriteRune(ch)
			}
		}
	}

	if escaped || inSingle || inDouble {
		return nil, fmt.Errorf("Cannot parse command")
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens, nil
}

func readCapped(r io.Reader, maxChars int) string {
	if r == nil {
		return ""
	}
	if maxChars <= 0 {
		maxChars = defaultBashMaxOutputLen
	}
	buf := make([]byte, 2048)
	output := bytes.Buffer{}
	truncated := false
	for {
		n, err := r.Read(buf)
		if n > 0 {
			remaining := maxChars - output.Len()
			if remaining > 0 {
				toWrite := n
				if toWrite > remaining {
					toWrite = remaining
					truncated = true
				}
				_, _ = output.Write(buf[:toWrite])
			} else {
				truncated = true
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}
	}
	text := output.String()
	if truncated {
		return text + "\n[TRUNCATED]"
	}
	return text
}

func normalizeCommandSet(values []string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				out[trimmed] = struct{}{}
			}
		}
	}
	return out
}

func normalizeAllowedRoots(base string, values []string) []string {
	if len(values) == 0 {
		return []string{base}
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(values))
	for _, value := range values {
		resolved := normalizeAbsolutePath(base, value)
		if resolved == "" {
			continue
		}
		if _, ok := seen[resolved]; ok {
			continue
		}
		seen[resolved] = struct{}{}
		out = append(out, resolved)
	}
	return out
}

func normalizeAbsolutePath(base string, value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return filepath.Clean(base)
	}
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed)
	}
	return filepath.Clean(filepath.Join(base, trimmed))
}

func isAllowedPath(path string, allowedRoots []string) bool {
	cleanPath := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		cleanPath = strings.ToLower(cleanPath)
	}
	for _, root := range allowedRoots {
		cleanRoot := filepath.Clean(root)
		if runtime.GOOS == "windows" {
			cleanRoot = strings.ToLower(cleanRoot)
		}
		if cleanPath == cleanRoot || strings.HasPrefix(cleanPath, cleanRoot+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
