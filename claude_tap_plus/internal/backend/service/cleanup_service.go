package service

import (
	"context"
	"log"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
)

type CleanupService struct {
	sessionStore store.SessionStore
}

func NewCleanupService(s store.SessionStore) *CleanupService {
	return &CleanupService{sessionStore: s}
}

func (svc *CleanupService) CleanupTimedOutSessions(ctx context.Context) {
	n, err := svc.sessionStore.CleanupTimedOut(ctx)
	if err != nil {
		log.Printf("cleanup timed-out sessions: %v", err)
		return
	}
	if n > 0 {
		log.Printf("cleaned up %d timed-out sessions", n)
	}
}
