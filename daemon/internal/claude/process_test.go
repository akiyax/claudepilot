package claude

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name string
		opts ProcessOptions
		want []string
	}{
		{
			name: "fresh session",
			opts: ProcessOptions{},
			want: []string{"--output-format", "stream-json", "--input-format", "stream-json", "--permission-prompt-tool", "stdio", "--verbose"},
		},
		{
			name: "resume session",
			opts: ProcessOptions{SessionID: "sess-123"},
			want: []string{"--resume", "sess-123"},
		},
		{
			name: "with agent and model",
			opts: ProcessOptions{AgentName: "code-reviewer", Model: "opus", PermissionMode: "bypassPermissions", Effort: "high"},
			want: []string{"--agent", "code-reviewer", "--model", "opus", "--permission-mode", "bypassPermissions", "--effort", "high"},
		},
		{
			name: "continue session",
			opts: ProcessOptions{SessionID: "continue"},
			want: []string{"--continue", "--fork-session"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildArgs(tt.opts)
			for _, w := range tt.want {
				found := false
				for _, a := range args {
					if a == w {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected arg %q not found in %v", w, args)
				}
			}
		})
	}
}

func TestFilterEnv(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"CLAUDECODE_SESSION=123",
		"HOME=/home/user",
		"CLAUDECODE_NESTED=true",
	}
	filtered := filterEnv(env, "CLAUDECODE")
	if len(filtered) != 2 {
		t.Errorf("filtered len = %d, want 2", len(filtered))
	}
	for _, e := range filtered {
		if strings.HasPrefix(e, "CLAUDECODE") {
			t.Errorf("CLAUDECODE var not filtered: %s", e)
		}
	}
}

func TestMergeEnv(t *testing.T) {
	base := []string{"PATH=/usr/bin", "HOME=/home/user"}
	extra := []string{"HOME=/new/home", "FOO=bar"}
	merged := mergeEnv(base, extra)

	envMap := make(map[string]string)
	for _, e := range merged {
		parts := strings.SplitN(e, "=", 2)
		envMap[parts[0]] = parts[1]
	}

	if envMap["HOME"] != "/new/home" {
		t.Errorf("HOME = %s, want /new/home", envMap["HOME"])
	}
	if envMap["FOO"] != "bar" {
		t.Errorf("FOO = %s, want bar", envMap["FOO"])
	}
	if envMap["PATH"] != "/usr/bin" {
		t.Errorf("PATH = %s, want /usr/bin", envMap["PATH"])
	}
}

func TestParseEventsFromRaw(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantType string
	}{
		{"system", `{"type":"system","subtype":"init","session_id":"s1"}`, "system"},
		{"assistant text", `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}`, "assistant"},
		{"result", `{"type":"result","subtype":"success","content":"done"}`, "result"},
		{"control_request", `{"type":"control_request","request_id":"p1","request":{"tool_name":"Bash"}}`, "control_request"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw map[string]any
			if err := json.Unmarshal([]byte(tt.raw), &raw); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if raw["type"] != tt.wantType {
				t.Errorf("type = %v, want %s", raw["type"], tt.wantType)
			}
		})
	}
}

func TestDetectCLIVersion(t *testing.T) {
	// This test just verifies the function doesn't panic
	// (claude may or may not be installed in CI)
	version := DetectCLIVersion()
	_ = version // we just want it to not panic
}

func TestDetectOrphans_NoSessionsDir(t *testing.T) {
	// Use a non-existent home dir to test the no-sessions case
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", "/tmp/nonexistent-claudepilot-test-"+strings.Repeat("x", 8))
	defer os.Setenv("HOME", origHome)

	orphans, err := DetectOrphans()
	if err != nil {
		t.Fatalf("DetectOrphans error: %v", err)
	}
	if len(orphans) != 0 {
		t.Errorf("orphans = %d, want 0 for non-existent dir", len(orphans))
	}
}
