// Package api 提供后端 HTTP API 的处理函数和路由定义。
package api

// CheckIssuesRequest 是检查 Issue 状态的请求体。
type CheckIssuesRequest struct {
	RepoFullName string `json:"repo_full_name"` // 仓库全名，如 "owner/repo"
	IssueNumbers []int  `json:"issue_numbers"`  // 要检查的 Issue 编号列表
}

// ClaimIssueRequest 是领取 Issue 的请求体。
type ClaimIssueRequest struct {
	RepoFullName string `json:"repo_full_name"` // 仓库全名
	IssueNumber  int    `json:"issue_number"`   // Issue 编号
	SessionID    string `json:"session_id"`     // 领取者的会话 ID
	IssueTitle   string `json:"issue_title"`    // Issue 标题（用于缓存）
}

// ReleaseIssueRequest 是释放单个 Issue 的请求体。
type ReleaseIssueRequest struct {
	RepoFullName string `json:"repo_full_name"` // 仓库全名
	IssueNumber  int    `json:"issue_number"`   // Issue 编号
	SessionID    string `json:"session_id"`     // 当前持有者的会话 ID
}

// ReleaseSessionRequest 是释放某个会话下所有 Issue 的请求体。
type ReleaseSessionRequest struct {
	SessionID string `json:"session_id"` // 要释放的会话 ID
}

// UpdateStatusRequest 是更新 Issue 状态的请求体。
type UpdateStatusRequest struct {
	RepoFullName string `json:"repo_full_name"` // 仓库全名
	IssueNumber  int    `json:"issue_number"`   // Issue 编号
	SessionID    string `json:"session_id"`     // 当前持有者的会话 ID
	Status       string `json:"status"`         // 新状态
}

// --- Session request types ---

// RegisterSessionRequest 是注册新会话的请求体。
type RegisterSessionRequest struct {
	SessionID      string `json:"session_id"`      // 会话唯一 ID（UUID）
	MachineID      string `json:"machine_id"`      // 机器标识，格式为 whoami@hostname
	OS             string `json:"os"`              // 操作系统类型
	ProjectSlug    string `json:"project_slug"`    // 项目标识
	ProjectCwd     string `json:"project_cwd"`     // 项目工作目录
	TranscriptPath string `json:"transcript_path"` // 对话记录文件路径
	LocalTracePath string `json:"local_trace_path"` // 本地跟踪文件路径
	Model          string `json:"model"`           // 使用的模型
	Source         string `json:"source"`          // 会话来源（如 startup/resume）
}

// CloseSessionRequest 是关闭会话的请求体。
type CloseSessionRequest struct {
	SessionID string `json:"session_id"` // 要关闭的会话 ID
	Reason    string `json:"reason"`     // 关闭原因
}

// TraceInitRequest 是代理 trace-init 转发请求体。
type TraceInitRequest struct {
	SessionID      string `json:"session_id"`      // 会话唯一 ID
	ProxyPID       string `json:"proxy_pid"`       // 写入 .init-pid 的代理 PID（精确路由用）
	MachineID      string `json:"machine_id"`      // 机器标识
	ProjectSlug    string `json:"project_slug"`    // 项目标识
	TranscriptPath string `json:"transcript_path"` // 对话记录路径
}
