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

// ResolveTargetConfig 按优先级解析最终的上游 API 配置（仅 bypass 模式使用）。
//
// 别名路由模式下凭证随别名变化，不由此函数解析；此函数仅供 CLI 全局旁路（--tap-base-url 等）
// 或无 profiles.json 时的自动探测使用。profile 不再承载凭证（凭证集中在 aliases 表），
// profile.env 由调用方另行注入。
//
// 优先级（从高到低）：
//
//	1. cliBaseURL / cliAPIKey / cliAuthToken（命令行直接指定）
//	2. 环境变量（ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN, ANTHROPIC_BASE_URL）
//	3. ~/.claude.json（Claude Code 配置文件）
//	4. 默认值
func ResolveTargetConfig(cliBaseURL, cliAPIKey, cliAuthToken string, cfg *ClientConfig) (*ResolvedConfig, error) {
	result := &ResolvedConfig{}

	// Level 1: 命令行直接指定
	if cliBaseURL != "" {
		result.BaseURL = cliBaseURL
	}
	if cliAPIKey != "" {
		result.APIKey = cliAPIKey
	}
	if cliAuthToken != "" {
		result.AuthToken = cliAuthToken
	}

	// Level 2: 环境变量（仅在仍为空时）
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
