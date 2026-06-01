// Package service 提供后端业务逻辑层。
package service

import (
	"bufio"
	"context"
	"encoding/json"
	"os"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/domain"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// TokenService 处理 Token 统计相关的业务逻辑。
type TokenService struct {
	sessionStore store.SessionStore
}

// NewTokenService 创建 TokenService 实例。
func NewTokenService(s store.SessionStore) *TokenService {
	return &TokenService{sessionStore: s}
}

// GetSessionTokens 获取指定会话的 Token 统计。
func (svc *TokenService) GetSessionTokens(ctx context.Context, sessionID string) (*domain.TokenStats, error) {
	logger.Debug("svc.token", "GetSessionTokens: session=%s", sessionID)

	// 获取会话信息
	sess, err := svc.sessionStore.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, store.ErrSessionNotFound
	}

	// 如果无 trace 路径，返回零值
	if sess.LocalTracePath == "" {
		return &domain.TokenStats{}, nil
	}

	// 解析 trace 文件
	stats, err := parseTraceFile(sess.LocalTracePath)
	if err != nil {
		logger.Warn("svc.token", "parse trace failed: %v", err)
		return &domain.TokenStats{}, nil
	}

	return stats, nil
}

// traceRecord 是 trace JSONL 文件中单条记录的解析结构。
type traceRecord struct {
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		CacheRead    int `json:"cache_read,omitempty"`
		CacheCreate  int `json:"cache_create,omitempty"`
	} `json:"usage"`
}

// parseTraceFile 解析 JSONL trace 文件，汇总 Token 统计。
func parseTraceFile(path string) (*domain.TokenStats, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var stats domain.TokenStats
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var record traceRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue // 跳过解析失败的行
		}

		stats.APICalls++
		stats.InputTokens += record.Usage.InputTokens
		stats.OutputTokens += record.Usage.OutputTokens
		stats.CacheRead += record.Usage.CacheRead
		stats.CacheCreate += record.Usage.CacheCreate
	}

	return &stats, scanner.Err()
}
