// Package domain 定义核心业务实体和枚举类型。
package domain

// TokenStats 表示 Token 使用统计。
type TokenStats struct {
	APICalls     int // API 调用次数
	InputTokens  int // 输入 Token 数
	OutputTokens int // 输出 Token 数
	CacheRead    int // 缓存读取 Token 数
	CacheCreate  int // 缓存创建 Token 数
}

// Total 计算总 Token 数（input + output）。
func (t TokenStats) Total() int {
	return t.InputTokens + t.OutputTokens
}
