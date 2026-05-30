// Package session 提供 Claude Code 会话的收集、恢复、状态查看等核心功能。
package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/trace"
)

// PushOptions 控制 session-push（收集会话）的行为。
type PushOptions struct {
	All     bool   // 是否收集所有项目，而不只是当前项目
	Force   bool   // 是否覆盖已存在的文件
	DryRun  bool   // 仅预览，不执行实际的文件操作
	Project string // 覆盖项目名称
}

// PushResult 记录一次 session-push 操作的结果。
type PushResult struct {
	Project         string // 项目名称
	SessionsNew     int    // 新收集的会话数量
	SessionsSkipped int    // 跳过的会话数量
	BytesCopied     int64  // 复制的字节数
}

// SessionPush 从 Claude Code 的会话目录 ~/.claude/projects/{slug}/
// 收集会话文件到本地存储目录 {exe-dir}/sessions/{slug}/。
// 根据 opts.All 决定是收集所有项目还是仅收集当前项目。
func SessionPush(opts PushOptions) ([]PushResult, error) {
	baseDir := BaseDir()
	logger.Info("session", "session-push: all=%v force=%v dry_run=%v", opts.All, opts.Force, opts.DryRun)

	if opts.All {
		return pushAllProjects(baseDir, opts)
	}
	return pushSingleProject(baseDir, opts)
}

// pushAllProjects 遍历 ~/.claude/projects/ 下的所有项目目录，逐个收集会话。
func pushAllProjects(baseDir string, opts PushOptions) ([]PushResult, error) {
	projectsDir := ClaudeProjectsDir()
	logger.Debug("session", "scanning projects dir: %s", projectsDir)
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

// pushSingleProject 仅收集当前工作目录对应项目的会话。
// 首先检测当前目录的 slug，然后收集该 slug 对应的 Claude 会话文件。
func pushSingleProject(baseDir string, opts PushOptions) ([]PushResult, error) {
	slug, _ := DetectLocalSlug()
	project := opts.Project
	if project == "" {
		project = trace.DetectProjectName()
	}

	claudeDir := filepath.Join(ClaudeProjectsDir(), slug)
	logger.Debug("session", "push: project=%s slug=%s claude_dir=%s", project, slug, claudeDir)

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

// collectProject 收集指定 slug 项目的会话文件。
// 流程：加载已有元数据 -> 扫描 Claude 目录中的 .jsonl 会话文件 ->
// 解析文件元信息 -> 复制新文件或强制覆盖 -> 更新 meta.json。
func collectProject(baseDir, slug, claudeDir string, opts PushOptions) (PushResult, error) {
	sessionDir := SessionDir(baseDir, slug)
	logger.Debug("session", "collect: slug=%s dir=%s", slug, sessionDir)

	// 加载已存在的会话元数据。
	meta, err := LoadMeta(sessionDir)
	if err != nil {
		return PushResult{}, err
	}

	// 在 Claude 的项目目录中查找所有会话 JSONL 文件。
	files, err := FindSessionJSONLFiles(claudeDir)
	if err != nil {
		return PushResult{}, fmt.Errorf("scan sessions: %w", err)
	}

	// 构建已收集会话 ID 的集合，用于快速判断哪些会话已存在。
	existing := make(map[string]bool)
	for _, s := range meta.Sessions {
		existing[s.SessionID] = true
	}

	project := slugToProjectName(slug)

	var result PushResult
	result.Project = project

	for _, file := range files {
		sessionID := strings.TrimSuffix(filepath.Base(file), ".jsonl")
		logger.Debug("session", "session file: %s size=%d", filepath.Base(file), fileStatSize(file))

		// 如果会话已存在且未强制覆盖，则跳过。
		if existing[sessionID] && !opts.Force {
			result.SessionsSkipped++
			if opts.DryRun {
				fmt.Printf("  [skip] %s (already collected)\n", filepath.Base(file))
			}
			continue
		}

		// 解析会话 JSONL 文件，提取元信息。
		entry, err := ParseSessionJSONL(file, slug)
		if err != nil {
			fmt.Printf("  ⚠ parse %s: %v\n", filepath.Base(file), err)
			continue
		}

		// 仅预览模式：只输出信息，不执行实际复制。
		if opts.DryRun {
			fmt.Printf("  [would copy] %s (%d records, %s)\n",
				entry.File, entry.RecordCount, formatSize(entry.FileSize))
			result.SessionsNew++
			result.BytesCopied += entry.FileSize
			continue
		}

		// 将会话文件复制到本地存储目录。
		dst := filepath.Join(sessionDir, entry.File)
		if err := copyFile(dst, file); err != nil {
			fmt.Printf("  ⚠ copy %s: %v\n", entry.File, err)
			continue
		}
		logger.Info("session", "copied: %s (%d bytes)", entry.File, entry.FileSize)

		// 更新或新增会话条目到元数据中。
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

		// 如果存在 subagents 子目录，一并复制。
		subagentsDir := filepath.Join(claudeDir, sessionID, "subagents")
		if info, err := os.Stat(subagentsDir); err == nil && info.IsDir() {
			dstSubagents := filepath.Join(sessionDir, sessionID, "subagents")
			if err := copyDir(dstSubagents, subagentsDir); err == nil {
				fmt.Printf("    ✓ subagents/ copied\n")
			}
		}
	}

	// 更新项目的元信息。
	meta.Project = project
	meta.LocalSlug = slug
	meta.GitRemote = gitRemoteURL()
	if cwd, err := os.Getwd(); err == nil {
		meta.LocalCwd = cwd
	}

	// 非预览模式下保存更新后的元数据。
	if !opts.DryRun {
		if err := SaveMeta(sessionDir, meta); err != nil {
			return result, fmt.Errorf("save meta.json: %w", err)
		}
		logger.Debug("session", "meta.json updated: %d sessions", len(meta.Sessions))
	}

	return result, nil
}

// fileStatSize 返回指定路径的文件大小，出错时返回 0。
func fileStatSize(path string) int64 {
	if fi, err := os.Stat(path); err == nil {
		return fi.Size()
	}
	return 0
}

// slugToProjectName 尝试从 slug 中提取项目名称。
// 例如："D--development-code-claude-tap" → "claude-tap"
func slugToProjectName(slug string) string {
	// 尝试找到最后一段有意义的片段。
	// 常见模式：
	//   D--development-code-claude-tap → claude-tap
	//   -home-user-projects-myapp → myapp
	parts := strings.Split(slug, "-")
	if len(parts) <= 1 {
		return slug
	}
	// 启发式规则：取最后 1-2 段作为项目名称。
	// 跳过驱动器盘符或空的首段。
	start := 0
	if parts[0] == "" {
		start = 1 // 以 "-" 开头（Unix 路径）
	}
	if len(parts)-start <= 2 {
		return strings.Join(parts[start:], "-")
	}
	// 取最后 2 段作为项目名称猜测。
	return strings.Join(parts[len(parts)-2:], "-")
}

// gitRemoteURL 获取当前工作目录对应的 Git 远程 origin URL。
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

// formatSize 将字节数格式化为人类可读的字符串（如 KB、MB）。
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

// copyDir 递归复制整个目录。
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
