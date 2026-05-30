// Package backend 是后端服务的主包，提供 HTTP API 服务器及其配置。
package backend

import (
	"fmt"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// Config 保存后端服务的配置信息。
type Config struct {
	Host   string // HTTP 监听地址
	Port   int    // HTTP 监听端口
	DBPath string // SQLite 数据库文件路径
}

// Addr 返回 HTTP 监听地址的字符串表示，格式为 "host:port"。
func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// DefaultConfig 返回默认配置：本地 8080 端口，数据库文件为 backend.db。
func DefaultConfig() Config {
	cfg := Config{
		Host:   "127.0.0.1",
		Port:   8080,
		DBPath: "backend.db",
	}
	logger.Debug("backend.config", "default config: host=%s port=%d db=%s", cfg.Host, cfg.Port, cfg.DBPath)
	return cfg
}
