// Package proxy 提供 HTTP 代理功能，包括请求转发、SSE 流式处理与 Trace 记录。
package proxy

import "strings"

// AllowedPathPrefixes 定义代理将转发的 API 路径前缀白名单。
// 未知路径（如扫描器、爬虫请求）将被拒绝并返回 404。
var AllowedPathPrefixes = []string{
	// Claude Code IDE 连接（VS Code / JetBrains 扩展）
	"/ide",
	// Anthropic API
	"/v1/messages",
	"/v1/complete",
	// OpenAI API
	"/v1/responses",
	"/v1/chat/completions",
	"/v1/completions",
	"/v1/models",
	"/v1/embeddings",
	"/v1/files",
	// OpenAI Responses API（strip_path_prefix 移除 /v1 后的路径）
	"/responses",
	"/chat/completions",
	"/completions",
	"/models",
	// Google Gemini API
	"/v1beta/",
	"/v1/",
	// Kimi API
	"/coding/v1/",
	// OpenAI 兼容中继端点
	"/anthropic",
}

// IsAllowedPath 判断请求路径是否在白名单中，决定是否转发。
// 未知路径会被拒绝并返回 404。
func IsAllowedPath(path string) bool {
	for _, prefix := range AllowedPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
