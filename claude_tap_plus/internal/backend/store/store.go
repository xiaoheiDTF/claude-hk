// Package store 提供数据持久化层，基于 SQLite 实现。
package store

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrSessionExists 表示会话已存在（重复注册）。
	ErrSessionExists = errors.New("session already exists")
	// ErrSessionNotFound 表示会话不存在或已被关闭。
	ErrSessionNotFound = errors.New("session not found or already closed")
)

// --- Issue types ---

// ClaimResult 表示 Issue 领取操作的结果。
type ClaimResult struct {
	Success   bool    // 是否成功领取
	Status    string  // 当前状态
	ClaimedBy *string // 被谁领取（冲突时）
	ClaimedAt *string // 领取时间
}

// UpdateStatusResult 表示 Issue 状态更新操作的结果。
type UpdateStatusResult struct {
	PreviousStatus string // 更新前的状态
	NewStatus      string // 更新后的状态
	Updated        bool   // 是否成功更新
}

// IssueStore 定义 Issue 数据存储的接口。
type IssueStore interface {
	CheckIssues(ctx context.Context, repo string, numbers []int) ([]IssueCheckResult, error)
	ClaimIssue(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*ClaimResult, error)
	UpdateIssueStatus(ctx context.Context, repo string, number int, sessionID string, newStatus string) (*UpdateStatusResult, error)
	ReleaseIssue(ctx context.Context, repo string, number int, sessionID string) (bool, error)
	ReleaseSessionIssues(ctx context.Context, sessionID string) ([]int, error)
}

// IssueCheckResult 表示单个 Issue 的状态查询结果。
type IssueCheckResult struct {
	Number    int     // Issue 编号
	Status    string  // 当前状态
	SessionID *string // 领取者 session ID
	ClaimedAt *string // 领取时间
}

// --- Session types ---

// Session 表示会话实体（对应数据库 sessions 表）。
type Session struct {
	SessionID      string     // 会话唯一 ID
	MachineID      string     // 机器标识
	OS             string     // 操作系统
	ProjectSlug    string     // 项目标识
	ProjectCwd     string     // 项目工作目录
	TranscriptPath string     // 对话记录路径
	LocalTracePath string     // 本地跟踪路径
	Model          string     // 使用的模型
	Source         string     // 来源
	Status         string     // 状态
	RegisteredAt   time.Time  // 注册时间
	ClosedAt       *time.Time // 关闭时间
	CloseReason    string     // 关闭原因
}

// SessionFilter 是会话列表的过滤条件（指针类型支持省略条件）。
type SessionFilter struct {
	MachineID   *string // 按机器 ID 过滤
	ProjectSlug *string // 按项目标识过滤
	Status      *string // 按状态过滤
}

// SessionStore 定义 Session 数据存储的接口。
type SessionStore interface {
	RegisterSession(ctx context.Context, s Session) error
	CloseSession(ctx context.Context, sessionID string, reason string) error
	ListSessions(ctx context.Context, filter SessionFilter) ([]Session, error)
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	CleanupTimedOut(ctx context.Context) (int, error)
}

// --- Store aggregate ---

// Store 是统一的数据存储聚合接口。
type Store interface {
	Issues() IssueStore   // 返回 Issue 存储
	Sessions() SessionStore // 返回 Session 存储
	Close() error         // 关闭存储连接
}
