// Package api 提供后端 HTTP API 的处理函数和路由定义。
package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// --- Issue response types ---

// CheckIssuesResponse 是检查 Issue 状态的响应体。
type CheckIssuesResponse struct {
	Issues []IssueStatusItem `json:"issues"` // Issue 状态列表
}

// IssueStatusItem 表示单个 Issue 的状态信息。
type IssueStatusItem struct {
	Number    int     `json:"number"`     // Issue 编号
	Status    string  `json:"status"`     // 当前状态
	SessionID *string `json:"session_id"` // 领取该 Issue 的会话 ID（空闲时为 null）
	ClaimedAt *string `json:"claimed_at"` // 领取时间（空闲时为 null）
}

// ReleaseIssueResponse 是释放 Issue 的响应体。
type ReleaseIssueResponse struct {
	Success  bool   `json:"success"`           // 是否成功
	Released *bool  `json:"released,omitempty"` // 是否已释放
	Error    string `json:"error,omitempty"`    // 错误信息（如果有）
}

// ClaimIssueResponse 是领取 Issue 的响应体。
type ClaimIssueResponse struct {
	Success   bool    `json:"success"`             // 是否成功
	Status    string  `json:"status,omitempty"`    // 领取后的状态
	ClaimedAt *string `json:"claimed_at,omitempty"` // 领取时间
	Error     string  `json:"error,omitempty"`      // 错误信息
	ClaimedBy *string `json:"claimed_by,omitempty"` // 已被谁领取
	Message   string  `json:"message,omitempty"`    // 提示信息
}

// ReleaseSessionResponse 是释放会话下所有 Issue 的响应体。
type ReleaseSessionResponse struct {
	Released []int `json:"released"` // 被释放的 Issue 编号列表
	Count    int   `json:"count"`    // 释放数量
}

// UpdateStatusResponse 是更新 Issue 状态的响应体。
type UpdateStatusResponse struct {
	Success        bool   `json:"success"`                  // 是否成功
	PreviousStatus string `json:"previous_status,omitempty"` // 更新前的状态
	NewStatus      string `json:"new_status,omitempty"`      // 更新后的状态
	Error          string `json:"error,omitempty"`           // 错误信息
}

// IssuesListResponse 是 Issue 列表的响应体。
type IssuesListResponse struct {
	Issues     []IssueListItem `json:"issues"`      // Issue 列表
	Total      int             `json:"total"`       // 总数量
	Page       int             `json:"page"`        // 当前页码
	PageSize   int             `json:"page_size"`   // 每页大小
	TotalPages int             `json:"total_pages"` // 总页数
}

// IssueListItem 是 Issue 列表中的单个条目。
type IssueListItem struct {
	ID           int64      `json:"id"`
	RepoFullName string     `json:"repo_full_name"`
	IssueNumber  int        `json:"issue_number"`
	IssueTitle   *string    `json:"issue_title"`
	Status       string     `json:"status"`
	SessionID    *string    `json:"session_id"`
	ClaimedAt    *time.Time `json:"claimed_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// --- Session response types ---

// SessionListResponse 是会话列表的响应体。
type SessionListResponse struct {
	Sessions []SessionListItem `json:"sessions"` // 会话列表
}

// SessionIssuesResponse 是会话关联 Issue 的响应体。
type SessionIssuesResponse struct {
	Issues []IssueListItem `json:"issues"` // 会话关联的 Issue 列表
}

// SessionTokensResponse 是会话 Token 统计的响应体。
type SessionTokensResponse struct {
	SessionID    string `json:"session_id"`
	APICalls     int    `json:"api_calls"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	CacheRead    int    `json:"cache_read"`
	CacheCreate  int    `json:"cache_create"`
	TotalTokens  int    `json:"total_tokens"`
}

// SessionTracesResponse 是会话 Trace 文件列表的响应体。
type SessionTracesResponse struct {
	SessionID string      `json:"session_id"`
	Traces    []TraceItem `json:"traces"`
}

// TraceItem 是单个 Trace 文件的信息。
type TraceItem struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	LineCount int    `json:"line_count"`
	Date      string `json:"date"`
	Filename  string `json:"filename"`
}

// SessionListItem 是会话列表中的单个条目。
type SessionListItem struct {
	SessionID    string     `json:"session_id"`    // 会话 ID
	MachineID    string     `json:"machine_id"`    // 机器 ID
	ProjectSlug  string     `json:"project_slug"`  // 项目标识
	Status       string     `json:"status"`        // 会话状态
	RegisteredAt time.Time  `json:"registered_at"` // 注册时间
	ClosedAt     *time.Time `json:"closed_at,omitempty"` // 关闭时间（未关闭时为 null）
}

// SessionDetail 是会话详情的响应体。
type SessionDetail struct {
	SessionID      string     `json:"session_id"`      // 会话 ID
	MachineID      string     `json:"machine_id"`      // 机器 ID
	OS             string     `json:"os"`              // 操作系统
	ProjectSlug    string     `json:"project_slug"`    // 项目标识
	ProjectCwd     string     `json:"project_cwd"`     // 项目工作目录
	TranscriptPath string     `json:"transcript_path"` // 对话记录路径
	LocalTracePath string     `json:"local_trace_path"` // 本地跟踪路径
	Model          string     `json:"model"`           // 使用的模型
	Source         string     `json:"source"`          // 来源
	Status         string     `json:"status"`          // 状态
	RegisteredAt   time.Time  `json:"registered_at"`   // 注册时间
	ClosedAt       *time.Time `json:"closed_at,omitempty"` // 关闭时间
	CloseReason    string     `json:"close_reason,omitempty"` // 关闭原因
}

// --- Machine response types ---

// MachinesListResponse 是机器列表的响应体。
type MachinesListResponse struct {
	Machines []MachineListItem `json:"machines"` // 机器列表
}

// MachineListItem 是机器列表中的单个条目。
type MachineListItem struct {
	ID          int64      `json:"id"`
	MachineID   string     `json:"machine_id"`
	OS          string     `json:"os"`
	Hostname    string     `json:"hostname"`
	Username    string     `json:"username"`
	FirstSeenAt time.Time  `json:"first_seen_at"`
	LastSeenAt  *time.Time `json:"last_seen_at"`
}

// --- Project response types ---

// ProjectsListResponse 是项目列表的响应体。
type ProjectsListResponse struct {
	Projects []ProjectListItem `json:"projects"` // 项目列表
}

// ProjectListItem 是项目列表中的单个条目。
type ProjectListItem struct {
	ID          int64     `json:"id"`
	ProjectSlug string    `json:"project_slug"`
	ProjectCwd  string    `json:"project_cwd"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

// --- Config response types ---

// ConfigResponse 是配置查询/更新的响应体。
type ConfigResponse struct {
	Config map[string]interface{} `json:"config"` // 配置项键值对
}

// --- Log response types ---

// LogsResponse 是日志查询的响应体。
type LogsResponse struct {
	Logs []LogItem `json:"logs"` // 日志条目列表
}

// LogItem 是单条日志记录。
type LogItem struct {
	Timestamp string `json:"timestamp"` // 时间戳，如 "2026-06-01 20:56:28.000"
	Level     string `json:"level"`     // 日志级别：DEBUG/INFO/WARN/ERROR
	Source    string `json:"source"`    // 来源模块
	Message   string `json:"message"`   // 日志内容
}

// --- Proxy response types ---

// ProxiesResponse 是代理列表的响应体。
type ProxiesResponse struct {
	Proxies []ProxyItem `json:"proxies"` // 代理列表
	Total   int         `json:"total"`   // 总数量
}

// ProxyItem 是代理列表中的单个条目。
type ProxyItem struct {
	ProxyID      string     `json:"proxy_id"`                // 代理唯一 ID
	ProjectSlug  string     `json:"project_slug"`            // 项目标识
	Status       string     `json:"status"`                  // 状态：active/offline
	RegisteredAt time.Time  `json:"registered_at"`           // 注册时间
	LastPingAt   *time.Time `json:"last_ping_at,omitempty"`  // 最后心跳时间
}

// --- Status response types ---

// StatusResponse 是系统状态的响应体。
type StatusResponse struct {
	Status        string     `json:"status"`         // 系统状态：healthy
	Version       string     `json:"version"`        // 版本号
	UptimeSeconds int64      `json:"uptime_seconds"` // 运行时间（秒）
	Stats         SystemStats `json:"stats"`          // 统计信息
	Timestamp     time.Time  `json:"timestamp"`      // 时间戳
}

// SystemStats 是系统统计信息的响应结构。
type SystemStats struct {
	ActiveSessions int64 `json:"active_sessions"` // 活跃会话数
	ActiveProxies  int64 `json:"active_proxies"`  // 活跃代理数
	PendingIssues  int64 `json:"pending_issues"`  // 待处理 Issue 数
	TotalMachines  int64 `json:"total_machines"`  // 机器总数
	TotalProjects  int64 `json:"total_projects"`  // 项目总数
}

// --- Shared helpers ---

// APIError 是统一的 API 错误结构体。
type APIError struct {
	Code    string `json:"error"`   // 错误码
	Message string `json:"message"` // 错误描述
}

// writeJSON 向响应中写入 JSON 数据，设置 Content-Type 为 application/json。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError 向响应中写入统一的 API 错误信息。
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIError{Code: code, Message: message})
}
