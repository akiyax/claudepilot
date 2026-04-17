package session

import (
	"strings"
	"unicode/utf8"
)

// EncodeProjectKey converts an absolute path to a Claude Code project key.
// Claude Code stores sessions at ~/.claude/projects/{projectKey}/
// The encoding replaces /, :, _ with - and non-ASCII with -.
func EncodeProjectKey(absPath string) string {
	normalized := strings.ReplaceAll(absPath, "\\", "/")

	var result strings.Builder
	for _, r := range normalized {
		if r == '/' || r == ':' || r == '_' {
			result.WriteRune('-')
		} else if r < 128 {
			result.WriteRune(r)
		} else {
			result.WriteRune('-')
		}
	}
	return result.String()
}

// TruncateSummary shortens text for display, respecting rune boundaries.
func TruncateSummary(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) <= maxRunes {
		return text
	}
	runes := []rune(text)
	return string(runes[:maxRunes]) + "..."
}

// StripXMLTags removes XML tags from text.
func StripXMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}
