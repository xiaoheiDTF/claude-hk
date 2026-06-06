package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestProfileConfig_Model 测试 ProfileConfig 解析 model 字段
// 来源：契约 1 — profiles.json ProfileConfig 结构
func TestProfileConfig_Model(t *testing.T) {
	jsonData := `{
		"default": "my-glm",
		"profiles": {
			"my-glm": {
				"base_url": "https://api.example.com",
				"api_key": "sk-xxx",
				"provider": "anthropic",
				"model": "glm-5.1"
			},
			"official": {
				"base_url": "https://api.anthropic.com",
				"provider": "anthropic"
			}
		}
	}`

	var pf ProfilesFile
	if err := json.Unmarshal([]byte(jsonData), &pf); err != nil {
		t.Fatalf("parse profiles: %v", err)
	}

	// 验证有 model 的 profile
	glm, ok := pf.Profiles["my-glm"]
	if !ok {
		t.Fatal("profile my-glm not found")
	}
	if glm.Model != "glm-5.1" {
		t.Errorf("my-glm model = %q, want %q", glm.Model, "glm-5.1")
	}

	// 验证没有 model 的 profile
	official, ok := pf.Profiles["official"]
	if !ok {
		t.Fatal("profile official not found")
	}
	if official.Model != "" {
		t.Errorf("official model = %q, want empty string", official.Model)
	}
}

// TestReadProfiles_WithModel 测试从文件读取含 model 的 profiles.json
// 来源：契约 1 — profiles.json ProfileConfig 结构
func TestReadProfiles_WithModel(t *testing.T) {
	// 创建临时 profiles.json
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".claude-tap-plus")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	profilesContent := `{
		"default": "my-glm",
		"profiles": {
			"my-glm": {
				"base_url": "https://api.example.com",
				"api_key": "sk-test",
				"model": "glm-5.1"
			}
		}
	}`
	profilesFile := filepath.Join(configDir, "profiles.json")
	if err := os.WriteFile(profilesFile, []byte(profilesContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// 覆盖 homeDir 使其指向临时目录
	origHomeDir := homeDir
	SetHomeDir(func() string { return tmpDir })
	defer func() { SetHomeDir(origHomeDir) }()

	pf, err := ReadProfiles()
	if err != nil {
		t.Fatalf("ReadProfiles: %v", err)
	}
	if pf == nil {
		t.Fatal("ReadProfiles returned nil")
	}

	glm, ok := pf.Profiles["my-glm"]
	if !ok {
		t.Fatal("profile my-glm not found")
	}
	if glm.Model != "glm-5.1" {
		t.Errorf("model = %q, want %q", glm.Model, "glm-5.1")
	}
}

// TestResolveProfileConfig_WithModel 测试解析指定 profile 并返回 model
// 来源：契约 1
func TestResolveProfileConfig_WithModel(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".claude-tap-plus")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	profilesContent := `{
		"default": "with-model",
		"profiles": {
			"with-model": {
				"base_url": "https://api.example.com",
				"model": "glm-5.1"
			},
			"without-model": {
				"base_url": "https://api.anthropic.com"
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(configDir, "profiles.json"), []byte(profilesContent), 0o644); err != nil {
		t.Fatal(err)
	}

	origHomeDir := homeDir
	SetHomeDir(func() string { return tmpDir })
	defer func() { SetHomeDir(origHomeDir) }()

	tests := []struct {
		name         string
		profileName  string
		wantModel    string
		wantBaseURL  string
		wantErr      bool
	}{
		{
			name:        "profile with model",
			profileName: "with-model",
			wantModel:   "glm-5.1",
			wantBaseURL: "https://api.example.com",
		},
		{
			name:        "profile without model",
			profileName: "without-model",
			wantModel:   "",
			wantBaseURL: "https://api.anthropic.com",
		},
		{
			name:        "use default profile name",
			profileName: "",
			wantModel:   "glm-5.1",
			wantBaseURL: "https://api.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ResolveProfileConfig(tt.profileName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveProfileConfig(%q) error = %v, wantErr %v", tt.profileName, err, tt.wantErr)
			}
			if p.Model != tt.wantModel {
				t.Errorf("model = %q, want %q", p.Model, tt.wantModel)
			}
			if p.BaseURL != tt.wantBaseURL {
				t.Errorf("base_url = %q, want %q", p.BaseURL, tt.wantBaseURL)
			}
		})
	}
}
