package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ClientConfig defines per-client settings for launching an AI CLI through the proxy.
type ClientConfig struct {
	Cmd               string   // binary name to locate
	BaseURLEnv         string   // env var for base URL override
	DefaultTarget      string   // upstream API URL when no override exists
	NestingEnvKeys     []string // env vars to clear before launch (prevent nested proxy)
	InjectSettingsEnv  bool     // inject base URL via --settings flag (Claude Code)
}

// ClaudeClient is the config for Claude Code.
var ClaudeClient = ClientConfig{
	Cmd:              "claude",
	BaseURLEnv:       "ANTHROPIC_BASE_URL",
	DefaultTarget:    "https://api.anthropic.com",
	NestingEnvKeys:   []string{"CLAUDECODE", "CLAUDE_CODE_SSE_PORT"},
	InjectSettingsEnv: true,
}

// DetectTarget determines the upstream API URL in priority order:
// 1. Explicit env var (ANTHROPIC_BASE_URL)
// 2. ~/.claude.json config
// 3. Default target
func DetectTarget(cfg *ClientConfig) string {
	// Check environment variable first.
	if envURL := os.Getenv(cfg.BaseURLEnv); envURL != "" {
		return envURL
	}

	// Check ~/.claude.json for Claude client.
	if cfg == &ClaudeClient {
		cc, err := ReadClaudeConfig()
		if err == nil && cc != nil {
			if url := ClaudeBaseURLFromConfig(cc); url != "" {
				return url
			}
		}
	}

	return cfg.DefaultTarget
}

// ResolveCmd finds the full path to the client binary.
func ResolveCmd(cfg *ClientConfig) (string, error) {
	path, err := exec.LookPath(cfg.Cmd)
	if err != nil {
		return "", fmt.Errorf("'%s' command not found in PATH.\nPlease install it first", cfg.Cmd)
	}
	return path, nil
}

// BuildChildEnv creates the environment for the child process:
// - Inherits the current environment
// - Sets the base URL env var to the local proxy
// - Clears nesting env keys
func BuildChildEnv(cfg *ClientConfig, proxyURL string) []string {
	env := os.Environ()

	result := make([]string, 0, len(env)+1)
	cleared := make(map[string]bool)
	for _, k := range cfg.NestingEnvKeys {
		cleared[k] = true
	}

	for _, e := range env {
		// Skip nesting keys and the base URL env (we'll set it ourselves).
		skip := false
		for _, prefix := range append(clearedKeys(cleared), cfg.BaseURLEnv) {
			if len(e) > len(prefix) && e[:len(prefix)+1] == prefix+"=" {
				skip = true
				break
			}
		}
		if !skip {
			result = append(result, e)
		}
	}

	result = append(result, cfg.BaseURLEnv+"="+proxyURL)
	return result
}

func clearedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// HasSettingsArg returns true if args already contains --settings.
func HasSettingsArg(args []string) bool {
	for _, arg := range args {
		if arg == "--settings" || strings.HasPrefix(arg, "--settings=") {
			return true
		}
	}
	return false
}

// BuildSettingsArgs returns ["--settings", JSON] to inject the proxy URL via Claude's settings.
func BuildSettingsArgs(cfg *ClientConfig, proxyURL string) []string {
	envJSON := fmt.Sprintf(`{"env":{"%s":"%s"}}`, cfg.BaseURLEnv, proxyURL)
	return []string{"--settings", envJSON}
}
