// Package session 提供 Claude Code 会话的收集、恢复、状态查看等核心功能。
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// PullOptions 控制 session-pull（恢复会话）的行为。
type PullOptions struct {
	All     bool   // 是否恢复所有项目
	Project string // 指定项目名称
	DryRun  bool   // 仅预览，不执行实际的文件操作
}

// PullResult 记录一次 session-pull 操作的结果。
type PullResult struct {
	Project          string // 项目名称
	SessionsRestored int    // 成功恢复的会话数量
	SessionsSkipped  int    // 跳过的会话数量
}

// SessionPull 从本地存储恢复会话文件到 ~/.claude/projects/{slug}/。
// 根据 opts.All 决定是恢复所有项目还是仅恢复当前项目。
func SessionPull(opts PullOptions) ([]PullResult, error) {
	baseDir := BaseDir()
	logger.Info("session", "session-pull: all=%v dry_run=%v", opts.All, opts.DryRun)

	if opts.All {
		return pullAllProjects(baseDir, opts)
	}
	return pullSingleProject(baseDir, opts)
}

// pullAllProjects 遍历本地 sessions 目录下的所有项目，逐个恢复会话。
func pullAllProjects(baseDir string, opts PullOptions) ([]PullResult, error) {
	sessionsDir := filepath.Join(baseDir, "sessions")
	logger.Debug("session", "scanning: %s", sessionsDir)
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

// pullSingleProject 仅恢复当前工作目录对应项目的会话。
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

// restoreProject 恢复指定 slug 项目的会话文件。
// 流程：加载本地元数据 -> 检查目标目录 -> 逐个复制会话文件 ->
// 恢复 subagents -> 更新 .claude.json。
func restoreProject(baseDir, slug string, opts PullOptions) (PullResult, error) {
	sessionDir := SessionDir(baseDir, slug)
	logger.Debug("session", "restore: slug=%s dir=%s", slug, sessionDir)
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

	// 确定恢复目标目录：~/.claude/projects/{slug}/
	// slug 是存储键，与原始来源匹配。
	targetDir := filepath.Join(ClaudeProjectsDir(), slug)
	_, cwd := DetectLocalSlug()

	fmt.Printf("Project: %s | Slug: %s | Target dir: %s\n", project, slug, targetDir)

	var result PullResult
	result.Project = project

	// 追踪最新的会话，用于后续更新 .claude.json。
	var mostRecentSession *SessionEntry

	for i := range meta.Sessions {
		entry := &meta.Sessions[i]
		srcFile := filepath.Join(sessionDir, entry.File)
		logger.Debug("session", "restore file: %s", entry.File)

		// 检查源文件是否存在。
		if _, err := os.Stat(srcFile); os.IsNotExist(err) {
			fmt.Printf("  ⚠ %s: source file missing\n", entry.File)
			result.SessionsSkipped++
			continue
		}

		// 检查目标位置是否已存在同名文件。
		dstFile := filepath.Join(targetDir, entry.File)
		if _, err := os.Stat(dstFile); err == nil && !opts.DryRun {
			// 文件已存在，比较大小。
			if fi, err := os.Stat(dstFile); err == nil && fi.Size() == entry.FileSize {
				logger.Debug("session", "skip existing: %s", entry.File)
				fmt.Printf("  [skip] %s (already exists)\n", entry.File)
				result.SessionsSkipped++
				continue
			}
		}

		if opts.DryRun {
			// 仅预览：输出预计恢复的信息。
			fmt.Printf("  [would restore] %s (%d records, %s)\n",
				entry.File, entry.RecordCount, formatSize(entry.FileSize))
			result.SessionsRestored++
		} else {
			// 实际复制文件。
			if err := copyFile(dstFile, srcFile); err != nil {
				fmt.Printf("  ⚠ restore %s: %v\n", entry.File, err)
				continue
			}
			logger.Info("session", "restored: %s -> %s", entry.File, targetDir)
			fmt.Printf("  ✓ %s → %s\n", entry.File, targetDir)
			result.SessionsRestored++

			// 如果存在 subagents 子目录，一并恢复。
			srcSubagents := filepath.Join(sessionDir, entry.SessionID, "subagents")
			if info, err := os.Stat(srcSubagents); err == nil && info.IsDir() {
				dstSubagents := filepath.Join(targetDir, entry.SessionID, "subagents")
				if err := copyDir(dstSubagents, srcSubagents); err == nil {
					fmt.Printf("    ✓ subagents/ restored\n")
				}
			}
		}

		// 更新最新会话追踪。
		if mostRecentSession == nil || entry.LastTimestamp > mostRecentSession.LastTimestamp {
			mostRecentSession = entry
		}
	}

	// 使用最新会话信息更新 .claude.json。
	if mostRecentSession != nil && !opts.DryRun {
		if err := updateClaudeJSON(cwd, slug, mostRecentSession); err != nil {
			fmt.Printf("  ⚠ update .claude.json: %v\n", err)
		} else {
			logger.Info("session", "updated .claude.json: lastSession=%s", mostRecentSession.SessionID)
			fmt.Printf("  ✓ .claude.json updated (lastSessionId=%s)\n", mostRecentSession.SessionID)
		}
	}

	return result, nil
}

// updateClaudeJSON 更新 ~/.claude.json 中对应项目的会话元数据。
func updateClaudeJSON(cwd, slug string, entry *SessionEntry) error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".claude.json")

	// 读取现有配置。
	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read .claude.json: %w", err)
		}
		data = []byte("{}")
	}

	// 使用泛型 map 解析以保留所有已有字段。
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		config = make(map[string]any)
	}

	// Claude Code 使用正斜杠路径作为键。
	projectKey := CwdToClaudeKey(cwd)

	// 获取或创建项目条目。
	projectEntry, ok := config[projectKey].(map[string]any)
	if !ok {
		projectEntry = make(map[string]any)
	}

	// 更新会话相关字段。
	projectEntry["lastSessionId"] = entry.SessionID
	projectEntry["lastSessionModified"] = time.Now().UnixMilli()
	projectEntry["lastGracefulShutdown"] = true

	if entry.FirstTimestamp != "" {
		projectEntry["lastSessionFirstPrompt"] = "" // 元数据中不包含提示文本。
	}

	// 如果已存在指向更新会话的 lastSessionId，则跳过（mostRecentSession 保证了这是最新的）。
	if existingLastSession, ok := projectEntry["lastSessionId"].(string); ok {
		if existingLastSession == entry.SessionID {
			// 已经指向该会话，无需更新。
		}
		// 如果不同，保留较新的那个 — mostRecentSession 已确保这一点。
	}

	config[projectKey] = projectEntry

	// 写回配置文件。
	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal .claude.json: %w", err)
	}

	return os.WriteFile(configPath, out, 0o644)
}

// parseTimestamp 解析 ISO 时间戳字符串。
// 支持多种常见格式，解析失败返回零值时间。
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
