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

// --- Session response types ---

// SessionListResponse 是会话列表的响应体。
type SessionListResponse struct {
	Sessions []SessionListItem `json:"sessions"` // 会话列表
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
