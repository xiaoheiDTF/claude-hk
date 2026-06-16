package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/config"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/proxy"
)

// TestLoadFallbackConfigs_FromSettings 测试 bypass 模式从 ~/.claude/settings.json 加载兜底
func TestLoadFallbackConfigs_FromSettings(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	os.MkdirAll(claudeDir, 0o755)
	settings := map[string]any{
		"model": "claude-sonnet-4-6",
		"env": map[string]string{
			"ANTHROPIC_BASE_URL":   "https://api.anthropic.com",
			"ANTHROPIC_AUTH_TOKEN": "tok-fallback-token",
		},
	}
	sData, _ := json.Marshal(settings)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), sData, 0o644)

	origHomeDir := config.HomeDir
	config.SetHomeDir(func() string { return tmpDir })
	defer config.SetHomeDir(origHomeDir)

	cfgs := loadFallbackConfigs()
	if len(cfgs) != 1 {
		t.Fatalf("expected 1 fallback config from settings, got %d", len(cfgs))
	}
	if cfgs[0].BaseURL != "https://api.anthropic.com" {
		t.Errorf("fallback base_url = %q, want api.anthropic.com", cfgs[0].BaseURL)
	}
	if cfgs[0].Model != "claude-sonnet-4-6" {
		t.Errorf("fallback model = %q, want claude-sonnet-4-6", cfgs[0].Model)
	}
}

// TestLoadFallbackConfigs_NoSettings 测试无配置时返回空
func TestLoadFallbackConfigs_NoSettings(t *testing.T) {
	tmpDir := t.TempDir()
	origHomeDir := config.HomeDir
	config.SetHomeDir(func() string { return tmpDir })
	defer config.SetHomeDir(origHomeDir)

	if cfgs := loadFallbackConfigs(); len(cfgs) != 0 {
		t.Errorf("expected 0 fallback configs, got %d", len(cfgs))
	}
}

// TestIntegration_AliasRouting 别名路由核心链路：
//  1. 请求 model="opus[1m]" → 上游收到 model="glm-5.2[1m]"
//  2. proxy 用别名凭证覆盖请求头（provider=anthropic → x-api-key）
//  3. 主别名返回 401 → 切同真实 model 的候选别名重试成功
func TestIntegration_AliasRouting(t *testing.T) {
	var (
		primaryHits   int
		fallbackHits  int
		capturedKey   string
		capturedModel string
	)

	// 主别名上游：始终 401，触发 fallback
	primaryUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits++
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer primaryUpstream.Close()

	// 候选别名上游（同真实 model）：成功，记录收到的 key 与 model
	fallbackUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackHits++
		capturedKey = r.Header.Get("x-api-key")
		body, _ := io.ReadAll(r.Body)
		var m map[string]any
		json.Unmarshal(body, &m)
		capturedModel, _ = m["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg","type":"message","role":"assistant","content":[],"model":"x","stop_reason":"end_turn","usage":{"input_tokens":0,"output_tokens":0}}`))
	}))
	defer fallbackUpstream.Close()

	// 装配别名表：两个别名同真实 model glm-5.2[1m]，不同 key/base_url
	aliases := []*proxy.Alias{
		{Name: "opus[1m]", Model: "glm-5.2[1m]", BaseURL: primaryUpstream.URL, APIKey: "sk-primary", Provider: "anthropic"},
		{Name: "opus2[1m]", Model: "glm-5.2[1m]", BaseURL: fallbackUpstream.URL, APIKey: "sk-fallback", Provider: "anthropic"},
	}

	traceDir := t.TempDir()
	rp := proxy.NewReverseProxy(primaryUpstream.URL, traceDir)
	rp.SetAliases(aliases, "")

	if _, err := rp.Start("127.0.0.1", 0); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	// Claude Code 发来的 model = 别名 name
	reqBody := `{"model":"opus[1m]","stream":false,"messages":[{"role":"user","content":"hi"}]}`
	req, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	// Claude Code 自带的占位凭证（proxy 应覆盖）
	req.Header.Set("x-api-key", "alias-mode-placeholder")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (fallback should succeed)", resp.StatusCode)
	}
	if primaryHits != 1 {
		t.Errorf("primary hits = %d, want 1", primaryHits)
	}
	if fallbackHits != 1 {
		t.Errorf("fallback hits = %d, want 1", fallbackHits)
	}
	// 改写为真实 model
	if capturedModel != "glm-5.2[1m]" {
		t.Errorf("upstream model = %q, want glm-5.2[1m]", capturedModel)
	}
	// 用候选别名的凭证，而非占位值
	if capturedKey != "sk-fallback" {
		t.Errorf("upstream x-api-key = %q, want sk-fallback (alias credential override)", capturedKey)
	}
}

// TestIntegration_AliasRouting_DefaultAlias 测试未命中别名时走 default_alias 兜底
func TestIntegration_AliasRouting_DefaultAlias(t *testing.T) {
	var capturedModel string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var m map[string]any
		json.Unmarshal(body, &m)
		capturedModel, _ = m["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg","type":"message","role":"assistant","content":[],"model":"x","stop_reason":"end_turn","usage":{"input_tokens":0,"output_tokens":0}}`))
	}))
	defer upstream.Close()

	aliases := []*proxy.Alias{
		{Name: "sonnet", Model: "glm-5.1", BaseURL: upstream.URL, APIKey: "sk-aaa", Provider: "anthropic"},
	}
	traceDir := t.TempDir()
	rp := proxy.NewReverseProxy(upstream.URL, traceDir)
	rp.SetAliases(aliases, "sonnet") // default_alias = sonnet
	if _, err := rp.Start("127.0.0.1", 0); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	// 未知 model → 兜底 sonnet → 改写为 glm-5.1
	reqBody := `{"model":"unknown-xxx","stream":false,"messages":[]}`
	req, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (default_alias)", resp.StatusCode)
	}
	if capturedModel != "glm-5.1" {
		t.Errorf("upstream model = %q, want glm-5.1 (default_alias real model)", capturedModel)
	}
}

// TestIntegration_AliasRouting_NoMatchNoDefault 测试未命中且无 default_alias → 明确错误
func TestIntegration_AliasRouting_NoMatchNoDefault(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("upstream should not be hit")
	}))
	defer upstream.Close()

	aliases := []*proxy.Alias{
		{Name: "sonnet", Model: "glm-5.1", BaseURL: upstream.URL, APIKey: "sk-aaa"},
	}
	traceDir := t.TempDir()
	rp := proxy.NewReverseProxy(upstream.URL, traceDir)
	rp.SetAliases(aliases, "") // 无 default_alias
	if _, err := rp.Start("127.0.0.1", 0); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	reqBody := `{"model":"unknown-xxx","stream":false,"messages":[]}`
	req, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Errorf("status = 200, want error (no match, no default_alias)")
	}
}

// TestProxySessionCloseEndpoint 验证 /_internal/session-close 端点触发 OnSessionClose
// 并真正清掉 proxy.json 中的条目（对称于 trace-init 注册，由 29-session-end 钩子调用）。
func TestProxySessionCloseEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	origBase := BaseDir
	SetBaseDir(func() string { return filepath.Join(tmpDir, ".claude-tap-plus") })
	defer SetBaseDir(origBase)

	key := "testproj_testsess"
	if err := RegisterProxySession(key, ProxySession{StartedAt: "2026-06-16T00:00:00Z", URL: "http://127.0.0.1:9999"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if got := len(ReadProxySessions()); got != 1 {
		t.Fatalf("after register: %d sessions, want 1", got)
	}
	sessions := ReadProxySessions()
	if sessions[key].URL != "http://127.0.0.1:9999" {
		t.Errorf("recorded URL = %q, want http://127.0.0.1:9999", sessions[key].URL)
	}

	rp := proxy.NewReverseProxy("http://127.0.0.1:0", t.TempDir())
	rp.OnSessionClose = func() {
		if err := UnregisterProxySession(key); err != nil {
			t.Errorf("unregister: %v", err)
		}
	}
	if _, err := rp.Start("127.0.0.1", 0); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	resp, err := http.Post(rp.URL()+"/_internal/session-close", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("session-close: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := len(ReadProxySessions()); got != 0 {
		t.Errorf("after session-close: %d sessions, want 0 (entry not cleaned)", got)
	}

	// GET 应被拒绝（仅 POST）
	getResp, _ := http.Get(rp.URL() + "/_internal/session-close")
	if getResp != nil {
		getResp.Body.Close()
		if getResp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("GET status = %d, want 405", getResp.StatusCode)
		}
	}
}
