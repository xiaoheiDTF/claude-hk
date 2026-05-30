// Package session 提供 Claude Code 会话的收集、恢复、状态查看等核心功能。
package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// StatusOptions 控制 session-status（状态查看）的显示选项。
type StatusOptions struct {
	Verbose bool // 是否显示文件级别的详细信息
}

// SessionStatus 展示本地会话存储的当前状态。
// 遍历本地 sessions 目录，显示每个项目的会话数量、Git 信息、与 Claude 目录的同步状态等。
func SessionStatus(opts StatusOptions) error {
	baseDir := BaseDir()
	sessionsDir := filepath.Join(baseDir, "sessions")
	logger.Info("session", "session-status: storage=%s", sessionsDir)

	fmt.Printf("Session storage: %s\n\n", sessionsDir)

	// 检查 sessions 目录是否存在。
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

	// 收集 Claude 当前的项目目录 slug 集合，用于后续同步状态对比。
	claudeProjects := listClaudeProjectSlugs()

	// 逐个展示每个项目的信息。
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

		// 检查是否存在匹配的 Claude 项目目录，并统计其中的会话数量。
		claudeSessionCount := 0
		claudeDir := filepath.Join(ClaudeProjectsDir(), slug)
		if _, ok := claudeProjects[slug]; ok {
			if files, err := FindSessionJSONLFiles(claudeDir); err == nil {
				claudeSessionCount = len(files)
			}
		}

		logger.Debug("session", "project: %s slug=%s local=%d claude=%d", project, slug, len(meta.Sessions), claudeSessionCount)

		fmt.Printf("Project: %s (slug: %s)\n", project, slug)
		fmt.Printf("  Local storage: %d sessions | Claude dir: %d sessions\n",
			len(meta.Sessions), claudeSessionCount)
		if meta.GitRemote != "" {
			fmt.Printf("  Git remote: %s\n", meta.GitRemote)
		}
		if meta.LocalCwd != "" {
			fmt.Printf("  CWD: %s\n", meta.LocalCwd)
		}

		// 详细模式下展示每个会话的具体信息。
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

		// 同步状态提示。
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

// listClaudeProjectSlugs 返回 ~/.claude/projects/ 下所有 slug 的集合。
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

// shortID 截断长 ID 字符串，最多保留前 8 个字符并附加省略号。
func shortID(id string) string {
	if len(id) > 8 {
		return id[:8] + "…"
	}
	return id
}
