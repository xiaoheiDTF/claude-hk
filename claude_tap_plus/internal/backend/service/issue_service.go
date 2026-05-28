package service

import (
	"context"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
)

type IssueService struct {
	store store.IssueStore
}

func NewIssueService(s store.IssueStore) *IssueService {
	return &IssueService{store: s}
}

func (svc *IssueService) Check(ctx context.Context, repo string, numbers []int) ([]store.IssueCheckResult, error) {
	return svc.store.CheckIssues(ctx, repo, numbers)
}
