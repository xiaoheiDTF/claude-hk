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

// TestLoadFallbackConfig_AllFields 测试完整兜底配置加载
// 来源：契约 4 — 从 ~/.claude/settings.json 读取完整 fallback
func TestLoadFallbackConfig_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	os.MkdirAll(claudeDir, 0o755)

	settings := map[string]any{
		"model": "GLM-5.1",
		"env": map[string]string{
			"ANTHROPIC_BASE_URL":   "https://api.anthropic.com",
			"ANTHROPIC_AUTH_TOKEN": "tok-test-token",
			"ANTHROPIC_API_KEY":    "sk-test-key",
		},
	}
	data, _ := json.Marshal(settings)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644)

	origHomeDir := config.HomeDir
	// 通过覆盖 config 的 homeDir 变量
	config.SetHomeDir(func() string { return tmpDir })
	defer config.SetHomeDir(origHomeDir)

	fb := loadFallbackConfig()
	if fb == nil {
		t.Fatal("loadFallbackConfig returned nil")
	}
	if fb.BaseURL != "https://api.anthropic.com" {
		t.Errorf("BaseURL = %q, want %q", fb.BaseURL, "https://api.anthropic.com")
	}
	if fb.Model != "GLM-5.1" {
		t.Errorf("Model = %q, want %q", fb.Model, "GLM-5.1")
	}
	if fb.AuthToken != "tok-test-token" {
		t.Errorf("AuthToken = %q, want %q", fb.AuthToken, "tok-test-token")
	}
	if fb.APIKey != "sk-test-key" {
		t.Errorf("APIKey = %q, want %q", fb.APIKey, "sk-test-key")
	}
}

// TestLoadFallbackConfig_NoSettings 测试无 settings 文件
func TestLoadFallbackConfig_NoSettings(t *testing.T) {
	tmpDir := t.TempDir()
	// 不创建 .claude 目录

	origHomeDir := config.HomeDir
	config.SetHomeDir(func() string { return tmpDir })
	defer config.SetHomeDir(origHomeDir)

	fb := loadFallbackConfig()
	if fb != nil {
		t.Error("expected nil when no settings file")
	}
}

// TestLoadFallbackConfig_OnlyModel 测试只有 model 无 auth
func TestLoadFallbackConfig_OnlyModel(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	os.MkdirAll(claudeDir, 0o755)

	settings := map[string]any{
		"model": "claude-sonnet-4-6",
	}
	data, _ := json.Marshal(settings)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644)

	origHomeDir := config.HomeDir
	config.SetHomeDir(func() string { return tmpDir })
	defer config.SetHomeDir(origHomeDir)

	fb := loadFallbackConfig()
	if fb == nil {
		t.Fatal("loadFallbackConfig returned nil")
	}
	if fb.Model != "claude-sonnet-4-6" {
		t.Errorf("Model = %q, want %q", fb.Model, "claude-sonnet-4-6")
	}
	if fb.AuthToken != "" {
		t.Errorf("AuthToken = %q, want empty", fb.AuthToken)
	}
	if fb.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", fb.APIKey)
	}
	// 无 base_url 应使用默认值
	if fb.BaseURL != config.ClaudeClient.DefaultTarget {
		t.Errorf("BaseURL = %q, want default %q", fb.BaseURL, config.ClaudeClient.DefaultTarget)
	}
}

// TestIntegration_FullStartupFlow 测试完整启动链路
// profile → resolved config → proxy model → fallback → 请求转发
func TestIntegration_FullStartupFlow(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. 创建 profiles.json
	configDir := filepath.Join(tmpDir, ".claude-tap-plus")
	os.MkdirAll(configDir, 0o755)
	profiles := map[string]any{
		"default": "test-profile",
		"profiles": map[string]any{
			"test-profile": map[string]any{
				"base_url": "PLACEHOLDER", // 会被 mock 替换
				"model":    "glm-5.1",
			},
		},
	}
	pData, _ := json.Marshal(profiles)
	os.WriteFile(filepath.Join(configDir, "profiles.json"), pData, 0o644)

	// 2. 创建 ~/.claude/settings.json（fallback 用）
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

	// 3. 设置 home 目录
	origHomeDir := config.HomeDir
	config.SetHomeDir(func() string { return tmpDir })
	defer config.SetHomeDir(origHomeDir)

	// 4. 创建 mock 上游（profile target）
	var capturedBody []byte
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"test","stop_reason":"end_turn","usage":{"input_tokens":0,"output_tokens":0}}`))
	}))
	defer mockUpstream.Close()

	// 5. 更新 profile 的 base_url 为 mock 地址
	profiles["profiles"] = map[string]any{
		"test-profile": map[string]any{
			"base_url": mockUpstream.URL,
			"model":    "glm-5.1",
		},
	}
	pData, _ = json.Marshal(profiles)
	os.WriteFile(filepath.Join(configDir, "profiles.json"), pData, 0o644)

	// 6. 解析配置（模拟 runProxy 的配置解析）
	resolved, err := config.ResolveTargetConfig(
		"",   // cliBaseURL
		"",   // cliAPIKey
		"",   // cliAuthToken
		"test-profile",
		&config.ClaudeClient,
	)
	if err != nil {
		t.Fatalf("ResolveTargetConfig error: %v", err)
	}

	// 验证 resolved model
	if resolved.Model != "glm-5.1" {
		t.Fatalf("resolved.Model = %q, want %q", resolved.Model, "glm-5.1")
	}

	// 7. 创建代理并设置 model + fallback
	traceDir := t.TempDir()
	rp := proxy.NewReverseProxy(resolved.BaseURL, traceDir)
	rp.SetModel(resolved.Model)

	fb := loadFallbackConfig()
	if fb == nil {
		t.Fatal("loadFallbackConfig returned nil")
	}
	rp.SetFallbackConfig(fb)

	// 8. 启动代理
	_, err = rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	// 9. 通过代理发送请求
	reqBody := `{"model":"claude-sonnet-4-6","stream":false,"messages":[{"role":"user","content":"hello"}]}`
	req, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	resp.Body.Close()

	// 10. 验证上游收到改写后的 model
	var upstreamBody map[string]any
	if err := json.Unmarshal(capturedBody, &upstreamBody); err != nil {
		t.Fatalf("parse upstream body: %v", err)
	}
	gotModel, _ := upstreamBody["model"].(string)
	if gotModel != "glm-5.1" {
		t.Errorf("upstream model = %q, want %q (should be overridden by profile model)", gotModel, "glm-5.1")
	}

	// 11. 验证 fallback 配置正确
	if fb.Model != "claude-sonnet-4-6" {
		t.Errorf("fallback model = %q, want %q", fb.Model, "claude-sonnet-4-6")
	}
	if fb.AuthToken != "tok-fallback-token" {
		t.Errorf("fallback token = %q, want %q", fb.AuthToken, "tok-fallback-token")
	}
}
