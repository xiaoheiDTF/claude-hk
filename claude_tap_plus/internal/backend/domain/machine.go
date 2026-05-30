// Package domain 定义核心业务实体和枚举类型。
package domain

import "time"

// Machine 对应 machines 表，记录使用 claude-tap-plus 的机器信息。
// machine_id 格式为 whoami@hostname，从请求中的 machine_id 按 @ 分割解析 username 和 hostname。
type Machine struct {
	ID          int64     `json:"id"`
	MachineID   string    `json:"machine_id"`   // whoami@hostname，UNIQUE
	OS          string    `json:"os"`            // windows/linux/macos，来自 platform.sh
	Hostname    string    `json:"hostname"`      // 从 machine_id 按 @ 分割取后半部分
	Username    string    `json:"username"`      // 从 machine_id 按 @ 分割取前半部分
	FirstSeenAt time.Time `json:"first_seen_at"` // 首次注册时间
	LastSeenAt  time.Time `json:"last_seen_at"`  // 每次 SessionStart 更新
}
