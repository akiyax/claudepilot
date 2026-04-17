package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ActiveSession represents a currently running Claude Code session,
// read from ~/.claude/sessions/*.json
type ActiveSession struct {
	PID        int    `json:"pid"`
	SessionID  string `json:"sessionId"`
	ProjectDir string `json:"projectDir"`
}

// GetActiveSessions reads ~/.claude/sessions/*.json to find running sessions.
func (r *Reader) GetActiveSessions() ([]ActiveSession, error) {
	sessionsDir := filepath.Join(r.homeDir, ".claude", "sessions")

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sessions dir: %w", err)
	}

	var active []ActiveSession
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(sessionsDir, entry.Name()))
		if err != nil {
			continue
		}

		var s ActiveSession
		if json.Unmarshal(data, &s) != nil {
			continue
		}

		// Check if PID is still alive
		if s.PID > 0 {
			proc, err := os.FindProcess(s.PID)
			if err != nil {
				continue
			}
			// Signal 0 checks if process exists without sending a signal
			if proc.Signal(nil) != nil {
				continue // Process is dead
			}
		}

		active = append(active, s)
	}

	return active, nil
}
