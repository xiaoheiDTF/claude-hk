// Package service 提供后端业务逻辑层。
package service

import (
	"context"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// IssueService 处理 Issue 相关的业务逻辑。
type IssueService struct {
	store store.IssueStore // Issue 数据存储接口
}

// NewIssueService 创建 IssueService 实例。
func NewIssueService(s store.IssueStore) *IssueService {
	return &IssueService{store: s}
}

// Check 检查指定仓库中多个 Issue 的当前状态。
func (svc *IssueService) Check(ctx context.Context, repo string, numbers []int) ([]store.IssueCheckResult, error) {
	logger.Debug("svc.issue", "Check: repo=%s numbers=%v", repo, numbers)
	return svc.store.CheckIssues(ctx, repo, numbers)
}

// Claim 尝试用指定 session 领取某个 Issue。
// 返回领取结果：成功时包含状态和时间，失败时包含当前持有者信息。
func (svc *IssueService) Claim(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*store.ClaimResult, error) {
	logger.Debug("svc.issue", "Claim: repo=%s #%d session=%s", repo, number, sessionID)
	return svc.store.ClaimIssue(ctx, repo, number, sessionID, issueTitle)
}

// UpdateStatus 更新指定 Issue 的状态（仅当前持有者可更新）。
func (svc *IssueService) UpdateStatus(ctx context.Context, repo string, number int, sessionID string, newStatus string) (*store.UpdateStatusResult, error) {
	logger.Debug("svc.issue", "UpdateStatus: repo=%s #%d session=%s -> %s", repo, number, sessionID, newStatus)
	return svc.store.UpdateIssueStatus(ctx, repo, number, sessionID, newStatus)
}

// Release 释放指定 session 持有的某个 Issue（仅限非终态）。
// 返回是否成功释放。
func (svc *IssueService) Release(ctx context.Context, repo string, number int, sessionID string) (bool, error) {
	logger.Debug("svc.issue", "Release: repo=%s #%d session=%s", repo, number, sessionID)
	return svc.store.ReleaseIssue(ctx, repo, number, sessionID)
}

// ReleaseSession 释放指定 session 持有的所有非终态 Issue。
// 返回被释放的 Issue 编号列表。
func (svc *IssueService) ReleaseSession(ctx context.Context, sessionID string) ([]int, error) {
	logger.Debug("svc.issue", "ReleaseSession: session=%s", sessionID)
	return svc.store.ReleaseSessionIssues(ctx, sessionID)
}
