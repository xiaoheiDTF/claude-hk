package store

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSessionExists   = errors.New("session already exists")
	ErrSessionNotFound = errors.New("session not found or already closed")
)

// --- Issue types ---

type ClaimResult struct {
	Success   bool
	Status    string
	ClaimedBy *string
	ClaimedAt *string
}

type UpdateStatusResult struct {
	PreviousStatus string
	NewStatus      string
	Updated        bool
}

type IssueStore interface {
	CheckIssues(ctx context.Context, repo string, numbers []int) ([]IssueCheckResult, error)
	ClaimIssue(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*ClaimResult, error)
	UpdateIssueStatus(ctx context.Context, repo string, number int, sessionID string, newStatus string) (*UpdateStatusResult, error)
	ReleaseIssue(ctx context.Context, repo string, number int, sessionID string) (bool, error)
	ReleaseSessionIssues(ctx context.Context, sessionID string) ([]int, error)
}

type IssueCheckResult struct {
	Number    int
	Status    string
	SessionID *string
	ClaimedAt *string
}

// --- Session types ---

type Session struct {
	SessionID      string
	MachineID      string
	OS             string
	ProjectSlug    string
	ProjectCwd     string
	TranscriptPath string
	LocalTracePath string
	Model          string
	Source         string
	Status         string
	RegisteredAt   time.Time
	ClosedAt       *time.Time
	CloseReason    string
}

type SessionFilter struct {
	MachineID   *string
	ProjectSlug *string
	Status      *string
}

type SessionStore interface {
	RegisterSession(ctx context.Context, s Session) error
	CloseSession(ctx context.Context, sessionID string, reason string) error
	ListSessions(ctx context.Context, filter SessionFilter) ([]Session, error)
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	CleanupTimedOut(ctx context.Context) (int, error)
}

// --- Store aggregate ---

type Store interface {
	Issues() IssueStore
	Sessions() SessionStore
	Close() error
}
