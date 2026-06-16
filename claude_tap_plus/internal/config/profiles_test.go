package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeProfilesFile 在临时 home 下写入 profiles.json（新格式）。
func writeProfilesFile(t *testing.T, home, content string) {
	t.Helper()
	configDir := filepath.Join(home, ".claude-tap-plus")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "profiles.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestReadProfiles_NewFormat 测试新格式解析（aliases + profiles.env）
func TestReadProfiles_NewFormat(t *testing.T) {
	tmpDir := t.TempDir()
	writeProfilesFile(t, tmpDir, `{
		"default_profile": "work",
		"default_alias": "sonnet",
		"aliases": [
			{"name": "opus[1m]", "model": "glm-5.2[1m]", "base_url": "https://glm.example.com", "api_key": "sk-aaa"},
			{"name": "kimi", "model": "kimi-k2", "base_url": "https://api.kimi.com", "auth_token": "tok-kkk"}
		],
		"profiles": {
			"work": {"env": {"ANTHROPIC_MODEL": "opus[1m]"}}
		}
	}`)

	orig := homeDir
	homeDir = func() string { return tmpDir }
	defer func() { homeDir = orig }()

	pf, err := ReadProfiles()
	if err != nil {
		t.Fatalf("ReadProfiles: %v", err)
	}
	if pf == nil {
		t.Fatal("ReadProfiles returned nil")
	}
	if len(pf.Aliases) != 2 {
		t.Fatalf("aliases count = %d, want 2", len(pf.Aliases))
	}
	if pf.DefaultProfile != "work" || pf.DefaultAlias != "sonnet" {
		t.Errorf("defaults = %q/%q, want work/sonnet", pf.DefaultProfile, pf.DefaultAlias)
	}
	// provider 缺省补 anthropic
	if pf.Aliases[0].Provider != "anthropic" {
		t.Errorf("provider = %q, want anthropic", pf.Aliases[0].Provider)
	}
	// auth_token 优先：同时配置时 api_key 被清空
	if pf.Aliases[1].AuthToken != "tok-kkk" || pf.Aliases[1].APIKey != "" {
		t.Errorf("auth_token/api_key conflict not resolved: token=%q key=%q", pf.Aliases[1].AuthToken, pf.Aliases[1].APIKey)
	}
}

// TestReadProfiles_OldFormatDetected 测试旧扁平格式被检测并报错（F4.2）
func TestReadProfiles_OldFormatDetected(t *testing.T) {
	tmpDir := t.TempDir()
	writeProfilesFile(t, tmpDir, `{
		"default": "my-glm",
		"profiles": {
			"my-glm": {"base_url": "https://api.example.com", "api_key": "sk-xxx", "model": "glm-5.1"}
		}
	}`)

	orig := homeDir
	homeDir = func() string { return tmpDir }
	defer func() { homeDir = orig }()

	pf, err := ReadProfiles()
	if err == nil {
		t.Fatalf("expected error for old format, got pf=%v", pf)
	}
}

// TestReadProfiles_NotFound 测试文件缺失返回 (nil, nil)
func TestReadProfiles_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	orig := homeDir
	homeDir = func() string { return tmpDir }
	defer func() { homeDir = orig }()

	pf, err := ReadProfiles()
	if err != nil || pf != nil {
		t.Fatalf("expected (nil, nil), got (%v, %v)", pf, err)
	}
}

// TestResolveAlias 测试精确命中、default_alias 兜底、未命中
func TestResolveAlias(t *testing.T) {
	pf := &ProfilesFile{
		DefaultAlias: "sonnet",
		Aliases: []Alias{
			{Name: "opus[1m]", Model: "glm-5.2[1m]"},
			{Name: "sonnet", Model: "glm-5.1"},
		},
	}

	// 精确命中
	a, ok := pf.ResolveAlias("opus[1m]")
	if !ok || a.Model != "glm-5.2[1m]" {
		t.Errorf("ResolveAlias(opus[1m]) = %v,%v, want glm-5.2[1m],true", a, ok)
	}
	// default_alias 兜底
	a, ok = pf.ResolveAlias("unknown-xxx")
	if !ok || a.Name != "sonnet" {
		t.Errorf("ResolveAlias(unknown) = %v,%v, want default sonnet", a, ok)
	}
	// 无 default_alias 时未命中
	pf.DefaultAlias = ""
	_, ok = pf.ResolveAlias("unknown-xxx")
	if ok {
		t.Error("expected miss when no default_alias")
	}
}

// TestResolveAlias_DupNameLatterWins 测试同名别名后者覆盖（决策 1）
func TestResolveAlias_DupNameLatterWins(t *testing.T) {
	pf := &ProfilesFile{
		Aliases: []Alias{
			{Name: "dup", Model: "first"},
			{Name: "dup", Model: "second"},
		},
	}
	a, ok := pf.ResolveAlias("dup")
	if !ok || a.Model != "second" {
		t.Errorf("dup alias model = %q, want second (latter wins)", a.Model)
	}
}

// TestResolveFallbackAliases 测试同真实 model 候选链：排除自身、按数组顺序
func TestResolveFallbackAliases(t *testing.T) {
	pf := &ProfilesFile{
		Aliases: []Alias{
			{Name: "opus[1m]", Model: "glm-5.2[1m]", APIKey: "sk-aaa"},
			{Name: "opus2[1m]", Model: "glm-5.2[1m]", APIKey: "sk-bbb"},
			{Name: "sonnet", Model: "glm-5.1"},
			{Name: "opus3[1m]", Model: "glm-5.2[1m]", APIKey: "sk-ccc"},
		},
	}
	cands := pf.ResolveFallbackAliases("glm-5.2[1m]", "opus[1m]")
	if len(cands) != 2 {
		t.Fatalf("candidates = %d, want 2", len(cands))
	}
	if cands[0].Name != "opus2[1m]" || cands[1].Name != "opus3[1m]" {
		t.Errorf("order = %s,%s, want opus2[1m],opus3[1m]", cands[0].Name, cands[1].Name)
	}
}

// TestResolveProfileEnv_ForbiddenStripped 测试禁止项被剔除（F3.3）
func TestResolveProfileEnv_ForbiddenStripped(t *testing.T) {
	pf := &ProfilesFile{
		Profiles: map[string]Profile{
			"work": {Env: map[string]string{
				"ANTHROPIC_MODEL":         "opus[1m]",
				"ANTHROPIC_BASE_URL":      "https://should-be-stripped.com",
				"CLAUDE_CODE_USE_BEDROCK": "1",
			}},
		},
	}
	env, err := pf.ResolveProfileEnv("work")
	if err != nil {
		t.Fatalf("ResolveProfileEnv: %v", err)
	}
	if env["ANTHROPIC_BASE_URL"] != "" {
		t.Error("ANTHROPIC_BASE_URL should be stripped")
	}
	if _, ok := env["CLAUDE_CODE_USE_BEDROCK"]; ok {
		t.Error("CLAUDE_CODE_USE_BEDROCK should be stripped")
	}
	if env["ANTHROPIC_MODEL"] != "opus[1m]" {
		t.Errorf("ANTHROPIC_MODEL = %q, want opus[1m]", env["ANTHROPIC_MODEL"])
	}
}

// TestResolveProfileEnv_DefaultProfile 测试 name 缺省时使用 default_profile
func TestResolveProfileEnv_DefaultProfile(t *testing.T) {
	pf := &ProfilesFile{
		DefaultProfile: "work",
		Profiles: map[string]Profile{
			"work": {Env: map[string]string{"ANTHROPIC_MODEL": "opus[1m]"}},
		},
	}
	env, err := pf.ResolveProfileEnv("")
	if err != nil {
		t.Fatalf("ResolveProfileEnv: %v", err)
	}
	if env["ANTHROPIC_MODEL"] != "opus[1m]" {
		t.Errorf("ANTHROPIC_MODEL = %q, want opus[1m]", env["ANTHROPIC_MODEL"])
	}
}
