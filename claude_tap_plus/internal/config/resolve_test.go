package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestResolveTargetConfig_ModelPriority 测试 model 优先级链
// 来源：契约 2 — ResolvedConfig 内部数据传递
//
// 优先级：profile model > Claude settings model > 空
func TestResolveTargetConfig_ModelPriority(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		setup     func(t *testing.T) // 创建 profiles.json 和 ~/.claude.json
		profile   string
		cliModel  string // 命令行直接指定的 model（暂不实现，预留）
		wantModel string
	}{
		{
			name: "profile has model - use profile model",
			setup: func(t *testing.T) {
				writeProfilesJSON(t, tmpDir, `{
					"default": "my-glm",
					"profiles": {
						"my-glm": {
							"base_url": "https://api.example.com",
							"model": "glm-5.1"
						}
					}
				}`)
			},
			profile:   "my-glm",
			wantModel: "glm-5.1",
		},
		{
			name: "profile no model, claude settings has model - use claude settings",
			setup: func(t *testing.T) {
				writeProfilesJSON(t, tmpDir, `{
					"default": "basic",
					"profiles": {
						"basic": {
							"base_url": "https://api.example.com"
						}
					}
				}`)
				writeClaudeJSON(t, tmpDir, map[string]any{
					"model":    "claude-sonnet-4-6",
					"base_url": "https://api.anthropic.com",
				})
			},
			profile:   "basic",
			wantModel: "claude-sonnet-4-6",
		},
		{
			name: "no profile, claude settings has model - use claude settings",
			setup: func(t *testing.T) {
				writeClaudeJSON(t, tmpDir, map[string]any{
					"model":    "claude-sonnet-4-6",
					"base_url": "https://api.anthropic.com",
				})
			},
			profile:   "",
			wantModel: "claude-sonnet-4-6",
		},
		{
			name: "no profile, no claude settings model - model empty",
			setup: func(t *testing.T) {
				writeClaudeJSON(t, tmpDir, map[string]any{
					"base_url": "https://api.anthropic.com",
				})
			},
			profile:   "",
			wantModel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置临时 home 目录
			origHomeDir := homeDir
			homeDir = func() string { return tmpDir }
	// Note: using homeDir directly inside config package is fine
			defer func() { homeDir = origHomeDir }()

			tt.setup(t)

			resolved, err := ResolveTargetConfig(
				"",   // cliBaseURL
				"",   // cliAPIKey
				"",   // cliAuthToken
				tt.profile,
				&ClaudeClient,
			)
			if err != nil {
				t.Fatalf("ResolveTargetConfig error: %v", err)
			}
			if resolved.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", resolved.Model, tt.wantModel)
			}
		})
	}
}

// TestReadClaudeConfig_Model 测试从 ~/.claude.json 读取 model
// 来源：契约 2 — Claude settings 作为默认兜底
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
	// Note: using homeDir directly inside config package is fine
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

func writeProfilesJSON(t *testing.T, homeDir, content string) {
	t.Helper()
	configDir := filepath.Join(homeDir, ".claude-tap-plus")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "profiles.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

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
