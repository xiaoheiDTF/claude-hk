package deploy

import (
	"fmt"
	"path/filepath"
	"time"
)

// CreateBackup 将已有的 .claude/ 目录完整备份到 .claude.backup.{timestamp}/。
// backupPath 返回备份目录的路径。
func CreateBackup(claudeDir string) (string, error) {
	parentDir := filepath.Dir(claudeDir)
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(parentDir, ".claude.backup."+timestamp)

	// 复制整个目录（包括运行时文件，保证备份完整）
	if err := CopyDir(backupPath, claudeDir); err != nil {
		return "", fmt.Errorf("创建备份失败: %w", err)
	}

	return backupPath, nil
}
