package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/trace"
)

// PushOptions controls session-push behavior.
type PushOptions struct {
	All     bool   // collect all projects, not just current
	Force   bool   // overwrite existing files
	DryRun  bool   // preview only, no file operations
	Project string // override project name
}

// PushResult contains the outcome of a session-push operation.
type PushResult struct {
	Project     string
	SessionsNew int
	SessionsSkipped int
	BytesCopied int64
}

// SessionPush collects Claude Code session files from ~/.claude/projects/{slug}/
// into the local session storage at {exe-dir}/sessions/{slug}/.
func SessionPush(opts PushOptions) ([]PushResult, error) {
	baseDir := BaseDir()

	if opts.All {
		return pushAllProjects(baseDir, opts)
	}
	return pushSingleProject(baseDir, opts)
}

func pushAllProjects(baseDir string, opts PushOptions) ([]PushResult, error) {
	projectsDir := ClaudeProjectsDir()
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, fmt.Errorf("read projects dir: %w", err)
	}

	var results []PushResult
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		result, err := collectProject(baseDir, slug, filepath.Join(projectsDir, slug), opts)
		if err != nil {
			fmt.Printf("  ⚠ %s: %v\n", slug, err)
			continue
		}
		results = append(results, result)
	}
	return results, nil
}

func pushSingleProject(baseDir string, opts PushOptions) ([]PushResult, error) {
	slug, _ := DetectLocalSlug()
	project := opts.Project
	if project == "" {
		project = trace.DetectProjectName()
	}

	claudeDir := filepath.Join(ClaudeProjectsDir(), slug)

	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no Claude session directory found for slug %q", slug)
	}

	fmt.Printf("Project: %s | Slug: %s | Claude dir: %s\n", project, slug, claudeDir)

	result, err := collectProject(baseDir, slug, claudeDir, opts)
	if err != nil {
		return nil, err
	}

	return []PushResult{result}, nil
}

func collectProject(baseDir, slug, claudeDir string, opts PushOptions) (PushResult, error) {
	sessionDir := SessionDir(baseDir, slug)

	// Load existing meta.
	meta, err := LoadMeta(sessionDir)
	if err != nil {
		return PushResult{}, err
	}

	// Find session JSONL files in Claude's project directory.
	files, err := FindSessionJSONLFiles(claudeDir)
	if err != nil {
		return PushResult{}, fmt.Errorf("scan sessions: %w", err)
	}

	// Build set of already-collected session IDs.
	existing := make(map[string]bool)
	for _, s := range meta.Sessions {
		existing[s.SessionID] = true
	}

	project := slugToProjectName(slug)

	var result PushResult
	result.Project = project

	for _, file := range files {
		sessionID := strings.TrimSuffix(filepath.Base(file), ".jsonl")

		if existing[sessionID] && !opts.Force {
			result.SessionsSkipped++
			if opts.DryRun {
				fmt.Printf("  [skip] %s (already collected)\n", filepath.Base(file))
			}
			continue
		}

		// Parse session metadata.
		entry, err := ParseSessionJSONL(file, slug)
		if err != nil {
			fmt.Printf("  ⚠ parse %s: %v\n", filepath.Base(file), err)
			continue
		}

		if opts.DryRun {
			fmt.Printf("  [would copy] %s (%d records, %s)\n",
				entry.File, entry.RecordCount, formatSize(entry.FileSize))
			result.SessionsNew++
			result.BytesCopied += entry.FileSize
			continue
		}

		// Copy file to local storage.
		dst := filepath.Join(sessionDir, entry.File)
		if err := copyFile(dst, file); err != nil {
			fmt.Printf("  ⚠ copy %s: %v\n", entry.File, err)
			continue
		}

		// Update or add entry in meta.
		if existing[sessionID] {
			for i, s := range meta.Sessions {
				if s.SessionID == sessionID {
					meta.Sessions[i] = *entry
					break
				}
			}
		} else {
			meta.Sessions = append(meta.Sessions, *entry)
		}

		result.SessionsNew++
		result.BytesCopied += entry.FileSize
		fmt.Printf("  ✓ %s (%d records, %s)\n", entry.File, entry.RecordCount, formatSize(entry.FileSize))

		// Copy subagents directory if present.
		subagentsDir := filepath.Join(claudeDir, sessionID, "subagents")
		if info, err := os.Stat(subagentsDir); err == nil && info.IsDir() {
			dstSubagents := filepath.Join(sessionDir, sessionID, "subagents")
			if err := copyDir(dstSubagents, subagentsDir); err == nil {
				fmt.Printf("    ✓ subagents/ copied\n")
			}
		}
	}

	// Update meta project info.
	meta.Project = project
	meta.LocalSlug = slug
	meta.GitRemote = gitRemoteURL()
	if cwd, err := os.Getwd(); err == nil {
		meta.LocalCwd = cwd
	}

	if !opts.DryRun {
		if err := SaveMeta(sessionDir, meta); err != nil {
			return result, fmt.Errorf("save meta.json: %w", err)
		}
	}

	return result, nil
}

// slugToProjectName attempts to extract a project name from a slug.
// e.g. "D--development-code-claude-tap" → "claude-tap"
func slugToProjectName(slug string) string {
	// Try to find last meaningful segment.
	// Common patterns:
	//   D--development-code-claude-tap → claude-tap
	//   -home-user-projects-myapp → myapp
	parts := strings.Split(slug, "-")
	if len(parts) <= 1 {
		return slug
	}
	// Heuristic: take last 1-2 segments that look like a project name.
	// Skip drive letter or empty first segment.
	start := 0
	if parts[0] == "" {
		start = 1 // leading "-" (Unix path)
	}
	if len(parts)-start <= 2 {
		return strings.Join(parts[start:], "-")
	}
	// Take last 2 segments as project name guess.
	return strings.Join(parts[len(parts)-2:], "-")
}

// gitRemoteURL returns the git remote origin URL for the current working directory.
func gitRemoteURL() string {
	cwd, _ := os.Getwd()
	if cwd == "" {
		return ""
	}
	configPath := filepath.Join(cwd, ".git", "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	content := string(data)
	lines := strings.Split(content, "\n")
	inOrigin := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == `[remote "origin"]` {
			inOrigin = true
			continue
		}
		if inOrigin && strings.HasPrefix(trimmed, "url = ") {
			return strings.TrimPrefix(trimmed, "url = ")
		}
		if inOrigin && strings.HasPrefix(trimmed, "[") {
			break
		}
	}
	return ""
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)
	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// copyDir recursively copies a directory.
func copyDir(dst, src string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		return copyFile(dstPath, path)
	})
}
