package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestResolveTargetConfig_SourcePriority 测试 bypass 模式凭证/base_url 来源优先级
// CLI > env > ~/.claude.json > default。profile 不再贡献凭证（凭证集中在 aliases）。
func TestResolveTargetConfig_SourcePriority(t *testing.T) {
	tmpDir := t.TempDir()

	// ~/.claude.json 提供 base_url + model 兜底
	writeClaudeJSON(t, tmpDir, map[string]any{
		"model":    "claude-sonnet-4-6",
		"base_url": "https://api.anthropic.com",
	})

	origHomeDir := homeDir
	homeDir = func() string { return tmpDir }
	defer func() { homeDir = origHomeDir }()

	// 清理真实环境变量，避免干扰优先级判定
	t.Setenv("ANTHROPIC_BASE_URL", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")

	// 无 CLI 覆盖 → 走 ~/.claude.json
	resolved, err := ResolveTargetConfig("", "", "", &ClaudeClient)
	if err != nil {
		t.Fatalf("ResolveTargetConfig: %v", err)
	}
	if resolved.BaseURL != "https://api.anthropic.com" {
		t.Errorf("base_url = %q, want claude.json value", resolved.BaseURL)
	}
	if resolved.Model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want claude-sonnet-4-6", resolved.Model)
	}

	// CLI 覆盖优先级最高
	resolved, _ = ResolveTargetConfig("https://cli.example.com", "sk-cli", "", &ClaudeClient)
	if resolved.BaseURL != "https://cli.example.com" {
		t.Errorf("base_url = %q, want cli override", resolved.BaseURL)
	}
	if resolved.APIKey != "sk-cli" {
		t.Errorf("api_key = %q, want sk-cli", resolved.APIKey)
	}
}

// TestResolveTargetConfig_DefaultFallback 无任何配置时回退默认值
func TestResolveTargetConfig_DefaultFallback(t *testing.T) {
	tmpDir := t.TempDir()
	origHomeDir := homeDir
	homeDir = func() string { return tmpDir }
	defer func() { homeDir = origHomeDir }()

	t.Setenv("ANTHROPIC_BASE_URL", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")

	resolved, err := ResolveTargetConfig("", "", "", &ClaudeClient)
	if err != nil {
		t.Fatalf("ResolveTargetConfig: %v", err)
	}
	if resolved.BaseURL != ClaudeClient.DefaultTarget {
		t.Errorf("base_url = %q, want default %q", resolved.BaseURL, ClaudeClient.DefaultTarget)
	}
}

// TestReadClaudeConfig_Model 测试从 ~/.claude.json 读取 model
func TestReadClaudeConfig_Model(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		content   map[string]any
		wantModel string
	}{
		{
			name: "claude.json with model",
			content: map[string]any{
				"model":    "claude-sonnet-4-6",
				"base_url": "https://api.anthropic.com",
			},
			wantModel: "claude-sonnet-4-6",
		},
		{
			name: "claude.json without model",
			content: map[string]any{
				"base_url": "https://api.anthropic.com",
			},
			wantModel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeClaudeJSON(t, tmpDir, tt.content)

			origHomeDir := homeDir
			homeDir = func() string { return tmpDir }
			defer func() { homeDir = origHomeDir }()

			cfg, err := ReadClaudeConfig()
			if err != nil {
				t.Fatalf("ReadClaudeConfig error: %v", err)
			}
			if cfg.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", cfg.Model, tt.wantModel)
			}
		})
	}
}

// --- 测试辅助函数 ---

func writeClaudeJSON(t *testing.T, homeDir string, content map[string]any) {
	t.Helper()
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, ".claude.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}
