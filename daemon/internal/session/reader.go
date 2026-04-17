package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SessionMeta is the lightweight metadata for a session list item.
type SessionMeta struct {
	ID           string    `json:"id"`
	ProjectDir   string    `json:"projectDir,omitempty"` // Original work directory
	Summary      string    `json:"summary"`              // First user message (truncated)
	MessageCount int       `json:"messageCount"`
	ModifiedAt   time.Time `json:"modifiedAt"`
}

// HistoryEntry is a single user/assistant message in a session.
type HistoryEntry struct {
	Type      string          `json:"type"`      // "user" or "assistant"
	Content   string          `json:"content"`
	Timestamp time.Time       `json:"timestamp"`
	Raw       json.RawMessage `json:"raw,omitempty"`
}

// SessionDetail contains detailed info about a session.
type SessionDetail struct {
	SessionMeta
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	ToolCount    int `json:"toolCount"`
}

// Reader reads Claude Code session data from the filesystem.
type Reader struct {
	homeDir string
}

// NewReader creates a new session reader.
func NewReader() *Reader {
	home, _ := os.UserHomeDir()
	return &Reader{homeDir: home}
}

// NewReaderWithDir creates a Reader with a custom home directory (for testing).
func NewReaderWithDir(homeDir string) *Reader {
	return &Reader{homeDir: homeDir}
}

// ListSessions returns all sessions for a given work directory.
// If workDir is empty, returns sessions from all projects.
func (r *Reader) ListSessions(workDir string) ([]SessionMeta, error) {
	projectsBase := filepath.Join(r.homeDir, ".claude", "projects")

	if workDir != "" {
		return r.listSessionsForProject(projectsBase, workDir)
	}

	// List all projects
	return r.listAllSessions(projectsBase)
}

// GetSessionHistory reads the JSONL transcript and returns user/assistant messages.
func (r *Reader) GetSessionHistory(sessionID string, workDir string, limit int) ([]HistoryEntry, error) {
	path, err := r.findSessionFile(sessionID, workDir)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open session file: %w", err)
	}
	defer f.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)

	for scanner.Scan() {
		var raw struct {
			Type      string `json:"type"`
			Timestamp string `json:"timestamp"`
			Message   struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(scanner.Bytes(), &raw) != nil {
			continue
		}
		if raw.Type != "user" && raw.Type != "assistant" {
			continue
		}

		ts, _ := time.Parse(time.RFC3339Nano, raw.Timestamp)
		text := extractTextContent(raw.Message.Content)
		if text == "" {
			continue
		}

		entries = append(entries, HistoryEntry{
			Type:      raw.Type,
			Content:   text,
			Timestamp: ts,
		})
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return entries, nil
}

// DeleteSession removes a session JSONL file.
func (r *Reader) DeleteSession(sessionID string, workDir string) error {
	path, err := r.findSessionFile(sessionID, workDir)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// --- Internal helpers ---

func (r *Reader) listSessionsForProject(projectsBase, workDir string) ([]SessionMeta, error) {
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve work dir: %w", err)
	}

	projectDir := r.findProjectDir(projectsBase, absWorkDir)
	if projectDir == "" {
		return nil, nil
	}

	return r.scanProjectDir(projectDir, absWorkDir)
}

func (r *Reader) listAllSessions(projectsBase string) ([]SessionMeta, error) {
	entries, err := os.ReadDir(projectsBase)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read projects dir: %w", err)
	}

	var allSessions []SessionMeta
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectPath := filepath.Join(projectsBase, entry.Name())
		sessions, err := r.scanProjectDir(projectPath, "")
		if err != nil {
			continue
		}
		allSessions = append(allSessions, sessions...)
	}

	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[i].ModifiedAt.After(allSessions[j].ModifiedAt)
	})
	return allSessions, nil
}

func (r *Reader) scanProjectDir(projectDir, originalWorkDir string) ([]SessionMeta, error) {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read project dir: %w", err)
	}

	var sessions []SessionMeta
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		sessionID := strings.TrimSuffix(name, ".jsonl")
		info, err := entry.Info()
		if err != nil {
			continue
		}

		summary, msgCount := scanSessionMeta(filepath.Join(projectDir, name))

		sessions = append(sessions, SessionMeta{
			ID:           sessionID,
			ProjectDir:   originalWorkDir,
			Summary:      summary,
			MessageCount: msgCount,
			ModifiedAt:   info.ModTime(),
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ModifiedAt.After(sessions[j].ModifiedAt)
	})
	return sessions, nil
}

func scanSessionMeta(path string) (string, int) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)

	var summary string
	var count int

	for scanner.Scan() {
		var entry struct {
			Type    string `json:"type"`
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(scanner.Bytes(), &entry) != nil {
			continue
		}
		if entry.Type == "user" || entry.Type == "assistant" {
			count++
			if entry.Type == "user" && entry.Message.Content != "" {
				summary = entry.Message.Content
			}
		}
	}

	summary = StripXMLTags(summary)
	summary = TruncateSummary(summary, 40)
	return summary, count
}

func extractTextContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try plain string
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}

	// Try array of content blocks
	var blocks []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		Thinking string `json:"thinking"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}

	var parts []string
	for _, b := range blocks {
		if b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// findProjectDir locates the Claude Code session directory for a work dir.
func (r *Reader) findProjectDir(projectsBase, absWorkDir string) string {
	candidates := []string{
		EncodeProjectKey(absWorkDir),
		strings.ReplaceAll(absWorkDir, string(filepath.Separator), "-"),
		strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(absWorkDir),
		strings.NewReplacer("/", "-", "\\", "-", ":", "-", "_", "-").Replace(absWorkDir),
	}
	fwd := strings.ReplaceAll(absWorkDir, "\\", "/")
	candidates = append(candidates, strings.ReplaceAll(fwd, "/", "-"))

	for _, key := range candidates {
		dir := filepath.Join(projectsBase, key)
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	return ""
}

// findSessionFile locates a specific session JSONL file.
func (r *Reader) findSessionFile(sessionID, workDir string) (string, error) {
	projectsBase := filepath.Join(r.homeDir, ".claude", "projects")

	if workDir != "" {
		absWorkDir, err := filepath.Abs(workDir)
		if err != nil {
			return "", fmt.Errorf("resolve work dir: %w", err)
		}
		projectDir := r.findProjectDir(projectsBase, absWorkDir)
		if projectDir != "" {
			p := filepath.Join(projectDir, sessionID+".jsonl")
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}

	// Search all projects
	entries, err := os.ReadDir(projectsBase)
	if err != nil {
		return "", fmt.Errorf("session %q not found", sessionID)
	}

	targetFile := sessionID + ".jsonl"
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		p := filepath.Join(projectsBase, entry.Name(), targetFile)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("session %q not found", sessionID)
}
