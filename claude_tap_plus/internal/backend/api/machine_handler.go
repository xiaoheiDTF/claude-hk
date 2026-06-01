// Package api 提供后端 HTTP API 的处理函数和路由定义。
package api

import (
	"net/http"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// MachineHandler 处理 Machine 相关的 HTTP 请求。
type MachineHandler struct {
	svc *service.MachineService
}

// NewMachineHandler 创建新的 MachineHandler 实例。
func NewMachineHandler(svc *service.MachineService) *MachineHandler {
	return &MachineHandler{svc: svc}
}

// List 处理获取机器列表的请求。
// 接收 GET 请求，支持按操作系统和主机名过滤。
func (h *MachineHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	var filter store.MachineFilter
	if v := r.URL.Query().Get("os"); v != "" {
		filter.OS = &v
	}
	if v := r.URL.Query().Get("hostname"); v != "" {
		filter.Hostname = &v
	}

	logger.Debug("api.machine", "GET /api/machines")

	machines, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list machines")
		return
	}

	// 确保返回空数组而非 null
	if machines == nil {
		machines = []store.Machine{}
	}

	items := make([]MachineListItem, len(machines))
	for i, m := range machines {
		items[i] = MachineListItem{
			ID:          m.ID,
			MachineID:   m.MachineID,
			OS:          m.OS,
			Hostname:    m.Hostname,
			Username:    m.Username,
			FirstSeenAt: m.FirstSeenAt,
			LastSeenAt:  m.LastSeenAt,
		}
	}

	writeJSON(w, http.StatusOK, MachinesListResponse{Machines: items})
}
