package api

import (
	"net/http"
	"strconv"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// LogHandler 处理日志查询的 HTTP 请求。
type LogHandler struct {
	svc *service.LogService // 日志查询服务
}

// NewLogHandler 创建新的 LogHandler 实例。
func NewLogHandler(svc *service.LogService) *LogHandler {
	return &LogHandler{svc: svc}
}

// Query 处理 GET /api/logs 请求，查询后端服务日志。
// 支持查询参数：level（级别过滤）、date（日期选择）、limit（限制条数）。
func (h *LogHandler) Query(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	level := r.URL.Query().Get("level")
	date := r.URL.Query().Get("date")
	limitStr := r.URL.Query().Get("limit")

	limit := 100
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	logger.Debug("api.log", "GET /api/logs level=%s date=%s limit=%d", level, date, limit)

	entries, err := h.svc.QueryLogs(r.Context(), level, date, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to query logs")
		return
	}

	items := make([]LogItem, len(entries))
	for i, e := range entries {
		items[i] = LogItem{
			Timestamp: e.Timestamp,
			Level:     e.Level,
			Source:    e.Source,
			Message:   e.Message,
		}
	}

	// 确保返回空数组而非 null
	if items == nil {
		items = []LogItem{}
	}

	writeJSON(w, http.StatusOK, LogsResponse{Logs: items})
}
