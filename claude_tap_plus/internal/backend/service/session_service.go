// Package service 提供后端业务逻辑层。
package service

import (
	"context"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// SessionService 处理 Session 相关的业务逻辑。
type SessionService struct {
	store store.SessionStore // 会话数据存储接口
}

// NewSessionService 创建 SessionService 实例。
func NewSessionService(s store.SessionStore) *SessionService {
	return &SessionService{store: s}
}

// Register 注册一个新会话，同时记录机器和项目信息。
func (svc *SessionService) Register(ctx context.Context, sess store.Session) error {
	logger.Debug("svc.session", "Register: id=%s slug=%s", sess.SessionID, sess.ProjectSlug)
	return svc.store.RegisterSession(ctx, sess)
}

// Close 关闭指定会话，标记其状态为 closed 并记录关闭原因。
func (svc *SessionService) Close(ctx context.Context, sessionID string, reason string) error {
	logger.Debug("svc.session", "Close: id=%s reason=%s", sessionID, reason)
	return svc.store.CloseSession(ctx, sessionID, reason)
}

// List 获取会话列表，支持按 machine_id、project_slug、status 过滤。
func (svc *SessionService) List(ctx context.Context, filter store.SessionFilter) ([]store.Session, error) {
	logger.Debug("svc.session", "List: filter applied")
	return svc.store.ListSessions(ctx, filter)
}

// Get 获取单个会话的详细信息。
func (svc *SessionService) Get(ctx context.Context, sessionID string) (*store.Session, error) {
	logger.Debug("svc.session", "Get: id=%s", sessionID)
	return svc.store.GetSession(ctx, sessionID)
}
