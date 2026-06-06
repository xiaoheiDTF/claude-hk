package config

import (
	"os"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// ResolvedConfig 包含按优先级链解析后的最终配置。
type ResolvedConfig struct {
	BaseURL   string // 最终的上游 API 地址
	APIKey    string // 最终的 API Key（可能为空）
	AuthToken string // 最终的 OAuth Token（可能为空）
	Model     string // 最终的模型名（可能为空，空表示不做替换）
}

// ResolveTargetConfig 按优先级解析最终的上游 API 配置。
//
// 优先级（从高到低）：
//
//	1. cliBaseURL / cliAPIKey / cliAuthToken（命令行直接指定）
//	2. profileName（读取 profiles.json）
//	3. 环境变量（ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN, ANTHROPIC_BASE_URL）
//	4. ~/.claude.json（Claude Code 配置文件）
//	5. 默认值
func ResolveTargetConfig(cliBaseURL, cliAPIKey, cliAuthToken, profileName string, cfg *ClientConfig) (*ResolvedConfig, error) {
	result := &ResolvedConfig{}

	// Level 2: 从 profiles.json 读取配置
	if profileName != "" {
		p, err := ResolveProfileConfig(profileName)
		if err != nil {
			logger.Warn("config", "profile resolution failed: %v", err)
		} else {
			result.BaseURL = p.BaseURL
			result.APIKey = p.APIKey
			result.AuthToken = p.AuthToken
			result.Model = p.Model
		}
	}

	// Level 1: 命令行直接指定，覆盖 profile 值
	if cliBaseURL != "" {
		result.BaseURL = cliBaseURL
	}
	if cliAPIKey != "" {
		result.APIKey = cliAPIKey
	}
	if cliAuthToken != "" {
		result.AuthToken = cliAuthToken
	}

	// Level 3: 环境变量（仅在仍为空时）
	if result.BaseURL == "" {
		if env := os.Getenv(cfg.BaseURLEnv); env != "" {
			result.BaseURL = env
			logger.Debug("config", "base_url from env %s", cfg.BaseURLEnv)
		}
	}
	if result.APIKey == "" && result.AuthToken == "" {
		if env := os.Getenv("ANTHROPIC_API_KEY"); env != "" {
			result.APIKey = env
			logger.Debug("config", "api_key from env ANTHROPIC_API_KEY")
		}
		if env := os.Getenv("ANTHROPIC_AUTH_TOKEN"); env != "" {
			result.AuthToken = env
			logger.Debug("config", "auth_token from env ANTHROPIC_AUTH_TOKEN")
		}
	}

	// Level 4: ~/.claude.json（仅在 base_url 或 model 仍为空时）
	if cfg == &ClaudeClient {
		cc, err := ReadClaudeConfig()
		if err == nil && cc != nil {
			if result.BaseURL == "" {
				if url := ClaudeBaseURLFromConfig(cc); url != "" {
					result.BaseURL = url
					logger.Debug("config", "base_url from ~/.claude.json")
				}
			}
			// model 优先级：profile model > ~/.claude.json model > 空
			if result.Model == "" && cc.Model != "" {
				result.Model = cc.Model
				logger.Debug("config", "model from ~/.claude.json")
			}
		}
	}

	// Level 5: 默认值
	if result.BaseURL == "" {
		result.BaseURL = cfg.DefaultTarget
	}

	authType := "none"
	if result.APIKey != "" {
		authType = "api_key"
	} else if result.AuthToken != "" {
		authType = "auth_token"
	}
	logger.Info("config", "resolved: base_url=%s auth=%s model=%s", result.BaseURL, authType, result.Model)
	return result, nil
}
