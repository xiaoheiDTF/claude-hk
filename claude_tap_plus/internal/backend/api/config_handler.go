package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// ConfigHandler 处理系统配置管理的 HTTP 请求。
type ConfigHandler struct {
	svc *service.ConfigService
}

// NewConfigHandler 创建新的 ConfigHandler 实例。
func NewConfigHandler(svc *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{svc: svc}
}

// ServeHTTP 处理 /api/config 的 GET 和 PUT 请求。
func (h *ConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.get(w, r)
	case http.MethodPut:
		h.update(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or PUT")
	}
}

// get 处理 GET /api/config，返回当前所有配置项。
func (h *ConfigHandler) get(w http.ResponseWriter, r *http.Request) {
	logger.Debug("api.config", "GET /api/config")

	config, err := h.svc.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get config")
		return
	}

	writeJSON(w, http.StatusOK, ConfigResponse{Config: config})
}

// update 处理 PUT /api/config，更新指定的配置项。
func (h *ConfigHandler) update(w http.ResponseWriter, r *http.Request) {
	// 解析为通用 map 以支持部分更新
	var raw map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	logger.Debug("api.config", "PUT /api/config: %v", raw)

	if len(raw) == 0 {
		writeError(w, http.StatusBadRequest, "no_updates", "no valid fields to update")
		return
	}

	if err := h.svc.Update(r.Context(), raw); err != nil {
		// 根据错误类型返回不同错误码
		errMsg := err.Error()
		code := "internal_error"
		status := http.StatusInternalServerError
		if strings.Contains(errMsg, "unknown config key") {
			code = "unknown_config_key"
			status = http.StatusBadRequest
		} else if strings.Contains(errMsg, "invalid") {
			code = "invalid_config"
			status = http.StatusBadRequest
		}
		writeError(w, status, code, errMsg)
		return
	}

	// 返回更新后的完整配置
	config, err := h.svc.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get updated config")
		return
	}

	writeJSON(w, http.StatusOK, ConfigResponse{Config: config})
}
