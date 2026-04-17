package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncodeProjectKey(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"/Users/test/projects/myapp", "-Users-test-projects-myapp"},
		{"/home/user/code/project", "-home-user-code-project"},
		{"C:\\Users\\test\\project", "C--Users-test-project"},
		{"/Users/test/中文项目", "-Users-test-----"},
	}
	for _, tt := range tests {
		got := EncodeProjectKey(tt.input)
		if got != tt.want {
			t.Errorf("EncodeProjectKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTruncateSummary(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world foo bar", 5, "hello..."},
		{"  spaced  ", 10, "spaced"},
		{"你好世界测试", 3, "你好世..."},
	}
	for _, tt := range tests {
		got := TruncateSummary(tt.input, tt.max)
		if got != tt.want {
			t.Errorf("TruncateSummary(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
		}
	}
}

func TestStripXMLTags(t *testing.T) {
	input := `<tool_use>some <param>text</param> here</tool_use>`
	want := "some text here"
	got := StripXMLTags(input)
	if got != want {
		t.Errorf("StripXMLTags(%q) = %q, want %q", input, got, want)
	}
}

func TestListSessionsFromDir(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	workDir := filepath.Join(tmpDir, "projects", "myapp")
	os.MkdirAll(workDir, 0o755)

	// Create the Claude projects directory structure
	projectKey := EncodeProjectKey(workDir)
	projectsDir := filepath.Join(homeDir, ".claude", "projects", projectKey)
	os.MkdirAll(projectsDir, 0o755)

	// Create a test session JSONL file
	sessionContent := `{"type":"system","timestamp":"2024-01-01T00:00:00Z"}
{"type":"user","message":{"role":"user","content":"Hello Claude"},"timestamp":"2024-01-01T00:00:01Z"}
{"type":"assistant","message":{"role":"assistant","content":"Hi there!"},"timestamp":"2024-01-01T00:00:02Z"}
{"type":"user","message":{"role":"user","content":"Second message"},"timestamp":"2024-01-01T00:00:03Z"}
`
	sessionFile := filepath.Join(projectsDir, "test-session-id.jsonl")
	os.WriteFile(sessionFile, []byte(sessionContent), 0o644)

	reader := NewReaderWithDir(homeDir)
	sessions, err := reader.ListSessions(workDir)
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("session count = %d, want 1", len(sessions))
	}
	if sessions[0].ID != "test-session-id" {
		t.Errorf("session ID = %q, want %q", sessions[0].ID, "test-session-id")
	}
	if sessions[0].Summary != "Second message" {
		t.Errorf("summary = %q, want %q", sessions[0].Summary, "Second message")
	}
	if sessions[0].MessageCount != 3 {
		t.Errorf("message count = %d, want 3", sessions[0].MessageCount)
	}
}

func TestGetSessionHistory(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	workDir := filepath.Join(tmpDir, "projects", "myapp")
	os.MkdirAll(workDir, 0o755)

	projectKey := EncodeProjectKey(workDir)
	projectsDir := filepath.Join(homeDir, ".claude", "projects", projectKey)
	os.MkdirAll(projectsDir, 0o755)

	sessionContent := `{"type":"system","timestamp":"2024-01-01T00:00:00Z"}
{"type":"user","message":{"role":"user","content":"First question"},"timestamp":"2024-01-01T00:00:01Z"}
{"type":"assistant","message":{"role":"assistant","content":"First answer"},"timestamp":"2024-01-01T00:00:02Z"}
{"type":"user","message":{"role":"user","content":"Second question"},"timestamp":"2024-01-01T00:00:03Z"}
{"type":"assistant","message":{"role":"assistant","content":"Second answer"},"timestamp":"2024-01-01T00:00:04Z"}
`
	sessionFile := filepath.Join(projectsDir, "hist-test.jsonl")
	os.WriteFile(sessionFile, []byte(sessionContent), 0o644)

	reader := NewReaderWithDir(homeDir)

	// Test without limit
	history, err := reader.GetSessionHistory("hist-test", workDir, 0)
	if err != nil {
		t.Fatalf("GetSessionHistory error: %v", err)
	}
	if len(history) != 4 {
		t.Errorf("history length = %d, want 4", len(history))
	}

	// Test with limit
	history, err = reader.GetSessionHistory("hist-test", workDir, 2)
	if err != nil {
		t.Fatalf("GetSessionHistory with limit error: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("limited history length = %d, want 2", len(history))
	}
	if history[0].Type != "user" {
		t.Errorf("first entry type = %q, want %q", history[0].Type, "user")
	}
}

func TestDeleteSession(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	workDir := filepath.Join(tmpDir, "projects", "myapp")
	os.MkdirAll(workDir, 0o755)

	projectKey := EncodeProjectKey(workDir)
	projectsDir := filepath.Join(homeDir, ".claude", "projects", projectKey)
	os.MkdirAll(projectsDir, 0o755)

	sessionFile := filepath.Join(projectsDir, "del-test.jsonl")
	os.WriteFile(sessionFile, []byte("{}\n"), 0o644)

	reader := NewReaderWithDir(homeDir)
	err := reader.DeleteSession("del-test", workDir)
	if err != nil {
		t.Fatalf("DeleteSession error: %v", err)
	}

	if _, err := os.Stat(sessionFile); !os.IsNotExist(err) {
		t.Error("session file should be deleted")
	}
}

func TestDeleteNonExistentSession(t *testing.T) {
	tmpDir := t.TempDir()
	reader := NewReaderWithDir(tmpDir)

	err := reader.DeleteSession("no-such-session", "")
	if err == nil {
		t.Error("expected error for non-existent session")
	}
}

func TestListSessionsEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	reader := NewReaderWithDir(tmpDir)

	sessions, err := reader.ListSessions("")
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("session count = %d, want 0", len(sessions))
	}
}
