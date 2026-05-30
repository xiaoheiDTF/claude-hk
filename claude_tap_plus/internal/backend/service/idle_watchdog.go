// Package service 提供后端业务逻辑层。
package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// proxySession 用于反序列化 proxy.json 中的会话条目。
type proxySession struct {
	StartedAt string `json:"started_at"`
}

// IdleWatchdog 监控 proxy.json，当所有会话注销且持续空闲超过指定时间后，
// 调用 shutdown 回调触发后端优雅关闭。
type IdleWatchdog struct {
	interval    time.Duration // 检查间隔
	idleTimeout time.Duration // 空闲超时（超过此时间无会话则关闭）
	shutdown    func()        // 触发服务器关闭的回调
}

// NewIdleWatchdog 创建空闲监控器。
//   - interval: 检查间隔（建议 30s）
//   - idleTimeout: 空闲超时（建议 10min）
//   - shutdown: 触发关闭的回调函数
func NewIdleWatchdog(interval, idleTimeout time.Duration, shutdown func()) *IdleWatchdog {
	return &IdleWatchdog{
		interval:    interval,
		idleTimeout: idleTimeout,
		shutdown:    shutdown,
	}
}

// Run 启动监控循环，直到 context 被取消或触发关闭。
func (w *IdleWatchdog) Run(ctx context.Context) {
	logger.Info("watchdog", "started: interval=%v idle_timeout=%v", w.interval, w.idleTimeout)

	idleStart := time.Time{} // 记录首次检测到空闲的时间

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("watchdog", "stopped")
			return
		case <-ticker.C:
			sessionCount := countProxySessions()
			if sessionCount > 0 {
				// 有活跃会话，重置空闲计时
				if !idleStart.IsZero() {
					logger.Info("watchdog", "sessions active (%d), resetting idle timer", sessionCount)
				}
				idleStart = time.Time{}
			} else {
				// 无活跃会话
				if idleStart.IsZero() {
					// 首次检测到空闲
					idleStart = time.Now()
					logger.Info("watchdog", "no sessions, idle timer started")
				} else if time.Since(idleStart) >= w.idleTimeout {
					// 空闲超时，触发关闭
					logger.Info("watchdog", "idle timeout (%v) reached, shutting down", w.idleTimeout)
					w.shutdown()
					return
				} else {
					logger.Debug("watchdog", "idle for %v, shutdown in %v",
						time.Since(idleStart).Round(time.Second),
						w.idleTimeout-time.Since(idleStart))
				}
			}
		}
	}
}

// proxyJSONPath 返回 proxy.json 的路径（~/.claude-tap-plus/proxy.json）。
func proxyJSONPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".claude-tap-plus/proxy.json"
	}
	return filepath.Join(home, ".claude-tap-plus", "proxy.json")
}

// countProxySessions 读取 proxy.json 并返回活跃会话数量。
func countProxySessions() int {
	data, err := os.ReadFile(proxyJSONPath())
	if err != nil {
		return 0 // 文件不存在或读不出 → 视为无会话
	}
	var sessions map[string]proxySession
	if json.Unmarshal(data, &sessions) != nil {
		return 0
	}
	return len(sessions)
}
