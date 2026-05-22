package session

import (
	"os"
	"path/filepath"
	"strings"
)

// GenerateSlug converts an absolute path to Claude Code's slug format.
// Algorithm: replace all ":", "/", "\" with "-"
func GenerateSlug(absPath string) string {
	s := strings.ReplaceAll(absPath, ":", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, `\`, "-")
	return s
}

// DetectLocalSlug computes the slug for the current working directory.
// Returns (slug, cwd).
func DetectLocalSlug() (string, string) {
	cwd, _ := os.Getwd()
	return GenerateSlug(cwd), cwd
}

// ClaudeProjectsDir returns ~/.claude/projects/
func ClaudeProjectsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

// FindSlugForProject searches ~/.claude/projects/ for a slug that matches
// the given project name. It looks for slugs ending with the project name
// (or containing it as the last path component in the slug).
func FindSlugForProject(projectName string) (slug string, found bool) {
	projectsDir := ClaudeProjectsDir()
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return "", false
	}

	// Look for slug ending with the project name.
	// e.g. "claude-tap" matches "D--development-code-claude-tap"
	suffix := "-" + projectName
	for _, entry := range entries {
		name := entry.Name()
		if name == projectName || strings.HasSuffix(name, suffix) {
			return name, true
		}
	}
	return "", false
}

// CwdToClaudeKey converts an absolute path to the key format used in .claude.json.
// Claude Code uses forward slashes: "D:/development/code/claude-tap"
func CwdToClaudeKey(absPath string) string {
	return strings.ReplaceAll(absPath, `\`, "/")
}
