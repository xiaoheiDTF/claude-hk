package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PullOptions controls session-pull behavior.
type PullOptions struct {
	All     bool   // restore all projects
	Project string // specify project name
	DryRun  bool   // preview only
}

// PullResult contains the outcome of a session-pull operation.
type PullResult struct {
	Project        string
	SessionsRestored int
	SessionsSkipped int
}

// SessionPull restores sessions from local storage back to ~/.claude/projects/{slug}/.
func SessionPull(opts PullOptions) ([]PullResult, error) {
	baseDir := BaseDir()

	if opts.All {
		return pullAllProjects(baseDir, opts)
	}
	return pullSingleProject(baseDir, opts)
}

func pullAllProjects(baseDir string, opts PullOptions) ([]PullResult, error) {
	sessionsDir := filepath.Join(baseDir, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No sessions stored locally.")
			return nil, nil
		}
		return nil, fmt.Errorf("read session dir: %w", err)
	}

	var results []PullResult
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		result, err := restoreProject(baseDir, slug, opts)
		if err != nil {
			fmt.Printf("  ⚠ %s: %v\n", slug, err)
			continue
		}
		results = append(results, result)
	}
	return results, nil
}

func pullSingleProject(baseDir string, opts PullOptions) ([]PullResult, error) {
	slug, _ := DetectLocalSlug()

	sessionDir := SessionDir(baseDir, slug)
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no local sessions for slug %q", slug)
	}

	result, err := restoreProject(baseDir, slug, opts)
	if err != nil {
		return nil, err
	}
	return []PullResult{result}, nil
}

func restoreProject(baseDir, slug string, opts PullOptions) (PullResult, error) {
	sessionDir := SessionDir(baseDir, slug)
	meta, err := LoadMeta(sessionDir)
	if err != nil {
		return PullResult{}, err
	}

	project := meta.Project
	if project == "" {
		project = slugToProjectName(slug)
	}

	if len(meta.Sessions) == 0 {
		fmt.Printf("Project %s (%s): no sessions in local storage.\n", project, slug)
		return PullResult{Project: project}, nil
	}

	// Determine target: restore to ~/.claude/projects/{slug}/
	// The slug IS the storage key, so it matches the original source.
	targetDir := filepath.Join(ClaudeProjectsDir(), slug)
	_, cwd := DetectLocalSlug()

	fmt.Printf("Project: %s | Slug: %s | Target dir: %s\n", project, slug, targetDir)

	var result PullResult
	result.Project = project

	// Track most recent session for .claude.json update.
	var mostRecentSession *SessionEntry

	for i := range meta.Sessions {
		entry := &meta.Sessions[i]
		srcFile := filepath.Join(sessionDir, entry.File)

		if _, err := os.Stat(srcFile); os.IsNotExist(err) {
			fmt.Printf("  ⚠ %s: source file missing\n", entry.File)
			result.SessionsSkipped++
			continue
		}

		// Check if already exists in target.
		dstFile := filepath.Join(targetDir, entry.File)
		if _, err := os.Stat(dstFile); err == nil && !opts.DryRun {
			// File exists, compare sizes.
			if fi, err := os.Stat(dstFile); err == nil && fi.Size() == entry.FileSize {
				fmt.Printf("  [skip] %s (already exists)\n", entry.File)
				result.SessionsSkipped++
				continue
			}
		}

		if opts.DryRun {
			fmt.Printf("  [would restore] %s (%d records, %s)\n",
				entry.File, entry.RecordCount, formatSize(entry.FileSize))
			result.SessionsRestored++
		} else {
			if err := copyFile(dstFile, srcFile); err != nil {
				fmt.Printf("  ⚠ restore %s: %v\n", entry.File, err)
				continue
			}
			fmt.Printf("  ✓ %s → %s\n", entry.File, targetDir)
			result.SessionsRestored++

			// Restore subagents if present.
			srcSubagents := filepath.Join(sessionDir, entry.SessionID, "subagents")
			if info, err := os.Stat(srcSubagents); err == nil && info.IsDir() {
				dstSubagents := filepath.Join(targetDir, entry.SessionID, "subagents")
				if err := copyDir(dstSubagents, srcSubagents); err == nil {
					fmt.Printf("    ✓ subagents/ restored\n")
				}
			}
		}

		// Track most recent session.
		if mostRecentSession == nil || entry.LastTimestamp > mostRecentSession.LastTimestamp {
			mostRecentSession = entry
		}
	}

	// Update .claude.json with the most recent session info.
	if mostRecentSession != nil && !opts.DryRun {
		if err := updateClaudeJSON(cwd, slug, mostRecentSession); err != nil {
			fmt.Printf("  ⚠ update .claude.json: %v\n", err)
		} else {
			fmt.Printf("  ✓ .claude.json updated (lastSessionId=%s)\n", mostRecentSession.SessionID)
		}
	}

	return result, nil
}

// updateClaudeJSON updates the project entry in ~/.claude.json with session metadata.
func updateClaudeJSON(cwd, slug string, entry *SessionEntry) error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".claude.json")

	// Read existing config.
	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read .claude.json: %w", err)
		}
		data = []byte("{}")
	}

	// Parse as generic map to preserve all existing fields.
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		config = make(map[string]any)
	}

	// Claude Code uses forward-slash path as key.
	projectKey := CwdToClaudeKey(cwd)

	// Get or create project entry.
	projectEntry, ok := config[projectKey].(map[string]any)
	if !ok {
		projectEntry = make(map[string]any)
	}

	// Update session fields.
	projectEntry["lastSessionId"] = entry.SessionID
	projectEntry["lastSessionModified"] = time.Now().UnixMilli()
	projectEntry["lastGracefulShutdown"] = true

	if entry.FirstTimestamp != "" {
		projectEntry["lastSessionFirstPrompt"] = "" // We don't have the prompt text in meta.
	}

	// Only update if not already set by a more recent session.
	if existingLastSession, ok := projectEntry["lastSessionId"].(string); ok {
		if existingLastSession == entry.SessionID {
			// Already pointing to this session, skip update.
		}
		// If different, we keep the more recent one — mostRecentSession ensures this.
	}

	config[projectKey] = projectEntry

	// Write back.
	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal .claude.json: %w", err)
	}

	return os.WriteFile(configPath, out, 0o644)
}

// parseTimestamp parses an ISO timestamp string.
func parseTimestamp(s string) time.Time {
	for _, format := range []string{
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
		time.RFC3339Nano,
	} {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
