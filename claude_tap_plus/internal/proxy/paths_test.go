// Package proxy_test 包含代理层路径过滤的单元测试。
package proxy_test

import (
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/proxy"
)

// TestIsAllowedPath 验证：代理的路径白名单和黑名单逻辑。
// 允许的路径包括 Anthropic、OpenAI、Google Gemini、Kimi 等 API 路径；
// 被屏蔽的路径包括管理后台、robots.txt、.env 等。
func TestIsAllowedPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		// Anthropic
		{"/v1/messages", true},
		{"/v1/complete", true},
		// OpenAI
		{"/v1/responses", true},
		{"/v1/chat/completions", true},
		{"/v1/completions", true},
		{"/v1/models", true},
		{"/v1/embeddings", true},
		{"/v1/files", true},
		// 无前缀变体
		{"/responses", true},
		{"/chat/completions", true},
		{"/completions", true},
		{"/models", true},
		// Google Gemini
		{"/v1beta/models", true},
		{"/v1/models/gemini-pro", true},
		// Kimi
		{"/coding/v1/messages", true},
		// OpenAI 兼容转发
		{"/anthropic/v1/messages", true},
		// 被屏蔽的路径
		{"/admin/config", false},
		{"/robots.txt", false},
		{"/wp-admin", false},
		{"/.env", false},
		{"/", false},
		{"", false},
	}

	for _, tt := range tests {
		got := proxy.IsAllowedPath(tt.path)
		if got != tt.expected {
			t.Errorf("IsAllowedPath(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}
