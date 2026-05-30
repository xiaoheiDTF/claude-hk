// Package service 提供后端业务逻辑层。
package service

import (
	"context"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// CleanupService 负责清理超时会话的服务。
type CleanupService struct {
	sessionStore store.SessionStore // 会话数据存储接口
}

// NewCleanupService 创建 CleanupService 实例。
func NewCleanupService(s store.SessionStore) *CleanupService {
	return &CleanupService{sessionStore: s}
}

// CleanupTimedOutSessions 清理注册超过 24 小时仍处于 active 状态的会话。
// 将这些超时会话标记为已关闭，并记录清理数量。
func (svc *CleanupService) CleanupTimedOutSessions(ctx context.Context) {
	logger.Debug("svc.cleanup", "starting timed-out session cleanup")
	n, err := svc.sessionStore.CleanupTimedOut(ctx)
	if err != nil {
		logger.Error("svc.cleanup", "cleanup failed: %v", err)
		return
	}
	if n > 0 {
		logger.Info("svc.cleanup", "cleaned up %d timed-out sessions", n)
	}
}
