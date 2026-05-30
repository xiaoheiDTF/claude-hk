// Package api 提供后端 HTTP API 的处理函数和路由定义。
package api

import (
	"encoding/json"
	"net/http"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// SessionHandler 处理 Session 相关的 HTTP 请求。
type SessionHandler struct {
	svc *service.SessionService // Session 业务逻辑服务
}

// NewSessionHandler 创建新的 SessionHandler 实例。
func NewSessionHandler(svc *service.SessionService) *SessionHandler {
	return &SessionHandler{svc: svc}
}

// Register 处理注册新会话的请求。
// 接收 POST 请求，将新的会话信息保存到数据库。
func (h *SessionHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}

	var req RegisterSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	// 校验必填字段
	if req.SessionID == "" || req.MachineID == "" || req.ProjectSlug == "" || req.TranscriptPath == "" {
		writeError(w, http.StatusBadRequest, "invalid_request",
			"session_id, machine_id, project_slug and transcript_path are required")
		return
	}

	logger.Debug("api.session", "POST /api/session/register id=%s", req.SessionID)

	// 构造 Session 存储对象
	sess := store.Session{
		SessionID:      req.SessionID,
		MachineID:      req.MachineID,
		OS:             req.OS,
		ProjectSlug:    req.ProjectSlug,
		ProjectCwd:     req.ProjectCwd,
		TranscriptPath: req.TranscriptPath,
		LocalTracePath: req.LocalTracePath,
		Model:          req.Model,
		Source:         req.Source,
	}

	// 调用服务层注册会话
	if err := h.svc.Register(r.Context(), sess); err != nil {
		if err == store.ErrSessionExists {
			logger.Warn("api.session", "session already exists: %s", req.SessionID)
			writeError(w, http.StatusConflict, "session_exists", "session already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to register session")
		return
	}

	logger.Info("api.session", "session registered: %s slug=%s", req.SessionID, req.ProjectSlug)
	writeJSON(w, http.StatusOK, map[string]string{"status": "registered"})
}

// Close 处理关闭会话的请求。
// 接收 POST 请求，将会话状态标记为已关闭。
func (h *SessionHandler) Close(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}

	var req CloseSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session_id is required")
		return
	}

	logger.Debug("api.session", "POST /api/session/close id=%s", req.SessionID)

	// 调用服务层关闭会话
	if err := h.svc.Close(r.Context(), req.SessionID, req.Reason); err != nil {
		if err == store.ErrSessionNotFound {
			writeError(w, http.StatusNotFound, "not_found", "session not found or already closed")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to close session")
		return
	}

	logger.Info("api.session", "session closed: %s reason=%s", req.SessionID, req.Reason)
	writeJSON(w, http.StatusOK, map[string]string{"status": "closed"})
}

// List 处理获取会话列表的请求。
// 接收 GET 请求，支持按 machine_id、project_slug、status 过滤。
func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	logger.Debug("api.session", "GET /api/sessions")

	// 解析查询参数构建过滤条件
	var filter store.SessionFilter
	if v := r.URL.Query().Get("machine_id"); v != "" {
		filter.MachineID = &v
	}
	if v := r.URL.Query().Get("project_slug"); v != "" {
		filter.ProjectSlug = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}

	// 调用服务层查询会话列表
	sessions, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list sessions")
		return
	}

	// 确保返回空数组而非 null
	if sessions == nil {
		sessions = []store.Session{}
	}

	// 转换为响应格式
	items := make([]SessionListItem, len(sessions))
	for i, s := range sessions {
		items[i] = toSessionListItem(s)
	}

	writeJSON(w, http.StatusOK, SessionListResponse{Sessions: items})
}

// Get 处理获取单个会话详情的请求。
// 接收 GET 请求，从 URL 路径中解析 session_id。
func (h *SessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	// 从 URL 路径中提取 session_id
	sessionID := r.URL.Path[len("/api/session/"):]
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session_id is required")
		return
	}

	logger.Debug("api.session", "GET /api/session/%s", sessionID)

	// 调用服务层查询会话详情
	sess, err := h.svc.Get(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get session")
		return
	}
	if sess == nil {
		writeError(w, http.StatusNotFound, "not_found", "session not found")
		return
	}

	writeJSON(w, http.StatusOK, toSessionDetail(*sess))
}

// toSessionListItem 将 store.Session 转换为会话列表展示用的 SessionListItem。
func toSessionListItem(s store.Session) SessionListItem {
	item := SessionListItem{
		SessionID:    s.SessionID,
		MachineID:    s.MachineID,
		ProjectSlug:  s.ProjectSlug,
		Status:       s.Status,
		RegisteredAt: s.RegisteredAt,
	}
	if s.ClosedAt != nil {
		item.ClosedAt = s.ClosedAt
	}
	return item
}

// toSessionDetail 将 store.Session 转换为会话详情展示用的 SessionDetail。
func toSessionDetail(s store.Session) SessionDetail {
	d := SessionDetail{
		SessionID:      s.SessionID,
		MachineID:      s.MachineID,
		OS:             s.OS,
		ProjectSlug:    s.ProjectSlug,
		ProjectCwd:     s.ProjectCwd,
		TranscriptPath: s.TranscriptPath,
		LocalTracePath: s.LocalTracePath,
		Model:          s.Model,
		Source:         s.Source,
		Status:         s.Status,
		RegisteredAt:   s.RegisteredAt,
	}
	if s.ClosedAt != nil {
		d.ClosedAt = s.ClosedAt
	}
	if s.CloseReason != "" {
		d.CloseReason = s.CloseReason
	}
	return d
}
