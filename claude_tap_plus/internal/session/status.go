package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StatusOptions controls session-status display.
type StatusOptions struct {
	Verbose bool // show file-level details
}

// SessionStatus displays the current state of local session storage.
func SessionStatus(opts StatusOptions) error {
	baseDir := BaseDir()
	sessionsDir := filepath.Join(baseDir, "sessions")

	fmt.Printf("Session storage: %s\n\n", sessionsDir)

	// Check if sessions directory exists.
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No sessions stored yet. Run 'claude-tap-plus session-push' first.")
			return nil
		}
		return fmt.Errorf("read session dir: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No sessions stored yet. Run 'claude-tap-plus session-push' first.")
		return nil
	}

	// Collect Claude's current project dirs for comparison.
	claudeProjects := listClaudeProjectSlugs()

	// Display each project.
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		projectDir := filepath.Join(sessionsDir, slug)

		meta, err := LoadMeta(projectDir)
		if err != nil {
			fmt.Printf("Slug: %s — error reading meta: %v\n", slug, err)
			continue
		}

		project := meta.Project
		if project == "" {
			project = slugToProjectName(slug)
		}

		// Check if there's a matching Claude project dir.
		claudeSessionCount := 0
		claudeDir := filepath.Join(ClaudeProjectsDir(), slug)
		if _, ok := claudeProjects[slug]; ok {
			if files, err := FindSessionJSONLFiles(claudeDir); err == nil {
				claudeSessionCount = len(files)
			}
		}

		fmt.Printf("Project: %s (slug: %s)\n", project, slug)
		fmt.Printf("  Local storage: %d sessions | Claude dir: %d sessions\n",
			len(meta.Sessions), claudeSessionCount)
		if meta.GitRemote != "" {
			fmt.Printf("  Git remote: %s\n", meta.GitRemote)
		}
		if meta.LocalCwd != "" {
			fmt.Printf("  CWD: %s\n", meta.LocalCwd)
		}

		if opts.Verbose && len(meta.Sessions) > 0 {
			fmt.Printf("  Sessions:\n")
			for _, s := range meta.Sessions {
				fmt.Printf("    %s  %d records  %s  branch=%s  models=%s\n",
					shortID(s.SessionID),
					s.RecordCount,
					formatSize(s.FileSize),
					s.GitBranch,
					strings.Join(s.ModelsUsed, ","))
				if s.FirstTimestamp != "" {
					fmt.Printf("      %s → %s\n", s.FirstTimestamp, s.LastTimestamp)
				}
			}
		}

		// Sync status.
		if claudeSessionCount > len(meta.Sessions) {
			fmt.Printf("  ⚠ Claude has %d sessions not yet collected\n", claudeSessionCount-len(meta.Sessions))
		} else if len(meta.Sessions) > claudeSessionCount {
			fmt.Printf("  ✓ %d sessions available for restore\n", len(meta.Sessions)-claudeSessionCount)
		} else {
			fmt.Printf("  ✓ Up to date\n")
		}
		fmt.Println()
	}

	return nil
}

// listClaudeProjectSlugs returns a set of all slugs in ~/.claude/projects/.
func listClaudeProjectSlugs() map[string]bool {
	result := make(map[string]bool)
	projectsDir := ClaudeProjectsDir()
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return result
	}
	for _, entry := range entries {
		if entry.IsDir() {
			result[entry.Name()] = true
		}
	}
	return result
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8] + "…"
	}
	return id
}
