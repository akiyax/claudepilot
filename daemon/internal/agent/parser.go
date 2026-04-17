package agent

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// AgentFrontmatter holds the YAML frontmatter fields of an Agent Markdown file.
// Fields match Claude Code's loadAgentsDir.ts AgentJsonSchema.
type AgentFrontmatter struct {
	Name            string   `json:"name" yaml:"name"`
	Description     string   `json:"description" yaml:"description"`
	Model           string   `json:"model,omitempty" yaml:"model,omitempty"`
	Tools           []string `json:"tools,omitempty" yaml:"tools,omitempty"`
	DisallowedTools []string `json:"disallowedTools,omitempty" yaml:"disallowedTools,omitempty"`
	PermissionMode  string   `json:"permissionMode,omitempty" yaml:"permissionMode,omitempty"`
	Effort          string   `json:"effort,omitempty" yaml:"effort,omitempty"` // low/medium/high/max or integer
	MaxTurns        int      `json:"maxTurns,omitempty" yaml:"maxTurns,omitempty"`
	Memory          string   `json:"memory,omitempty" yaml:"memory,omitempty"` // user/project/local
	Color           string   `json:"color,omitempty" yaml:"color,omitempty"`   // AgentColorName
	InitialPrompt   string   `json:"initialPrompt,omitempty" yaml:"initialPrompt,omitempty"`
	Isolation       string   `json:"isolation,omitempty" yaml:"isolation,omitempty"` // worktree/remote
	Background      bool     `json:"background,omitempty" yaml:"background,omitempty"`
}

// ParseAgentFile parses a Markdown file with YAML frontmatter into frontmatter + body.
// Format:
//
//	---
//	name: "agent-name"
//	description: "Agent description"
//	model: opus
//	---
//	# Agent prompt body...
func ParseAgentFile(content []byte) (frontmatter AgentFrontmatter, body string, err error) {
	// Split by --- delimiters
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var inFrontmatter bool
	var fmLines []string
	var bodyLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			// End of frontmatter
			inFrontmatter = false
			continue
		}

		if inFrontmatter {
			fmLines = append(fmLines, line)
		} else {
			bodyLines = append(bodyLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return AgentFrontmatter{}, "", fmt.Errorf("scanning file: %w", err)
	}

	// Parse YAML frontmatter (simple key: value parser)
	frontmatter = parseSimpleYAML(fmLines)

	// Trim leading empty lines from body
	body = strings.TrimLeft(strings.Join(bodyLines, "\n"), "\n")
	body = strings.TrimRight(body, "\n")

	return frontmatter, body, nil
}

// GenerateAgentFile creates a Markdown file with YAML frontmatter.
func GenerateAgentFile(fm AgentFrontmatter, body string) []byte {
	var buf bytes.Buffer
	buf.WriteString("---\n")

	// Required fields
	buf.WriteString(fmt.Sprintf("name: %q\n", fm.Name))
	buf.WriteString(fmt.Sprintf("description: %q\n", fm.Description))

	// Optional fields — only write if non-zero
	if fm.Model != "" {
		buf.WriteString(fmt.Sprintf("model: %s\n", fm.Model))
	}
	if fm.Color != "" {
		buf.WriteString(fmt.Sprintf("color: %s\n", fm.Color))
	}
	if fm.Memory != "" {
		buf.WriteString(fmt.Sprintf("memory: %s\n", fm.Memory))
	}
	if fm.Effort != "" {
		buf.WriteString(fmt.Sprintf("effort: %s\n", fm.Effort))
	}
	if fm.PermissionMode != "" {
		buf.WriteString(fmt.Sprintf("permissionMode: %s\n", fm.PermissionMode))
	}
	if fm.Isolation != "" {
		buf.WriteString(fmt.Sprintf("isolation: %s\n", fm.Isolation))
	}
	if fm.InitialPrompt != "" {
		buf.WriteString(fmt.Sprintf("initialPrompt: %q\n", fm.InitialPrompt))
	}
	if fm.MaxTurns > 0 {
		buf.WriteString(fmt.Sprintf("maxTurns: %d\n", fm.MaxTurns))
	}
	if fm.Background {
		buf.WriteString("background: true\n")
	}
	if len(fm.Tools) > 0 {
		buf.WriteString(fmt.Sprintf("tools:\n"))
		for _, t := range fm.Tools {
			buf.WriteString(fmt.Sprintf("  - %q\n", t))
		}
	}
	if len(fm.DisallowedTools) > 0 {
		buf.WriteString("disallowedTools:\n")
		for _, t := range fm.DisallowedTools {
			buf.WriteString(fmt.Sprintf("  - %q\n", t))
		}
	}

	buf.WriteString("---\n")
	buf.WriteString(body)
	buf.WriteString("\n")

	return buf.Bytes()
}

// parseSimpleYAML parses a minimal subset of YAML (key: value, key: [list]).
// This avoids importing a full YAML library — Agent frontmatter only uses simple scalars.
func parseSimpleYAML(lines []string) AgentFrontmatter {
	var fm AgentFrontmatter
	var currentListKey string
	var currentList []string

	flushList := func() {
		if currentListKey != "" && len(currentList) > 0 {
			switch currentListKey {
			case "tools":
				fm.Tools = currentList
			case "disallowedTools":
				fm.DisallowedTools = currentList
			}
		}
		currentListKey = ""
		currentList = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// List item (indented with - prefix)
		if strings.HasPrefix(trimmed, "- ") && currentListKey != "" {
			val := strings.Trim(strings.TrimSpace(trimmed[1:]), "\"'")
			currentList = append(currentList, val)
			continue
		}

		// New key:value pair
		idx := strings.Index(trimmed, ":")
		if idx < 0 {
			continue
		}

		// Flush previous list if any
		flushList()

		key := strings.TrimSpace(trimmed[:idx])
		val := strings.TrimSpace(trimmed[idx+1:])

		// Check if this starts a list (value is empty)
		if val == "" {
			currentListKey = key
			currentList = nil
			continue
		}

		// Strip quotes
		val = strings.Trim(val, "\"'")

		switch key {
		case "name":
			fm.Name = val
		case "description":
			fm.Description = val
		case "model":
			fm.Model = val
		case "color":
			fm.Color = val
		case "memory":
			fm.Memory = val
		case "effort":
			fm.Effort = val
		case "permissionMode":
			fm.PermissionMode = val
		case "isolation":
			fm.Isolation = val
		case "initialPrompt":
			fm.InitialPrompt = val
		case "maxTurns":
			var n int
			fmt.Sscanf(val, "%d", &n)
			fm.MaxTurns = n
		case "background":
			fm.Background = val == "true"
		}
	}

	flushList()
	return fm
}
