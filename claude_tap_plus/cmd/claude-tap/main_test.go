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

// TestLoadFallbackConfigs_FromProfiles 测试从 profiles.json 加载 fallback
func TestLoadFallbackConfigs_FromProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".claude-tap-plus")
	os.MkdirAll(configDir, 0o755)

	profiles := map[string]any{
		"default": "current",
		"profiles": map[string]any{
			"current": map[string]any{
				"base_url": "https://api.current.com",
				"model":    "glm-5.1",
				"api_key":  "sk-current",
			},
			"fb-same": map[string]any{
				"base_url": "https://api.fbsame.com",
				"model":    "glm-5.1",
				"api_key":  "sk-fbsame",
			},
			"fb-other": map[string]any{
				"base_url": "https://api.fbothe.com",
				"model":    "claude-sonnet-4-6",
				"api_key":  "sk-fbothe",
			},
		},
	}
	pData, _ := json.Marshal(profiles)
	os.WriteFile(filepath.Join(configDir, "profiles.json"), pData, 0o644)

	origHomeDir := config.HomeDir
	config.SetHomeDir(func() string { return tmpDir })
	defer config.SetHomeDir(origHomeDir)

	cfgs := loadFallbackConfigs("glm-5.1", "current")
	if len(cfgs) != 2 {
		t.Fatalf("expected 2 fallback configs, got %d", len(cfgs))
	}

	// 第一个应该是同 model 的 fb-same
	if cfgs[0].BaseURL != "https://api.fbsame.com" {
		t.Errorf("first fallback base_url = %q, want %q", cfgs[0].BaseURL, "https://api.fbsame.com")
	}
	if cfgs[0].Model != "glm-5.1" {
		t.Errorf("first fallback model = %q, want %q", cfgs[0].Model, "glm-5.1")
	}

	// 第二个应该是其他 model 的 fb-other
	if cfgs[1].BaseURL != "https://api.fbothe.com" {
		t.Errorf("second fallback base_url = %q, want %q", cfgs[1].BaseURL, "https://api.fbothe.com")
	}
}

// TestLoadFallbackConfigs_FromSettings 测试 profiles 无匹配时回退到 settings
func TestLoadFallbackConfigs_FromSettings(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建空的 profiles.json（只有当前 profile）
	configDir := filepath.Join(tmpDir, ".claude-tap-plus")
	os.MkdirAll(configDir, 0o755)
	profiles := map[string]any{
		"default": "only",
		"profiles": map[string]any{
			"only": map[string]any{
				"base_url": "https://api.only.com",
				"model":    "glm-5.1",
			},
		},
	}
	pData, _ := json.Marshal(profiles)
	os.WriteFile(filepath.Join(configDir, "profiles.json"), pData, 0o644)

	// 创建 settings.json
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

	cfgs := loadFallbackConfigs("glm-5.1", "only")
	if len(cfgs) != 1 {
		t.Fatalf("expected 1 fallback config from settings, got %d", len(cfgs))
	}
	if cfgs[0].BaseURL != "https://api.anthropic.com" {
		t.Errorf("fallback base_url = %q, want %q", cfgs[0].BaseURL, "https://api.anthropic.com")
	}
	if cfgs[0].Model != "claude-sonnet-4-6" {
		t.Errorf("fallback model = %q, want %q", cfgs[0].Model, "claude-sonnet-4-6")
	}
}

// TestLoadFallbackConfigs_NoSettings 测试无配置时返回空
func TestLoadFallbackConfigs_NoSettings(t *testing.T) {
	tmpDir := t.TempDir()
	// 不创建任何配置文件

	origHomeDir := config.HomeDir
	config.SetHomeDir(func() string { return tmpDir })
	defer config.SetHomeDir(origHomeDir)

	cfgs := loadFallbackConfigs("glm-5.1", "nonexistent")
	if len(cfgs) != 0 {
		t.Errorf("expected 0 fallback configs, got %d", len(cfgs))
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
			"fallback": map[string]any{
				"base_url": "PLACEHOLDER",
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
		"fallback": map[string]any{
			"base_url": "https://fallback.example.com",
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

	fbCfgs := loadFallbackConfigs(resolved.Model, "test-profile")
	if len(fbCfgs) == 0 {
		t.Fatal("loadFallbackConfigs returned empty")
	}
	rp.SetFallbackConfigs(fbCfgs)

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
	if fbCfgs[0].Model != "glm-5.1" {
		t.Errorf("fallback model = %q, want %q", fbCfgs[0].Model, "glm-5.1")
	}
}
