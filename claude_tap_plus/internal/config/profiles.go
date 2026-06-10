package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// ProfileConfig 表示 profiles.json 中单个配置的定义。
type ProfileConfig struct {
	BaseURL   string `json:"base_url"`              // 上游 API 地址
	APIKey    string `json:"api_key"`               // API 密钥
	AuthToken string `json:"auth_token,omitempty"`  // OAuth Token
	Provider  string `json:"provider,omitempty"`    // 供应商标识（anthropic/openai/gemini）
	Model     string `json:"model,omitempty"`       // 强制替换的模型名
}

// ProfilesFile 表示 profiles.json 的完整结构。
type ProfilesFile struct {
	Default  string                   `json:"default"`  // 默认配置名
	Profiles map[string]ProfileConfig `json:"profiles"` // 配置集合
}

// ConfigDir 返回 claude-tap-plus 的配置根目录（~/.claude-tap-plus/）。
func ConfigDir() string {
	return filepath.Join(HomeDir(), ".claude-tap-plus")
}

// profilesPath 返回 profiles.json 的完整路径。
func profilesPath() string {
	return filepath.Join(ConfigDir(), "profiles.json")
}

// ReadProfiles 读取并解析 ~/.claude-tap-plus/profiles.json。
// 如果文件不存在，返回 nil 且不报错（非致命）。
func ReadProfiles() (*ProfilesFile, error) {
	path := profilesPath()
	logger.Debug("config", "reading profiles: %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("config", "profiles.json not found")
			return nil, nil
		}
		return nil, fmt.Errorf("read profiles: %w", err)
	}

	var pf ProfilesFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parse profiles: %w", err)
	}

	logger.Debug("config", "profiles loaded: default=%s count=%d", pf.Default, len(pf.Profiles))
	return &pf, nil
}

// ResolveProfileConfig 解析指定名称的配置。
// 如果 name 为空，使用 profiles.json 中的 default 字段。
// 如果 name 非空但未找到对应配置，返回错误。
func ResolveProfileConfig(name string) (*ProfileConfig, error) {
	pf, err := ReadProfiles()
	if err != nil {
		return nil, err
	}
	if pf == nil {
		return nil, fmt.Errorf("profiles.json not found")
	}

	// 使用默认配置名
	if name == "" {
		name = pf.Default
	}
	if name == "" {
		return nil, fmt.Errorf("no profile name specified and no default configured")
	}

	p, ok := pf.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found in profiles.json", name)
	}

	logger.Info("config", "resolved profile: %s (base_url=%s)", name, p.BaseURL)
	return &p, nil
}

// ResolveFallbackProfiles 按优先级查找 fallback profile 列表。
//
// 优先级（从高到低）：
//  1. profiles.json 中与 targetModel 匹配的 profile（同 model）
//  2. profiles.json 中其他 model 的 profile（不同 model）
//  3. 返回空（调用方走原始 fallback 逻辑）
//
// excludeProfile 为当前使用的 profile 名称，查找时会被排除。
// 同 model 的多个 profile 按字母序排列，方便调用方轮询。
func ResolveFallbackProfiles(targetModel, excludeProfile string) ([]ProfileConfig, error) {
	pf, err := ReadProfiles()
	if err != nil {
		return nil, err
	}
	if pf == nil {
		return nil, fmt.Errorf("profiles.json not found")
	}

	var sameModel []ProfileConfig
	var otherModel []ProfileConfig

	for name, p := range pf.Profiles {
		// 排除当前正在使用的 profile
		if name == excludeProfile {
			continue
		}
		if p.BaseURL == "" {
			continue // base_url 为空无法作为 fallback
		}
		if p.Model == targetModel && targetModel != "" {
			sameModel = append(sameModel, p)
		} else {
			otherModel = append(otherModel, p)
		}
	}

	// 优先返回同 model 的 profile
	if len(sameModel) > 0 {
		logger.Info("config", "fallback profiles: %d same-model candidates", len(sameModel))
		return sameModel, nil
	}

	if len(otherModel) > 0 {
		logger.Info("config", "fallback profiles: %d other-model candidates", len(otherModel))
		return otherModel, nil
	}

	logger.Debug("config", "no fallback profiles found")
	return nil, nil
}
