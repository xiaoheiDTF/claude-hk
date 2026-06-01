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

// IssueFilter 是 Issue 列表的过滤条件。
type IssueFilter struct {
	RepoFullName *string // 按仓库过滤
	Status       *string // 按状态过滤
	SessionID    *string // 按 session 过滤
	Page         int     // 页码，默认 1
	PageSize     int     // 每页大小，默认 20
}

// IssueListItem 是 Issue 列表中的单个条目。
type IssueListItem struct {
	ID           int64      // 记录 ID
	RepoFullName string     // 仓库全名
	IssueNumber  int        // Issue 编号
	IssueTitle   *string    // Issue 标题（可为 NULL）
	Status       string     // 当前状态
	SessionID    *string    // 领取者的 session ID
	ClaimedAt    *time.Time // 领取时间
	UpdatedAt    time.Time  // 更新时间
}

// IssueStore 定义 Issue 数据存储的接口。
type IssueStore interface {
	CheckIssues(ctx context.Context, repo string, numbers []int) ([]IssueCheckResult, error)
	ClaimIssue(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*ClaimResult, error)
	UpdateIssueStatus(ctx context.Context, repo string, number int, sessionID string, newStatus string) (*UpdateStatusResult, error)
	ReleaseIssue(ctx context.Context, repo string, number int, sessionID string) (bool, error)
	ReleaseSessionIssues(ctx context.Context, sessionID string) ([]int, error)
	ListIssues(ctx context.Context, filter IssueFilter) ([]IssueListItem, int, error)
	ListIssuesBySession(ctx context.Context, sessionID string) ([]IssueListItem, error)
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

// --- Machine types ---

// Machine 表示机器实体（对应数据库 machines 表）。
type Machine struct {
	ID          int64      // 记录 ID
	MachineID   string     // 机器标识，格式 username@hostname
	OS          string     // 操作系统
	Hostname    string     // 主机名
	Username    string     // 用户名
	FirstSeenAt time.Time  // 首次发现时间
	LastSeenAt  *time.Time // 最后发现时间
}

// MachineFilter 是机器列表的过滤条件。
type MachineFilter struct {
	OS       *string // 按操作系统过滤
	Hostname *string // 按主机名过滤
}

// MachineStore 定义 Machine 数据存储的接口。
type MachineStore interface {
	ListMachines(ctx context.Context, filter MachineFilter) ([]Machine, error)
	GetMachine(ctx context.Context, machineID string) (*Machine, error)
}

// --- Project types ---

// Project 表示项目实体（对应数据库 projects 表）。
type Project struct {
	ID          int64     // 记录 ID
	ProjectSlug string    // 项目标识
	ProjectCwd  string    // 项目工作目录
	FirstSeenAt time.Time // 首次发现时间
	LastSeenAt  time.Time // 最后发现时间
}

// ProjectStore 定义 Project 数据存储的接口。
type ProjectStore interface {
	ListProjects(ctx context.Context) ([]Project, error)
	GetProject(ctx context.Context, projectSlug string) (*Project, error)
}

// --- Config types ---

// ConfigStore 定义系统配置存储的接口。
type ConfigStore interface {
	GetConfig(ctx context.Context) (map[string]interface{}, error)
	UpdateConfig(ctx context.Context, updates map[string]interface{}) error
}

// --- Store aggregate ---

// Store 是统一的数据存储聚合接口。
type Store interface {
	Issues() IssueStore     // 返回 Issue 存储
	Sessions() SessionStore // 返回 Session 存储
	Machines() MachineStore // 返回 Machine 存储
	Projects() ProjectStore // 返回 Project 存储
	Configs() ConfigStore   // 返回 Config 存储
	Close() error           // 关闭存储连接
}
