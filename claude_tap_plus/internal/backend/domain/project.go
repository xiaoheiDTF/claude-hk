package domain

import "time"

// Project 对应 projects 表，记录 Claude Code 工作过的项目。
// project_slug 从 transcript_path 中解析得到（Claude Code 内置的项目标识，由 cwd 路径分隔符替换为 - 得到）。
type Project struct {
	ID          int64     `json:"id"`
	ProjectSlug string    `json:"project_slug"` // 从 transcript_path 解析，UNIQUE
	ProjectCwd  string    `json:"project_cwd"`  // SessionStart hook 的 cwd 字段
	FirstSeenAt time.Time `json:"first_seen_at"` // 项目首次出现时间
	LastSeenAt  time.Time `json:"last_seen_at"`  // 每次该项目有新会话时更新
}
