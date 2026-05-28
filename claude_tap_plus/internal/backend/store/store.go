package store

import "context"

type IssueStore interface {
	CheckIssues(ctx context.Context, repo string, numbers []int) ([]IssueCheckResult, error)
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
