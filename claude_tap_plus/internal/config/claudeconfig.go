// Package config 提供代理所需的配置读取与解析功能。
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// ClaudeConfig 表示 ~/.claude.json 配置文件中我们关心的子集。
type ClaudeConfig struct {
	BaseURL string `json:"base_url,omitempty"` // 顶层 base_url 字段
	Model   string `json:"model,omitempty"`    // 默认使用的模型
	Env     struct {
		AnthropicBaseURL string `json:"ANTHROPIC_BASE_URL,omitempty"` // env 下的 ANTHROPIC_BASE_URL
	} `json:"env,omitempty"`
}

// ReadClaudeConfig 读取并解析 ~/.claude.json 配置文件。
// 如果文件不存在，返回 nil 且不报错。
func ReadClaudeConfig() (*ClaudeConfig, error) {
	home := HomeDir()
	if home == "" {
		return nil, nil
	}

	path := filepath.Join(home, ".claude.json")
	logger.Debug("config", "reading claude config: %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("config", "claude config not found")
			return nil, nil
		}
		return nil, fmt.Errorf("read claude config: %w", err)
	}

	var cfg ClaudeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, nil // 格式错误不致命，忽略即可
	}
	logger.Debug("config", "claude config loaded: base_url=%s", cfg.BaseURL)
	return &cfg, nil
}

// ClaudeBaseURLFromConfig 从 ClaudeConfig 中提取 base URL，
// 优先检查 env 下的 AnthropicBaseURL，其次使用顶层的 BaseURL。
func ClaudeBaseURLFromConfig(cfg *ClaudeConfig) string {
	if cfg == nil {
		return ""
	}
	if cfg.Env.AnthropicBaseURL != "" {
		return cfg.Env.AnthropicBaseURL
	}
	return cfg.BaseURL
}

// ClaudeSettings 表示 ~/.claude/settings.json 中我们关心的字段。
type ClaudeSettings struct {
	Model string `json:"model,omitempty"` // 默认模型
	Env   struct {
		AnthropicBaseURL  string `json:"ANTHROPIC_BASE_URL,omitempty"`
		AnthropicAuthToken string `json:"ANTHROPIC_AUTH_TOKEN,omitempty"`
		AnthropicAPIKey   string `json:"ANTHROPIC_API_KEY,omitempty"`
	} `json:"env,omitempty"`
}

// ReadClaudeSettings 读取并解析 ~/.claude/settings.json。
// 如果文件不存在，返回 nil 且不报错。
func ReadClaudeSettings() (*ClaudeSettings, error) {
	home := HomeDir()
	if home == "" {
		return nil, nil
	}

	path := filepath.Join(home, ".claude", "settings.json")
	logger.Debug("config", "reading claude settings: %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("config", "claude settings not found")
			return nil, nil
		}
		return nil, fmt.Errorf("read claude settings: %w", err)
	}

	var s ClaudeSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, nil // 格式错误不致命
	}
	logger.Debug("config", "claude settings loaded: model=%s has_auth=%v", s.Model, s.Env.AnthropicAuthToken != "" || s.Env.AnthropicAPIKey != "")
	return &s, nil
}

// homeDir 是用户主目录的可覆盖变量。测试中可替换为临时目录。
var homeDir = defaultHomeDir

// HomeDir 返回跨平台的用户主目录。
// Windows 下优先使用 USERPROFILE 环境变量。
func HomeDir() string {
	return homeDir()
}

// SetHomeDir 设置 homeDir 的实现（用于测试）。
func SetHomeDir(fn func() string) {
	homeDir = fn
}

func defaultHomeDir() string {
	if runtime.GOOS == "windows" {
		if home := os.Getenv("USERPROFILE"); home != "" {
			return home
		}
	}
	home, _ := os.UserHomeDir()
	return home
}
