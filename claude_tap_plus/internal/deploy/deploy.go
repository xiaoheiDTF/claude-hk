package deploy

import (
	"fmt"
	"os"
	"path/filepath"
)

// Options 部署选项。
type Options struct {
	// TargetPath 目标项目根目录路径。
	TargetPath string
	// TemplateDir 模板目录路径。
	TemplateDir string
	// DryRun 只输出计划不执行复制。
	DryRun bool
}

// Result 部署结果。
type Result struct {
	// FilesCopied 已复制的文件数。
	FilesCopied int
	// FilesSkipped 已跳过的文件数。
	FilesSkipped int
	// BackupPath 备份目录路径（仅再次部署时有值）。
	BackupPath string
	// Plan 文件操作计划（DryRun 时用于展示）。
	Plan []FilePlan
}

// Deploy 执行配置部署：将模板目录复制到目标项目的 .claude/ 下。
func Deploy(opts Options) (*Result, error) {
	// 1. 校验模板
	if err := ValidateTemplate(opts.TemplateDir); err != nil {
		return nil, fmt.Errorf("模板校验失败: %w", err)
	}

	// 2. 校验目标
	if err := ValidateTarget(opts.TargetPath); err != nil {
		return nil, fmt.Errorf("目标校验失败: %w", err)
	}

	// 3. 检测首次/再次部署
	targetClaudeDir := filepath.Join(opts.TargetPath, ".claude")
	_, err := os.Stat(targetClaudeDir)
	isFirstDeploy := os.IsNotExist(err)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("检测目标 .claude/ 失败: %w", err)
	}

	// 4. 构建文件操作计划
	plans, err := BuildFilter(opts.TemplateDir, targetClaudeDir, isFirstDeploy)
	if err != nil {
		return nil, fmt.Errorf("构建文件计划失败: %w", err)
	}

	result := &Result{Plan: plans}

	// 5. DryRun 模式只输出计划
	if opts.DryRun {
		for range plans {
			result.FilesSkipped++ // dry-run 全部算跳过
		}
		return result, nil
	}

	// 6. 确保目标 .claude/ 目录存在
	if err := os.MkdirAll(targetClaudeDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建目标 .claude/ 目录失败: %w", err)
	}

	// 7. 再次部署：加锁 + 备份
	var lock *DeployLock
	if !isFirstDeploy {
		lock = NewDeployLock(opts.TargetPath)
		if err := lock.Acquire(); err != nil {
			return nil, fmt.Errorf("获取部署锁失败: %w", err)
		}
		defer lock.Release()

		backupPath, err := CreateBackup(targetClaudeDir)
		if err != nil {
			return nil, fmt.Errorf("创建备份失败: %w", err)
		}
		result.BackupPath = backupPath
	}

	// 8. 构建需要复制的文件集合
	copySet := make(map[string]bool)
	for _, p := range plans {
		if p.Action == Copy {
			copySet[p.RelPath] = true
		} else {
			result.FilesSkipped++
		}
	}

	// 9. 执行过滤复制
	shouldCopy := func(relPath string) bool {
		return copySet[relPath]
	}

	if err := CopyDirFiltered(targetClaudeDir, opts.TemplateDir, shouldCopy); err != nil {
		return result, fmt.Errorf("复制文件失败: %w", err)
	}

	result.FilesCopied = len(copySet)

	return result, nil
}
