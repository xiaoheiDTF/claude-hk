// Package service 提供后端业务逻辑层。
package service

import (
	"bufio"
	"context"
	"os"
	"path/filepath"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// TraceFileInfo 表示 Trace 文件元数据。
type TraceFileInfo struct {
	Path      string // 文件路径
	SizeBytes int64  // 文件大小（字节）
	LineCount int    // 行数
	Date      string // 日期（从父目录名提取）
	Filename  string // 文件名
}

// TraceService 处理 Trace 文件相关的业务逻辑。
type TraceService struct {
	sessionStore store.SessionStore
}

// NewTraceService 创建 TraceService 实例。
func NewTraceService(s store.SessionStore) *TraceService {
	return &TraceService{sessionStore: s}
}

// GetSessionTraces 获取指定会话的所有 trace 文件信息。
func (svc *TraceService) GetSessionTraces(ctx context.Context, sessionID string) ([]TraceFileInfo, error) {
	logger.Debug("svc.trace", "GetSessionTraces: session=%s", sessionID)

	sess, err := svc.sessionStore.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, store.ErrSessionNotFound
	}

	if sess.LocalTracePath == "" {
		return []TraceFileInfo{}, nil
	}

	// 获取 trace 文件信息
	info, err := getTraceFileInfo(sess.LocalTracePath)
	if err != nil {
		logger.Warn("svc.trace", "get trace info failed: %v", err)
		return []TraceFileInfo{}, nil
	}

	return []TraceFileInfo{*info}, nil
}

// getTraceFileInfo 获取单个 trace 文件的元数据。
func getTraceFileInfo(path string) (*TraceFileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// 计算行数
	lineCount, err := countLines(path)
	if err != nil {
		lineCount = 0
	}

	dir := filepath.Dir(path)
	date := filepath.Base(dir)
	filename := filepath.Base(path)

	return &TraceFileInfo{
		Path:      path,
		SizeBytes: stat.Size(),
		LineCount: lineCount,
		Date:      date,
		Filename:  filename,
	}, nil
}

// countLines 计算文件行数。
func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}
