package proxy_test

import (
	"net/http"
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/proxy"
)

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
