package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ClaudeConfig represents the subset of ~/.claude.json that we need.
type ClaudeConfig struct {
	BaseURL string `json:"base_url,omitempty"`
	Env     struct {
		AnthropicBaseURL string `json:"ANTHROPIC_BASE_URL,omitempty"`
	} `json:"env,omitempty"`
}

// ReadClaudeConfig reads and parses ~/.claude.json. Returns nil config without error if the file does not exist.
func ReadClaudeConfig() (*ClaudeConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil
	}

	path := filepath.Join(home, ".claude.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read claude config: %w", err)
	}

	var cfg ClaudeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, nil // not fatal, just ignore malformed config
	}
	return &cfg, nil
}

// ClaudeBaseURLFromConfig returns the base URL from claude config, checking both top-level and env fields.
func ClaudeBaseURLFromConfig(cfg *ClaudeConfig) string {
	if cfg == nil {
		return ""
	}
	if cfg.Env.AnthropicBaseURL != "" {
		return cfg.Env.AnthropicBaseURL
	}
	return cfg.BaseURL
}

// HomeDir returns the user's home directory across platforms.
func HomeDir() string {
	if runtime.GOOS == "windows" {
		if home := os.Getenv("USERPROFILE"); home != "" {
			return home
		}
	}
	home, _ := os.UserHomeDir()
	return home
}
