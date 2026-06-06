package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestReadClaudeSettings_AllFields 测试完整读取 ~/.claude/settings.json
// 来源：契约 4 — 兜底配置来源
func TestReadClaudeSettings_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	settings := map[string]any{
		"model": "GLM-5.1",
		"env": map[string]string{
			"ANTHROPIC_BASE_URL":   "https://api.anthropic.com",
			"ANTHROPIC_AUTH_TOKEN": "tok-test-token-1234567890",
			"ANTHROPIC_API_KEY":    "sk-test-api-key",
			"API_TIMEOUT_MS":       "3000000",
		},
	}
	data, _ := json.Marshal(settings)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	origHomeDir := homeDir
	homeDir = func() string { return tmpDir }
	defer func() { homeDir = origHomeDir }()

	s, err := ReadClaudeSettings()
	if err != nil {
		t.Fatalf("ReadClaudeSettings error: %v", err)
	}
	if s == nil {
		t.Fatal("ReadClaudeSettings returned nil")
	}

	// 验证 model
	if s.Model != "GLM-5.1" {
		t.Errorf("Model = %q, want %q", s.Model, "GLM-5.1")
	}
	// 验证 base_url
	if s.Env.AnthropicBaseURL != "https://api.anthropic.com" {
		t.Errorf("AnthropicBaseURL = %q, want %q", s.Env.AnthropicBaseURL, "https://api.anthropic.com")
	}
	// 验证 auth token
	if s.Env.AnthropicAuthToken != "tok-test-token-1234567890" {
		t.Errorf("AnthropicAuthToken = %q, want %q", s.Env.AnthropicAuthToken, "tok-test-token-1234567890")
	}
	// 验证 api key
	if s.Env.AnthropicAPIKey != "sk-test-api-key" {
		t.Errorf("AnthropicAPIKey = %q, want %q", s.Env.AnthropicAPIKey, "sk-test-api-key")
	}
}

// TestReadClaudeSettings_PartialFields 测试部分字段
// 来源：契约 4 — 有 token 用 token，有 api_key 用 api_key
func TestReadClaudeSettings_PartialFields(t *testing.T) {
	tests := []struct {
		name           string
		settings       map[string]any
		wantModel      string
		wantBaseURL    string
		wantAuthToken  string
		wantAPIKey     string
	}{
		{
			name: "only token, no api_key",
			settings: map[string]any{
				"model": "claude-sonnet-4-6",
				"env": map[string]string{
					"ANTHROPIC_BASE_URL":   "https://api.anthropic.com",
					"ANTHROPIC_AUTH_TOKEN": "tok-only-token",
				},
			},
			wantModel:     "claude-sonnet-4-6",
			wantBaseURL:   "https://api.anthropic.com",
			wantAuthToken: "tok-only-token",
			wantAPIKey:    "",
		},
		{
			name: "only api_key, no token",
			settings: map[string]any{
				"env": map[string]string{
					"ANTHROPIC_BASE_URL": "https://custom-api.example.com",
					"ANTHROPIC_API_KEY":  "sk-only-key",
				},
			},
			wantModel:     "",
			wantBaseURL:   "https://custom-api.example.com",
			wantAuthToken: "",
			wantAPIKey:    "sk-only-key",
		},
		{
			name: "both token and api_key",
			settings: map[string]any{
				"env": map[string]string{
					"ANTHROPIC_AUTH_TOKEN": "tok-both",
					"ANTHROPIC_API_KEY":    "sk-both",
				},
			},
			wantAuthToken: "tok-both",
			wantAPIKey:    "sk-both",
		},
		{
			name: "only model, no auth",
			settings: map[string]any{
				"model": "glm-5.1",
			},
			wantModel:     "glm-5.1",
			wantBaseURL:   "",
			wantAuthToken: "",
			wantAPIKey:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			claudeDir := filepath.Join(tmpDir, ".claude")
			os.MkdirAll(claudeDir, 0o755)

			data, _ := json.Marshal(tt.settings)
			os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644)

			origHomeDir := homeDir
			homeDir = func() string { return tmpDir }
			defer func() { homeDir = origHomeDir }()

			s, err := ReadClaudeSettings()
			if err != nil {
				t.Fatalf("ReadClaudeSettings error: %v", err)
			}
			if s == nil {
				t.Fatal("ReadClaudeSettings returned nil")
			}
			if s.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", s.Model, tt.wantModel)
			}
			if s.Env.AnthropicBaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL = %q, want %q", s.Env.AnthropicBaseURL, tt.wantBaseURL)
			}
			if s.Env.AnthropicAuthToken != tt.wantAuthToken {
				t.Errorf("AuthToken = %q, want %q", s.Env.AnthropicAuthToken, tt.wantAuthToken)
			}
			if s.Env.AnthropicAPIKey != tt.wantAPIKey {
				t.Errorf("APIKey = %q, want %q", s.Env.AnthropicAPIKey, tt.wantAPIKey)
			}
		})
	}
}

// TestReadClaudeSettings_NotFound 测试 settings.json 不存在
func TestReadClaudeSettings_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	// 不创建 .claude 目录

	origHomeDir := homeDir
	homeDir = func() string { return tmpDir }
	defer func() { homeDir = origHomeDir }()

	s, err := ReadClaudeSettings()
	if err != nil {
		t.Fatalf("ReadClaudeSettings error: %v", err)
	}
	if s != nil {
		t.Error("expected nil when settings.json not found")
	}
}
