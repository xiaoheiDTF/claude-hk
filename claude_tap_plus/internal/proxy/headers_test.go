// Package proxy_test 包含代理层 HTTP 头处理的单元测试。
package proxy_test

import (
	"net/http"
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/proxy"
)

// TestFilterHeadersNoRedact 验证：不脱敏时，请求头原样通过。
func TestFilterHeadersNoRedact(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("Authorization", "Bearer sk-secret-12345")
	h.Set("X-Api-Key", "key-abcde123456")

	filtered := proxy.FilterHeaders(h, false)

	if filtered.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type should pass through, got %q", filtered.Get("Content-Type"))
	}
	if filtered.Get("Authorization") != "Bearer sk-secret-12345" {
		t.Errorf("Authorization should pass through without redact, got %q", filtered.Get("Authorization"))
	}
}

// TestFilterHeadersWithRedact 验证：脱敏模式下，敏感头被正确脱敏。
// Authorization 和 X-Api-Key 被前缀脱敏，Cookie 被完全掩码。
func TestFilterHeadersWithRedact(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("Authorization", "Bearer sk-secret-12345")
	h.Set("X-Api-Key", "key-abcde123456")
	h.Set("Cookie", "session=abc")

	filtered := proxy.FilterHeaders(h, true)

	if filtered.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type should pass through, got %q", filtered.Get("Content-Type"))
	}
	if filtered.Get("Authorization") != "Bearer sk-se..." {
		t.Errorf("Authorization should be prefix-redacted, got %q", filtered.Get("Authorization"))
	}
	if filtered.Get("X-Api-Key") != "key-abcde123..." {
		t.Errorf("X-Api-Key should be prefix-redacted, got %q", filtered.Get("X-Api-Key"))
	}
	if filtered.Get("Cookie") != "***" {
		t.Errorf("Cookie should be fully masked, got %q", filtered.Get("Cookie"))
	}
}

// TestFilterHeadersHopByHop 验证：逐跳头（hop-by-hop）被正确移除。
// Connection 和 Transfer-Encoding 等逐跳头不应出现在过滤后的头中。
func TestFilterHeadersHopByHop(t *testing.T) {
	h := http.Header{}
	h.Set("Connection", "keep-alive")
	h.Set("Transfer-Encoding", "chunked")
	h.Set("Content-Type", "text/plain")

	filtered := proxy.FilterHeaders(h, false)

	if filtered.Get("Connection") != "" {
		t.Error("Connection should be removed (hop-by-hop)")
	}
	if filtered.Get("Transfer-Encoding") != "" {
		t.Error("Transfer-Encoding should be removed (hop-by-hop)")
	}
	if filtered.Get("Content-Type") != "text/plain" {
		t.Error("Content-Type should remain")
	}
}

// TestHeadersToMap 验证：HTTP 头正确转换为 map，多值头取第一个值。
func TestHeadersToMap(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Add("X-Custom", "val1")
	h.Add("X-Custom", "val2")

	m := proxy.HeadersToMap(h)

	if m["Content-Type"] != "application/json" {
		t.Errorf("expected 'application/json', got %q", m["Content-Type"])
	}
	if m["X-Custom"] != "val1" {
		t.Errorf("expected first value 'val1', got %q", m["X-Custom"])
	}
}
