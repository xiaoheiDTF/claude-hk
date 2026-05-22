package proxy_test

import (
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/proxy"
)

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
		// Stripped prefix variants
		{"/responses", true},
		{"/chat/completions", true},
		{"/completions", true},
		{"/models", true},
		// Google Gemini
		{"/v1beta/models", true},
		{"/v1/models/gemini-pro", true},
		// Kimi
		{"/coding/v1/messages", true},
		// OpenAI compatible relay
		{"/anthropic/v1/messages", true},
		// Blocked paths
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
