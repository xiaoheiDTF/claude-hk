// Package session 提供 Claude Code 会话的收集、恢复、状态查看等核心功能。
package session

import (
	"os"
	"path/filepath"
	"strings"
)

// GenerateSlug 将绝对路径转换为 Claude Code 使用的 slug 格式。
// 算法：将所有 ":"、"/"、"\" 替换为 "-"。
func GenerateSlug(absPath string) string {
	s := strings.ReplaceAll(absPath, ":", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, `\`, "-")
	return s
}

// DetectLocalSlug 计算当前工作目录对应的 slug。
// 返回 (slug, cwd)。
func DetectLocalSlug() (string, string) {
	cwd, _ := os.Getwd()
	return GenerateSlug(cwd), cwd
}

// ClaudeProjectsDir 返回 Claude Code 的项目目录 ~/.claude/projects/。
func ClaudeProjectsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

// FindSlugForProject 在 ~/.claude/projects/ 中搜索与给定项目名称匹配的 slug。
// 查找以项目名称结尾的 slug（或项目名称作为 slug 最后路径组件）。
func FindSlugForProject(projectName string) (slug string, found bool) {
	projectsDir := ClaudeProjectsDir()
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return "", false
	}

	// 查找以项目名称结尾的 slug。
	// 例如 "claude-tap" 可匹配 "D--development-code-claude-tap"。
	suffix := "-" + projectName
	for _, entry := range entries {
		name := entry.Name()
		if name == projectName || strings.HasSuffix(name, suffix) {
			return name, true
		}
	}
	return "", false
}

// CwdToClaudeKey 将绝对路径转换为 .claude.json 中使用的键格式。
// Claude Code 使用正斜杠格式："D:/development/code/claude-tap"。
func CwdToClaudeKey(absPath string) string {
	return strings.ReplaceAll(absPath, `\`, "/")
}
