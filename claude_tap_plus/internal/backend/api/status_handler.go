// Package api 提供后端 HTTP API 的处理函数和路由定义。
package api

import (
	"net/http"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// StatusHandler 处理系统状态查询。
type StatusHandler struct {
	svc *service.StatusService
}

// NewStatusHandler 创建 StatusHandler 实例。
func NewStatusHandler(svc *service.StatusService) *StatusHandler {
	return &StatusHandler{svc: svc}
}

// Get 处理获取系统状态的请求。
// GET /api/status
func (h *StatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	logger.Debug("api.status", "GET /api/status")

	status, err := h.svc.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get status")
		return
	}

	writeJSON(w, http.StatusOK, StatusResponse{
		Status:        status.Status,
		Version:       status.Version,
		UptimeSeconds: status.UptimeSeconds,
		Stats: SystemStats{
			ActiveSessions: status.Stats.ActiveSessions,
			ActiveProxies:  status.Stats.ActiveProxies,
			PendingIssues:  status.Stats.PendingIssues,
			TotalMachines:  status.Stats.TotalMachines,
			TotalProjects:  status.Stats.TotalProjects,
		},
		Timestamp: status.Timestamp,
	})
}
