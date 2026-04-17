package agent

import (
	"os"
	"path/filepath"
	"testing"
)

const testAgentContent = `---
name: "test-agent"
description: "A test agent for unit tests"
model: opus
color: purple
memory: user
tools:
  - "Read"
  - "Write"
---

# Test Agent

You are a test agent.
`

func TestParseAgentFile(t *testing.T) {
	fm, body, err := ParseAgentFile([]byte(testAgentContent))
	if err != nil {
		t.Fatalf("ParseAgentFile error: %v", err)
	}

	if fm.Name != "test-agent" {
		t.Errorf("name = %q, want %q", fm.Name, "test-agent")
	}
	if fm.Description != "A test agent for unit tests" {
		t.Errorf("description = %q, want %q", fm.Description, "A test agent for unit tests")
	}
	if fm.Model != "opus" {
		t.Errorf("model = %q, want %q", fm.Model, "opus")
	}
	if fm.Color != "purple" {
		t.Errorf("color = %q, want %q", fm.Color, "purple")
	}
	if fm.Memory != "user" {
		t.Errorf("memory = %q, want %q", fm.Memory, "user")
	}
	if len(fm.Tools) != 2 || fm.Tools[0] != "Read" || fm.Tools[1] != "Write" {
		t.Errorf("tools = %v, want [Read Write]", fm.Tools)
	}
	if body != "# Test Agent\n\nYou are a test agent." {
		t.Errorf("body = %q", body)
	}
}

func TestParseAgentFileMinimal(t *testing.T) {
	content := `---
name: "minimal"
description: "Minimal agent"
---
Body text.
`
	fm, body, err := ParseAgentFile([]byte(content))
	if err != nil {
		t.Fatalf("ParseAgentFile error: %v", err)
	}

	if fm.Name != "minimal" {
		t.Errorf("name = %q", fm.Name)
	}
	if fm.Model != "" {
		t.Errorf("model should be empty, got %q", fm.Model)
	}
	if body != "Body text." {
		t.Errorf("body = %q", body)
	}
}

func TestGenerateAgentFile(t *testing.T) {
	fm := AgentFrontmatter{
		Name:        "gen-agent",
		Description: "Generated agent",
		Model:       "sonnet",
		Color:       "cyan",
		Tools:       []string{"Read", "Glob"},
	}

	body := "# Generated Agent\n\nDo things."
	content := GenerateAgentFile(fm, body)

	// Verify round-trip
	parsedFm, parsedBody, err := ParseAgentFile(content)
	if err != nil {
		t.Fatalf("round-trip parse error: %v", err)
	}

	if parsedFm.Name != fm.Name {
		t.Errorf("round-trip name = %q, want %q", parsedFm.Name, fm.Name)
	}
	if parsedFm.Description != fm.Description {
		t.Errorf("round-trip description = %q, want %q", parsedFm.Description, fm.Description)
	}
	if parsedFm.Model != fm.Model {
		t.Errorf("round-trip model = %q, want %q", parsedFm.Model, fm.Model)
	}
	if parsedFm.Color != fm.Color {
		t.Errorf("round-trip color = %q, want %q", parsedFm.Color, fm.Color)
	}
	if len(parsedFm.Tools) != 2 {
		t.Errorf("round-trip tools = %v, want 2 items", parsedFm.Tools)
	}
	if parsedBody != body {
		t.Errorf("round-trip body = %q, want %q", parsedBody, body)
	}
}

func TestManagerCRUD(t *testing.T) {
	// Setup temp directories
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(homeDir, 0o755)
	os.MkdirAll(projectDir, 0o755)

	mgr := NewManagerWithDir(homeDir)

	// Create user-level agent
	err := mgr.CreateAgent(AgentDetail{
		AgentMeta: AgentMeta{
			Name:        "user-bot",
			Description: "User level agent",
			Model:       "opus",
		},
		Prompt: "You are a user-level agent.",
	}, "")
	if err != nil {
		t.Fatalf("CreateAgent error: %v", err)
	}

	// Verify file exists
	p := filepath.Join(homeDir, ".claude", "agents", "user-bot.md")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Error("agent file should exist")
	}

	// Get agent
	detail, err := mgr.GetAgent("user-bot", "")
	if err != nil {
		t.Fatalf("GetAgent error: %v", err)
	}
	if detail.Name != "user-bot" {
		t.Errorf("name = %q, want %q", detail.Name, "user-bot")
	}
	if detail.Source != SourceUser {
		t.Errorf("source = %q, want %q", detail.Source, SourceUser)
	}
	if detail.Prompt != "You are a user-level agent." {
		t.Errorf("prompt = %q", detail.Prompt)
	}

	// Update agent
	err = mgr.UpdateAgent("user-bot", "", AgentDetail{
		AgentMeta: AgentMeta{
			Description: "Updated description",
			Model:       "sonnet",
		},
	})
	if err != nil {
		t.Fatalf("UpdateAgent error: %v", err)
	}

	detail, err = mgr.GetAgent("user-bot", "")
	if err != nil {
		t.Fatalf("GetAgent after update error: %v", err)
	}
	if detail.Description != "Updated description" {
		t.Errorf("description after update = %q", detail.Description)
	}
	if detail.Model != "sonnet" {
		t.Errorf("model after update = %q", detail.Model)
	}

	// Create project-level agent
	err = mgr.CreateAgent(AgentDetail{
		AgentMeta: AgentMeta{
			Name:        "proj-bot",
			Description: "Project level agent",
		},
		Prompt: "You are a project-level agent.",
	}, projectDir)
	if err != nil {
		t.Fatalf("CreateAgent project error: %v", err)
	}

	// List agents
	agents, err := mgr.ListAgents(projectDir)
	if err != nil {
		t.Fatalf("ListAgents error: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("agent count = %d, want 2", len(agents))
	}

	// Delete agent
	err = mgr.DeleteAgent("user-bot", "")
	if err != nil {
		t.Fatalf("DeleteAgent error: %v", err)
	}

	agents, err = mgr.ListAgents("")
	if err != nil {
		t.Fatalf("ListAgents after delete error: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("agent count after delete = %d, want 0", len(agents))
	}
}

func TestManagerProjectOverridesUser(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(homeDir, 0o755)
	os.MkdirAll(projectDir, 0o755)

	mgr := NewManagerWithDir(homeDir)

	// Create same-named agent at both levels
	mgr.CreateAgent(AgentDetail{
		AgentMeta: AgentMeta{Name: "shared", Description: "User version", Model: "opus"},
		Prompt:    "User prompt",
	}, "")
	mgr.CreateAgent(AgentDetail{
		AgentMeta: AgentMeta{Name: "shared", Description: "Project version", Model: "sonnet"},
		Prompt:    "Project prompt",
	}, projectDir)

	// List should show only project version
	agents, err := mgr.ListAgents(projectDir)
	if err != nil {
		t.Fatalf("ListAgents error: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("agent count = %d, want 1", len(agents))
	}
	if agents[0].Source != SourceProject {
		t.Errorf("source = %q, want %q", agents[0].Source, SourceProject)
	}
	if agents[0].Model != "sonnet" {
		t.Errorf("model = %q, want %q", agents[0].Model, "sonnet")
	}

	// Get should return project version first
	detail, err := mgr.GetAgent("shared", projectDir)
	if err != nil {
		t.Fatalf("GetAgent error: %v", err)
	}
	if detail.Source != SourceProject {
		t.Errorf("source = %q, want %q", detail.Source, SourceProject)
	}

	// Get without projectDir should return user version
	detail, err = mgr.GetAgent("shared", "")
	if err != nil {
		t.Fatalf("GetAgent user error: %v", err)
	}
	if detail.Source != SourceUser {
		t.Errorf("source = %q, want %q", detail.Source, SourceUser)
	}
}

func TestManagerDuplicateCreate(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManagerWithDir(tmpDir)

	err := mgr.CreateAgent(AgentDetail{
		AgentMeta: AgentMeta{Name: "dup", Description: "First"},
		Prompt:    "Prompt",
	}, "")
	if err != nil {
		t.Fatalf("first CreateAgent error: %v", err)
	}

	err = mgr.CreateAgent(AgentDetail{
		AgentMeta: AgentMeta{Name: "dup", Description: "Second"},
		Prompt:    "Prompt",
	}, "")
	if err == nil {
		t.Error("expected error for duplicate agent name")
	}
}

func TestManagerDeleteNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManagerWithDir(tmpDir)

	err := mgr.DeleteAgent("no-such-agent", "")
	if err == nil {
		t.Error("expected error for non-existent agent")
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"simple", "simple"},
		{"with spaces", "with-spaces"},
		{"with/slash", "with-slash"},
		{"with..dots", "withdots"},
	}
	for _, tt := range tests {
		got := sanitizeName(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
