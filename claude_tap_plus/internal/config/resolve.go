package config

import (
	"os"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// ResolvedConfig 包含按优先级链解析后的最终配置。
type ResolvedConfig struct {
	BaseURL string // 最终的上游 API 地址
	APIKey  string // 最终的 API Key（可能为空）
}

// ResolveTargetConfig 按优先级解析最终的上游 API 配置。
//
// 优先级（从高到低）：
//
//	1. cliBaseURL / cliAPIKey（命令行直接指定）
//	2. profileName（读取 profiles.json）
//	3. 环境变量（ANTHROPIC_API_KEY, ANTHROPIC_BASE_URL）
//	4. ~/.claude.json（Claude Code 配置文件）
//	5. 默认值
func ResolveTargetConfig(cliBaseURL, cliAPIKey, profileName string, cfg *ClientConfig) (*ResolvedConfig, error) {
	result := &ResolvedConfig{}

	// Level 2: 从 profiles.json 读取配置
	if profileName != "" {
		p, err := ResolveProfileConfig(profileName)
		if err != nil {
			logger.Warn("config", "profile resolution failed: %v", err)
		} else {
			result.BaseURL = p.BaseURL
			result.APIKey = p.APIKey
		}
	}

	// Level 1: 命令行直接指定，覆盖 profile 值
	if cliBaseURL != "" {
		result.BaseURL = cliBaseURL
	}
	if cliAPIKey != "" {
		result.APIKey = cliAPIKey
	}

	// Level 3: 环境变量（仅在仍为空时）
	if result.BaseURL == "" {
		if env := os.Getenv(cfg.BaseURLEnv); env != "" {
			result.BaseURL = env
			logger.Debug("config", "base_url from env %s", cfg.BaseURLEnv)
		}
	}
	if result.APIKey == "" {
		if env := os.Getenv("ANTHROPIC_API_KEY"); env != "" {
			result.APIKey = env
			logger.Debug("config", "api_key from env ANTHROPIC_API_KEY")
		}
	}

	// Level 4: ~/.claude.json（仅在 base_url 仍为空时）
	if result.BaseURL == "" && cfg == &ClaudeClient {
		cc, err := ReadClaudeConfig()
		if err == nil && cc != nil {
			if url := ClaudeBaseURLFromConfig(cc); url != "" {
				result.BaseURL = url
				logger.Debug("config", "base_url from ~/.claude.json")
			}
		}
	}

	// Level 5: 默认值
	if result.BaseURL == "" {
		result.BaseURL = cfg.DefaultTarget
	}

	logger.Info("config", "resolved: base_url=%s api_key_set=%v", result.BaseURL, result.APIKey != "")
	return result, nil
}
