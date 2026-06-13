package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// lockDirName 部署锁目录名。
	lockDirName = ".claude.deploy.lock"
	// lockMaxRetries 最大重试次数。
	lockMaxRetries = 10
	// lockRetryInterval 重试间隔。
	lockRetryInterval = 500 * time.Millisecond
	// lockStaleThreshold 锁过期时间。
	lockStaleThreshold = 5 * time.Minute
)

// DeployLock 部署锁，使用 mkdir 原子操作实现并发安全。
type DeployLock struct {
	lockPath string
	acquired bool
}

// NewDeployLock 创建部署锁实例。
func NewDeployLock(targetPath string) *DeployLock {
	return &DeployLock{
		lockPath: filepath.Join(targetPath, lockDirName),
	}
}

// Acquire 获取部署锁。如果锁已被持有，会重试直到超时。
// 超过 5 分钟的脏锁会被自动清理。
func (l *DeployLock) Acquire() error {
	for i := 0; i < lockMaxRetries; i++ {
		// 尝试创建锁目录（原子操作）
		err := os.Mkdir(l.lockPath, 0o755)
		if err == nil {
			// 成功获取锁，写入 PID 和时间戳
			l.writeLockInfo()
			l.acquired = true
			return nil
		}

		// 目录已存在，检查是否是脏锁
		if os.IsExist(err) {
			if l.isStale() {
				// 脏锁，强制清理后重试
				os.RemoveAll(l.lockPath)
				continue
			}
		} else {
			return fmt.Errorf("创建部署锁失败: %w", err)
		}

		time.Sleep(lockRetryInterval)
	}

	return fmt.Errorf("部署锁获取超时: 另一个部署进程可能正在运行 (%s)", l.lockPath)
}

// Release 释放部署锁。
func (l *DeployLock) Release() error {
	if !l.acquired {
		return nil
	}
	l.acquired = false
	return os.RemoveAll(l.lockPath)
}

// writeLockInfo 写入锁信息（PID + 时间戳）。
func (l *DeployLock) writeLockInfo() {
	infoFile := filepath.Join(l.lockPath, "info")
	content := fmt.Sprintf("pid=%d\ntime=%s\n", os.Getpid(), time.Now().Format(time.RFC3339))
	os.WriteFile(infoFile, []byte(content), 0o644)
}

// isStale 检查锁是否过期。
func (l *DeployLock) isStale() bool {
	infoFile := filepath.Join(l.lockPath, "info")
	info, err := os.Stat(infoFile)
	if err != nil {
		// 锁信息文件不存在，检查目录本身的修改时间
		dirInfo, err := os.Stat(l.lockPath)
		if err != nil {
			return false
		}
		return time.Since(dirInfo.ModTime()) > lockStaleThreshold
	}
	return time.Since(info.ModTime()) > lockStaleThreshold
}
