package store

import "context"

type ClaimResult struct {
	Success   bool
	Status    string
	ClaimedBy *string
	ClaimedAt *string
}

type IssueStore interface {
	CheckIssues(ctx context.Context, repo string, numbers []int) ([]IssueCheckResult, error)
	ClaimIssue(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*ClaimResult, error)
	ReleaseIssue(ctx context.Context, repo string, number int, sessionID string) (bool, error)
	ReleaseSessionIssues(ctx context.Context, sessionID string) ([]int, error)
}

type IssueCheckResult struct {
	Number    int
	Status    string
	SessionID *string
	ClaimedAt *string
}

type Store interface {
	Issues() IssueStore
	Close() error
}
