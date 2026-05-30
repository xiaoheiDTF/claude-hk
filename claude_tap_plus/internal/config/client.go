// Package config 提供代理所需的配置读取与解析功能。
package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// ClientConfig 定义通过代理启动 AI CLI 时的单客户端配置。
type ClientConfig struct {
	Cmd               string   // 要定位的二进制文件名
	BaseURLEnv         string   // 用于覆盖 base URL 的环境变量名
	DefaultTarget      string   // 无覆盖时的默认上游 API 地址
	NestingEnvKeys     []string // 启动前需清空的环境变量（防止嵌套代理）
	InjectSettingsEnv  bool     // 是否通过 --settings 参数注入 base URL（Claude Code 专用）
}

// ClaudeClient 是 Claude Code 的默认配置。
var ClaudeClient = ClientConfig{
	Cmd:              "claude",
	BaseURLEnv:       "ANTHROPIC_BASE_URL",
	DefaultTarget:    "https://api.anthropic.com",
	NestingEnvKeys:   []string{"CLAUDECODE", "CLAUDE_CODE_SSE_PORT"},
	InjectSettingsEnv: true,
}

// DetectTarget 按优先级确定上游 API URL：
//  1. 显式环境变量（如 ANTHROPIC_BASE_URL）
//  2. ~/.claude.json 配置文件
//  3. 默认目标地址
func DetectTarget(cfg *ClientConfig) string {
	// 优先检查环境变量
	if envURL := os.Getenv(cfg.BaseURLEnv); envURL != "" {
		logger.Debug("config", "detected target from env %s=%s", cfg.BaseURLEnv, envURL)
		return envURL
	}
	logger.Debug("config", "env %s not set", cfg.BaseURLEnv)

	// 若为 ClaudeClient，则尝试读取 ~/.claude.json
	if cfg == &ClaudeClient {
		cc, err := ReadClaudeConfig()
		if err == nil && cc != nil {
			if url := ClaudeBaseURLFromConfig(cc); url != "" {
				logger.Debug("config", "detected target from claude config: %s", url)
				return url
			}
		}
	}

	logger.Info("config", "upstream target: %s", cfg.DefaultTarget)
	return cfg.DefaultTarget
}

// ResolveCmd 在 PATH 中查找客户端二进制文件的完整路径。
func ResolveCmd(cfg *ClientConfig) (string, error) {
	path, err := exec.LookPath(cfg.Cmd)
	if err != nil {
		return "", fmt.Errorf("'%s' command not found in PATH.\nPlease install it first", cfg.Cmd)
	}
	logger.Debug("config", "resolved command: %s -> %s", cfg.Cmd, path)
	return path, nil
}

// BuildChildEnv 创建子进程所需的环境变量：
//   - 继承当前环境
//   - 将 base URL 环境变量指向本地代理
//   - 清空嵌套代理相关环境变量
func BuildChildEnv(cfg *ClientConfig, proxyURL string) []string {
	env := os.Environ()

	result := make([]string, 0, len(env)+1)
	cleared := make(map[string]bool)
	for _, k := range cfg.NestingEnvKeys {
		cleared[k] = true
	}

	for _, e := range env {
		// 跳过嵌套键与 base URL 环境变量（我们自己设置）
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
	logger.Debug("config", "built child env: %s=%s, cleared %v", cfg.BaseURLEnv, proxyURL, cfg.NestingEnvKeys)
	return result
}

// clearedKeys 将 map 的键提取为字符串切片。
func clearedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// HasSettingsArg 判断参数列表中是否已包含 --settings。
func HasSettingsArg(args []string) bool {
	for _, arg := range args {
		if arg == "--settings" || strings.HasPrefix(arg, "--settings=") {
			return true
		}
	}
	return false
}

// BuildSettingsArgs 返回 ["--settings", JSON] 用于通过 Claude 的 settings 注入代理 URL。
func BuildSettingsArgs(cfg *ClientConfig, proxyURL string) []string {
	envJSON := fmt.Sprintf(`{"env":{"%s":"%s"}}`, cfg.BaseURLEnv, proxyURL)
	return []string{"--settings", envJSON}
}
