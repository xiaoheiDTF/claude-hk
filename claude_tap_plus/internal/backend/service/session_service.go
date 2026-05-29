package service

import (
	"context"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
)

type SessionService struct {
	store store.SessionStore
}

func NewSessionService(s store.SessionStore) *SessionService {
	return &SessionService{store: s}
}

func (svc *SessionService) Register(ctx context.Context, sess store.Session) error {
	return svc.store.RegisterSession(ctx, sess)
}

func (svc *SessionService) Close(ctx context.Context, sessionID string, reason string) error {
	return svc.store.CloseSession(ctx, sessionID, reason)
}

func (svc *SessionService) List(ctx context.Context, filter store.SessionFilter) ([]store.Session, error) {
	return svc.store.ListSessions(ctx, filter)
}

func (svc *SessionService) Get(ctx context.Context, sessionID string) (*store.Session, error) {
	return svc.store.GetSession(ctx, sessionID)
}
