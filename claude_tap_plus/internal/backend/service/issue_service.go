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

func (svc *IssueService) Claim(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*store.ClaimResult, error) {
	return svc.store.ClaimIssue(ctx, repo, number, sessionID, issueTitle)
}

func (svc *IssueService) UpdateStatus(ctx context.Context, repo string, number int, sessionID string, newStatus string) (*store.UpdateStatusResult, error) {
	return svc.store.UpdateIssueStatus(ctx, repo, number, sessionID, newStatus)
}

func (svc *IssueService) Release(ctx context.Context, repo string, number int, sessionID string) (bool, error) {
	return svc.store.ReleaseIssue(ctx, repo, number, sessionID)
}

func (svc *IssueService) ReleaseSession(ctx context.Context, sessionID string) ([]int, error) {
	return svc.store.ReleaseSessionIssues(ctx, sessionID)
}
