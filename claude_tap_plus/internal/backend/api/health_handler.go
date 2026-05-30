// Package api 提供后端 HTTP API 的处理函数和路由定义。
package api

import (
	"net/http"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// Health 处理健康检查请求，返回服务状态。
func Health(w http.ResponseWriter, r *http.Request) {
	logger.Debug("api.health", "health check")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
