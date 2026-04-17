package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Source indicates where an Agent definition comes from.
type Source string

const (
	SourceUser    Source = "user"    // ~/.claude/agents/<name>.md
	SourceProject Source = "project" // <project>/.claude/agents/<name>.md
)

// AgentMeta is the lightweight metadata returned by listing operations.
type AgentMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Model       string `json:"model,omitempty"`
	Color       string `json:"color,omitempty"`
	Source      Source `json:"source"` // "user" or "project"
}

// AgentDetail is the full Agent definition including prompt body.
type AgentDetail struct {
	AgentMeta
	Tools           []string `json:"tools,omitempty"`
	DisallowedTools []string `json:"disallowedTools,omitempty"`
	PermissionMode  string   `json:"permissionMode,omitempty"`
	Effort          string   `json:"effort,omitempty"`
	MaxTurns        int      `json:"maxTurns,omitempty"`
	Memory          string   `json:"memory,omitempty"`
	InitialPrompt   string   `json:"initialPrompt,omitempty"`
	Isolation       string   `json:"isolation,omitempty"`
	Background      bool     `json:"background,omitempty"`
	Prompt          string   `json:"prompt"` // Markdown body (the actual system prompt)
}

// Manager handles CRUD operations on Agent Markdown files.
type Manager struct {
	homeDir string // User home directory (for ~/.claude/agents/)
}

// NewManager creates a new Agent manager.
func NewManager() *Manager {
	home, _ := os.UserHomeDir()
	return &Manager{homeDir: home}
}

// NewManagerWithDir creates a Manager with a custom home directory (for testing).
func NewManagerWithDir(homeDir string) *Manager {
	return &Manager{homeDir: homeDir}
}

// ListAgents returns all agents, combining user-level and project-level.
// Project-level agents override user-level agents with the same name.
func (m *Manager) ListAgents(projectDir string) ([]AgentMeta, error) {
	agents := make(map[string]AgentMeta) // name -> meta, project overrides user

	// 1. Load user-level agents (~/.claude/agents/)
	userAgents, errs := m.loadAgentsFromDir(m.userAgentsDir(), SourceUser)
	for _, err := range errs {
		return nil, err
	}
	for _, a := range userAgents {
		agents[a.Name] = a
	}

	// 2. Load project-level agents (<project>/.claude/agents/)
	if projectDir != "" {
		projectAgents, errs := m.loadAgentsFromDir(m.projectAgentsDir(projectDir), SourceProject)
		for _, err := range errs {
			return nil, err
		}
		for _, a := range projectAgents {
			agents[a.Name] = a // Project-level overrides user-level
		}
	}

	// Convert map to sorted slice
	result := make([]AgentMeta, 0, len(agents))
	for _, a := range agents {
		result = append(result, a)
	}
	return result, nil
}

// GetAgent returns the full detail of a single agent by name.
func (m *Manager) GetAgent(name string, projectDir string) (*AgentDetail, error) {
	// Try project-level first
	if projectDir != "" {
		detail, err := m.readAgentFile(m.projectAgentPath(projectDir, name))
		if err == nil {
			detail.Source = SourceProject
			return detail, nil
		}
	}

	// Fall back to user-level
	detail, err := m.readAgentFile(m.userAgentPath(name))
	if err != nil {
		return nil, fmt.Errorf("agent %q not found: %w", name, err)
	}
	detail.Source = SourceUser
	return detail, nil
}

// CreateAgent writes a new agent Markdown file.
// dir controls where: empty = user-level, non-empty = project-level.
func (m *Manager) CreateAgent(detail AgentDetail, projectDir string) error {
	if detail.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if detail.Prompt == "" {
		return fmt.Errorf("agent prompt is required")
	}

	fm := AgentFrontmatter{
		Name:            detail.Name,
		Description:     detail.Description,
		Model:           detail.Model,
		Color:           detail.Color,
		Memory:          detail.Memory,
		Effort:          detail.Effort,
		PermissionMode:  detail.PermissionMode,
		Isolation:       detail.Isolation,
		InitialPrompt:   detail.InitialPrompt,
		MaxTurns:        detail.MaxTurns,
		Background:      detail.Background,
		Tools:           detail.Tools,
		DisallowedTools: detail.DisallowedTools,
	}

	content := GenerateAgentFile(fm, detail.Prompt)

	var dir string
	if projectDir != "" {
		dir = m.projectAgentsDir(projectDir)
	} else {
		dir = m.userAgentsDir()
	}

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating agents dir: %w", err)
	}

	filePath := filepath.Join(dir, sanitizeName(detail.Name)+".md")
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("agent %q already exists", detail.Name)
	}

	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		return fmt.Errorf("writing agent file: %w", err)
	}
	return nil
}

// UpdateAgent modifies an existing agent.
func (m *Manager) UpdateAgent(name string, projectDir string, update AgentDetail) error {
	// Find the existing agent
	existing, err := m.GetAgent(name, projectDir)
	if err != nil {
		return err
	}

	// Merge updates (non-zero fields overwrite)
	if update.Name != "" {
		existing.Name = update.Name
	}
	if update.Description != "" {
		existing.Description = update.Description
	}
	if update.Prompt != "" {
		existing.Prompt = update.Prompt
	}
	if update.Model != "" {
		existing.Model = update.Model
	}
	if update.Color != "" {
		existing.Color = update.Color
	}
	if update.Memory != "" {
		existing.Memory = update.Memory
	}
	if update.Effort != "" {
		existing.Effort = update.Effort
	}
	if update.PermissionMode != "" {
		existing.PermissionMode = update.PermissionMode
	}
	if update.Isolation != "" {
		existing.Isolation = update.Isolation
	}
	if update.InitialPrompt != "" {
		existing.InitialPrompt = update.InitialPrompt
	}
	if update.MaxTurns > 0 {
		existing.MaxTurns = update.MaxTurns
	}
	if update.Tools != nil {
		existing.Tools = update.Tools
	}
	if update.DisallowedTools != nil {
		existing.DisallowedTools = update.DisallowedTools
	}
	existing.Background = update.Background

	fm := AgentFrontmatter{
		Name:            existing.Name,
		Description:     existing.Description,
		Model:           existing.Model,
		Color:           existing.Color,
		Memory:          existing.Memory,
		Effort:          existing.Effort,
		PermissionMode:  existing.PermissionMode,
		Isolation:       existing.Isolation,
		InitialPrompt:   existing.InitialPrompt,
		MaxTurns:        existing.MaxTurns,
		Background:      existing.Background,
		Tools:           existing.Tools,
		DisallowedTools: existing.DisallowedTools,
	}

	content := GenerateAgentFile(fm, existing.Prompt)

	// Determine write path
	var writePath string
	if existing.Source == SourceProject && projectDir != "" {
		writePath = m.projectAgentPath(projectDir, name)
	} else {
		writePath = m.userAgentPath(name)
	}

	// If name changed, write to new path and delete old
	if update.Name != "" && update.Name != name {
		newPath := filepath.Join(filepath.Dir(writePath), sanitizeName(update.Name)+".md")
		if err := os.WriteFile(newPath, content, 0o644); err != nil {
			return fmt.Errorf("writing renamed agent file: %w", err)
		}
		os.Remove(writePath)
		return nil
	}

	if err := os.WriteFile(writePath, content, 0o644); err != nil {
		return fmt.Errorf("writing agent file: %w", err)
	}
	return nil
}

// DeleteAgent removes an agent file.
func (m *Manager) DeleteAgent(name string, projectDir string) error {
	// Try project-level first
	if projectDir != "" {
		p := m.projectAgentPath(projectDir, name)
		if _, err := os.Stat(p); err == nil {
			return os.Remove(p)
		}
	}

	// Try user-level
	p := m.userAgentPath(name)
	if _, err := os.Stat(p); err != nil {
		return fmt.Errorf("agent %q not found", name)
	}
	return os.Remove(p)
}

// --- Internal helpers ---

func (m *Manager) userAgentsDir() string {
	return filepath.Join(m.homeDir, ".claude", "agents")
}

func (m *Manager) projectAgentsDir(projectDir string) string {
	return filepath.Join(projectDir, ".claude", "agents")
}

func (m *Manager) userAgentPath(name string) string {
	return filepath.Join(m.userAgentsDir(), sanitizeName(name)+".md")
}

func (m *Manager) projectAgentPath(projectDir, name string) string {
	return filepath.Join(m.projectAgentsDir(projectDir), sanitizeName(name)+".md")
}

func (m *Manager) loadAgentsFromDir(dir string, source Source) ([]AgentMeta, []error) {
	var agents []AgentMeta
	var errs []error

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Directory doesn't exist is not an error
		}
		return nil, []error{fmt.Errorf("reading dir %s: %w", dir, err)}
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			errs = append(errs, fmt.Errorf("reading %s: %w", filePath, err))
			continue
		}

		fm, _, err := ParseAgentFile(data)
		if err != nil {
			errs = append(errs, fmt.Errorf("parsing %s: %w", filePath, err))
			continue
		}

		// Use filename as name if frontmatter name is empty
		name := fm.Name
		if name == "" {
			name = strings.TrimSuffix(entry.Name(), ".md")
		}

		agents = append(agents, AgentMeta{
			Name:        name,
			Description: fm.Description,
			Model:       fm.Model,
			Color:       fm.Color,
			Source:      source,
		})
	}

	return agents, errs
}

func (m *Manager) readAgentFile(path string) (*AgentDetail, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fm, body, err := ParseAgentFile(data)
	if err != nil {
		return nil, fmt.Errorf("parsing agent file: %w", err)
	}

	return &AgentDetail{
		AgentMeta: AgentMeta{
			Name:        fm.Name,
			Description: fm.Description,
			Model:       fm.Model,
			Color:       fm.Color,
		},
		Tools:           fm.Tools,
		DisallowedTools: fm.DisallowedTools,
		PermissionMode:  fm.PermissionMode,
		Effort:          fm.Effort,
		MaxTurns:        fm.MaxTurns,
		Memory:          fm.Memory,
		InitialPrompt:   fm.InitialPrompt,
		Isolation:       fm.Isolation,
		Background:      fm.Background,
		Prompt:          body,
	}, nil
}

// sanitizeName ensures the agent name is safe for use as a filename.
func sanitizeName(name string) string {
	// Replace problematic characters
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "..", "")
	return name
}
