package deploy

import (
	"os"
	"path/filepath"
	"strings"
)

// FileAction 表示对文件执行的操作类型。
type FileAction int

const (
	// Copy 复制文件。
	Copy FileAction = iota
	// Skip 跳过文件。
	Skip
)

func (a FileAction) String() string {
	switch a {
	case Copy:
		return "复制"
	case Skip:
		return "跳过"
	default:
		return "未知"
	}
}

// FilePlan 描述对单个文件的操作计划。
type FilePlan struct {
	// RelPath 相对于 .claude/ 根目录的路径（使用 / 分隔符）。
	RelPath string
	// Action 要执行的操作。
	Action FileAction
	// Reason 操作原因（用于日志和 dry-run 输出）。
	Reason string
}

// runtimeExcludes 是运行时生成的文件/目录（首次和再次部署都不复制）。
// 目录以 "/" 结尾，用 HasPrefix 匹配。
var runtimeExcludes = []string{
	// 1. 初始化标记
	".initialized",
	// 2. 环境状态
	".python-state",
	"localLanguage/",
	// 3. 会话/技能状态
	"skills/.active",
	// 4. 日志
	"hooks/logs/",
	"hooks/lib/win32-foreground.log",
	"skills/log/",
}

// isRuntimeFile 判断文件是否属于运行时生成的文件。
func isRuntimeFile(relPath string) bool {
	for _, pattern := range runtimeExcludes {
		if strings.HasSuffix(pattern, "/") {
			// 目录前缀匹配
			if strings.HasPrefix(relPath, pattern) {
				return true
			}
		} else {
			// 精确文件匹配
			if relPath == pattern {
				return true
			}
		}
	}
	return false
}

// isProtectedFile 判断文件是否受保护（再次部署时不覆盖）。
// 受保护文件：settings.local.json、888-*/project.md
func isProtectedFile(relPath string) bool {
	// settings.local.json
	if relPath == "settings.local.json" {
		return true
	}

	// skills/888-*/project.md
	if strings.HasPrefix(relPath, "skills/888-") && strings.HasSuffix(relPath, "/project.md") {
		return true
	}

	return false
}

// BuildFilter 构建文件操作计划。
// templateDir: 模板目录路径
// targetClaudeDir: 目标项目的 .claude/ 目录路径（首次部署时可能不存在）
// isFirstDeploy: 是否首次部署（目标无 .claude/）
func BuildFilter(templateDir, targetClaudeDir string, isFirstDeploy bool) ([]FilePlan, error) {
	var plans []FilePlan

	err := filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		// 统一使用 / 分隔符
		normalizedRelPath := filepath.ToSlash(relPath)

		plan := FilePlan{
			RelPath: normalizedRelPath,
			Action:  Copy,
			Reason:  "模板文件",
		}

		// 目录始终标记为复制（CopyDirFiltered 会创建目录）
		if info.IsDir() {
			plans = append(plans, plan)
			return nil
		}

		// 运行时文件一律跳过
		if isRuntimeFile(normalizedRelPath) {
			plan.Action = Skip
			plan.Reason = "运行时文件"
			plans = append(plans, plan)
			return nil
		}

		// 再次部署时，受保护文件跳过
		if !isFirstDeploy && isProtectedFile(normalizedRelPath) {
			// 检查目标是否已存在该文件
			targetPath := filepath.Join(targetClaudeDir, relPath)
			if _, err := os.Stat(targetPath); err == nil {
				plan.Action = Skip
				plan.Reason = "受保护文件"
			}
		}

		plans = append(plans, plan)
		return nil
	})

	return plans, err
}
