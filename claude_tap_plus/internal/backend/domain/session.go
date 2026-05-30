// Package domain 定义核心业务实体和枚举类型。
package domain

import "time"

// SessionStatus 表示会话的状态。
type SessionStatus string

const (
	SessionActive SessionStatus = "active" // 活跃会话
	SessionClosed SessionStatus = "closed" // 已关闭
)

// Session 对应 sessions 表，记录每次 Claude Code 会话的元数据。
// 后端只存元数据，消息内容存储在本地 JSONL 文件中。
type Session struct {
	ID             int64         `json:"id"`
	SessionID      string        `json:"session_id"`      // UUID，SessionStart hook 传入，UNIQUE
	MachineID      string        `json:"machine_id"`       // whoami@hostname
	OS             string        `json:"os"`               // windows/linux/macos
	ProjectSlug    string        `json:"project_slug"`     // 从 transcript_path 解析
	ProjectCwd     string        `json:"project_cwd"`      // SessionStart hook 的 cwd 字段
	TranscriptPath string        `json:"transcript_path"`  // Claude Code hook 提供的 transcript_path，便于溯源
	LocalTracePath string        `json:"local_trace_path"` // proxy 本地 trace 文件路径，由代理首次拦截该 session 的 API 调用时构造写入
	Model          string        `json:"model"`            // SessionStart hook 的 model 字段
	Source         string        `json:"source"`           // startup/resume
	Status         SessionStatus `json:"status"`           // active / closed，默认 active
	RegisteredAt   time.Time     `json:"registered_at"`    // SessionStart 注册时间
	ClosedAt       *time.Time    `json:"closed_at"`        // SessionEnd 注销时间
	CloseReason    string        `json:"close_reason"`     // SessionEnd 的 reason 字段（如 prompt_input_exit）
}
