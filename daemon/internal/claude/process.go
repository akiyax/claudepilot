package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ProcessOptions contains all parameters for starting a Claude Code process.
type ProcessOptions struct {
	WorkDir        string            // working directory (default: ~)
	Model          string            // model to use
	SessionID      string            // empty=new, "continue"=continue, otherwise resume
	AgentName      string            // agent persona name
	Effort         string            // reasoning effort: low/medium/high/max
	PermissionMode string            // permission mode
	Provider       string            // provider name (for env var injection)
	Env            []string          // extra environment variables
}

// ProcessState represents the current state of a Claude process.
type ProcessState int

const (
	StateStarting ProcessState = iota
	StateRunning
	StateStopping
	StateStopped
)

// ClaudeProcess manages a single Claude Code CLI process.
type ClaudeProcess struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	stderr   bytes.Buffer
	events   chan ClaudeEvent
	state    atomic.Int32 // ProcessState
	sessionID string
	mu       sync.Mutex // protects stdin writes
	pid      int

	// Callbacks
	onEvent func(event ClaudeEvent)
}

// NewProcess starts a new Claude Code CLI process with the given options.
func NewProcess(ctx context.Context, opts ProcessOptions) (*ClaudeProcess, error) {
	args := buildArgs(opts)

	cmd := exec.CommandContext(ctx, "claude", args...)
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	// Filter CLAUDECODE env vars to prevent nested session detection
	env := filterEnv(os.Environ(), "CLAUDECODE")
	if len(opts.Env) > 0 {
		env = mergeEnv(env, opts.Env)
	}
	cmd.Env = env

	// Create pipes
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	p := &ClaudeProcess{
		cmd:     cmd,
		stdin:   stdinPipe,
		stdout:  stdoutPipe,
		stderr:  stderrBuf,
		events:  make(chan ClaudeEvent, 256),
	}

	p.state.Store(int32(StateStarting))

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	p.pid = cmd.Process.Pid
	p.state.Store(int32(StateRunning))

	slog.Info("Claude process started",
		"pid", p.pid,
		"workDir", opts.WorkDir,
		"sessionID", opts.SessionID,
		"model", opts.Model,
		"agent", opts.AgentName,
	)

	// Start reading stdout in background
	go p.readLoop(stdoutPipe)

	return p, nil
}

// Events returns the channel that emits Claude events.
func (p *ClaudeProcess) Events() <-chan ClaudeEvent {
	return p.events
}

// Alive returns true if the process is still running.
func (p *ClaudeProcess) Alive() bool {
	return p.state.Load() == int32(StateRunning)
}

// PID returns the process ID.
func (p *ClaudeProcess) PID() int {
	return p.pid
}

// SessionID returns the session ID (set after receiving system init event).
func (p *ClaudeProcess) SessionID() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sessionID
}

// Send writes a user message to the process's stdin.
func (p *ClaudeProcess) Send(prompt string, images []ImageAttachment, files []FileAttachment) error {
	if !p.Alive() {
		return fmt.Errorf("process is not running")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if len(images) == 0 && len(files) == 0 {
		return p.writeJSON(UserInput{
			Type:    "user",
			Message: UserMessage{Role: "user", Content: prompt},
		})
	}

	// Build multipart content
	var parts []map[string]any
	for _, img := range images {
		parts = append(parts, map[string]any{
			"type":   "image",
			"source": map[string]any{
				"type":       "base64",
				"media_type": img.MimeType,
				"data":       img.Data,
			},
		})
	}
	for _, f := range files {
		parts = append(parts, map[string]any{
			"type": "text",
			"text": fmt.Sprintf("File: %s\n%s", f.FileName, string(f.Data)),
		})
	}
	parts = append(parts, map[string]any{
		"type": "text",
		"text": prompt,
	})

	return p.writeJSON(UserInput{
		Type:    "user",
		Message: UserMessage{Role: "user", Content: parts},
	})
}

// RespondPermission sends a permission decision back to Claude Code.
func (p *ClaudeProcess) RespondPermission(requestID string, result PermissionResult) error {
	if !p.Alive() {
		return fmt.Errorf("process is not running")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	var permResponse any
	if result.Behavior == "allow" {
		permResponse = PermissionAllow{
			Behavior:     "allow",
			UpdatedInput: result.UpdatedInput,
		}
	} else {
		permResponse = PermissionDeny{
			Behavior: "deny",
			Message:  result.Message,
		}
	}

	return p.writeJSON(ControlResponse{
		Type: "control_response",
		Response: ControlResponseBody{
			Subtype:   "success",
			RequestID: requestID,
			Response:  permResponse,
		},
	})
}

// Close shuts down the process gracefully.
// Phase 1: close stdin (let Claude finish)
// Phase 2: SIGTERM (after 5s)
// Phase 3: SIGKILL (after another 5s)
func (p *ClaudeProcess) Close() error {
	p.state.Store(int32(StateStopping))

	// Phase 1: close stdin
	if p.stdin != nil {
		p.stdin.Close()
	}

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()

	select {
	case err := <-done:
		p.state.Store(int32(StateStopped))
		close(p.events)
		slog.Info("Claude process exited", "pid", p.pid, "err", err)
		return err
	case <-time.After(5 * time.Second):
		// Phase 2: SIGTERM
		slog.Warn("Claude process did not exit, sending SIGTERM", "pid", p.pid)
		p.cmd.Process.Signal(os.Interrupt)
	}

	select {
	case err := <-done:
		p.state.Store(int32(StateStopped))
		close(p.events)
		return err
	case <-time.After(5 * time.Second):
		// Phase 3: SIGKILL
		slog.Warn("Claude process did not exit, sending SIGKILL", "pid", p.pid)
		p.cmd.Process.Kill()
		err := <-done
		p.state.Store(int32(StateStopped))
		close(p.events)
		return err
	}
}

// readLoop reads stdout line by line, parses JSON events.
func (p *ClaudeProcess) readLoop(stdout io.ReadCloser) {
	defer func() {
		if p.state.Load() == int32(StateRunning) {
			p.state.Store(int32(StateStopped))
		}
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 10MB max line

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			slog.Debug("Failed to parse stdout line", "line", line, "err", err)
			continue
		}

		eventType, _ := raw["type"].(string)

		// Extract session_id from system init
		if eventType == "system" {
			if sid, ok := raw["session_id"].(string); ok && sid != "" {
				p.mu.Lock()
				p.sessionID = sid
				p.mu.Unlock()
			}
		}

		event := ClaudeEvent{
			Type: eventType,
			Raw:  raw,
		}

		select {
		case p.events <- event:
		default:
			slog.Warn("Event channel full, dropping event", "type", eventType)
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Stdout scanner error", "pid", p.pid, "err", err)
	}

	slog.Info("Claude stdout closed", "pid", p.pid)
}

// writeJSON marshals and writes a JSON object to stdin with a newline.
func (p *ClaudeProcess) writeJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	data = append(data, '\n')
	_, err = p.stdin.Write(data)
	return err
}

// ─── Helper functions ────────────────────────────────────

func buildArgs(opts ProcessOptions) []string {
	args := []string{
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--permission-prompt-tool", "stdio",
		"--verbose",
	}

	if opts.PermissionMode != "" && opts.PermissionMode != "default" {
		args = append(args, "--permission-mode", opts.PermissionMode)
	}

	switch opts.SessionID {
	case "":
		// Fresh session
	case "continue":
		args = append(args, "--continue", "--fork-session")
	default:
		args = append(args, "--resume", opts.SessionID)
	}

	if opts.AgentName != "" {
		args = append(args, "--agent", opts.AgentName)
	}

	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}

	if opts.Effort != "" {
		args = append(args, "--effort", opts.Effort)
	}

	return args
}

// filterEnv removes environment variables that start with the given prefix.
func filterEnv(env []string, prefix string) []string {
	var filtered []string
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if !strings.HasPrefix(parts[0], prefix) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// mergeEnv merges extra env vars into base, overriding existing keys.
func mergeEnv(base, extra []string) []string {
	envMap := make(map[string]string)
	for _, e := range base {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	for _, e := range extra {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	result := make([]string, 0, len(envMap))
	for k, v := range envMap {
		result = append(result, k+"="+v)
	}
	return result
}

// DetectCLIVersion runs `claude --version` and returns the output.
func DetectCLIVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "claude", "--version").Output()
	if err != nil {
		slog.Warn("Failed to detect Claude CLI version", "err", err)
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// DetectOrphans scans ~/.claude/sessions/ for active PIDs and returns
// sessions whose processes are still alive but have no parent managing them.
func DetectOrphans() ([]OrphanSession, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	sessionsDir := home + "/.claude/sessions"
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var orphans []OrphanSession
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(sessionsDir + "/" + entry.Name())
		if err != nil {
			continue
		}

		var session struct {
			SessionID string `json:"sessionId"`
			ProjectDir string `json:"projectDir"`
			PID       int    `json:"pid"`
		}
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		if session.PID <= 0 {
			continue
		}

		// Check if process is still alive
		proc, err := os.FindProcess(session.PID)
		if err != nil {
			continue
		}
		if err := proc.Signal(os.Signal(nil)); err != nil {
			// Process is dead, not an orphan
			continue
		}

		orphans = append(orphans, OrphanSession{
			SessionID:  session.SessionID,
			ProjectDir: session.ProjectDir,
			PID:        session.PID,
		})
	}

	return orphans, nil
}

// OrphanSession represents a Claude process that's running without a manager.
type OrphanSession struct {
	SessionID  string
	ProjectDir string
	PID        int
}

// ─── Attachment types ────────────────────────────────────

// ImageAttachment represents an image to send to Claude.
type ImageAttachment struct {
	Data     []byte
	MimeType string
}

// FileAttachment represents a file to send to Claude.
type FileAttachment struct {
	Data     []byte
	FileName string
}

// PermissionResult represents the user's decision on a permission request.
type PermissionResult struct {
	Behavior     string         `json:"behavior"`
	UpdatedInput map[string]any `json:"updatedInput,omitempty"`
	Message      string         `json:"message,omitempty"`
}
