// Package domain 定义核心业务实体和枚举类型。
package domain

import "time"

// IssueStatus 表示 Issue 的流转状态。
type IssueStatus string

const (
	IssueIdle       IssueStatus = "idle"        // 空闲，无人领取
	IssueClaimed    IssueStatus = "claimed"     // 已被某 session 领取
	IssueFixing     IssueStatus = "fixing"      // 正在开发中
	IssueReadyForPR IssueStatus = "ready-for-pr" // 开发完成，等待提 PR
	IssuePRCreated  IssueStatus = "pr-created"  // PR 已创建
	IssueTesting    IssueStatus = "testing"     // 测试中
	IssueReviewing  IssueStatus = "reviewing"   // 审核中
	IssueMerged     IssueStatus = "merged"      // 已合并（终态）
	IssueRejected   IssueStatus = "rejected"    // 被打回
)

// IssueClaim 对应 issue_claims 表，存储 Issue 领取关系和状态流转。
// 只存"哪个 session 领取了哪个 issue"及其状态，不存 issue 完整内容（内容在 GitHub 上）。
// UNIQUE(repo_full_name, issue_number)
type IssueClaim struct {
	ID           int64       `json:"id"`
	RepoFullName string      `json:"repo_full_name"` // 如 xiaoheiDTF/claude-hk，来自 gh repo view
	IssueNumber  int         `json:"issue_number"`   // GitHub issue 编号
	IssueTitle   string      `json:"issue_title"`    // issue 标题（缓存，方便展示）
	Status       IssueStatus `json:"status"`         // 见 IssueStatus 枚举，默认 idle
	SessionID    string      `json:"session_id"`     // 领取者的 session_id，idle 时为空
	ClaimedAt    *time.Time  `json:"claimed_at"`     // 领取时间，idle 时为 null
	UpdatedAt    time.Time   `json:"updated_at"`     // 最后更新时间
}
