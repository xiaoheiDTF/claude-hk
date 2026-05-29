package api

import (
	"encoding/json"
	"net/http"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
)

type SessionHandler struct {
	svc *service.SessionService
}

func NewSessionHandler(svc *service.SessionService) *SessionHandler {
	return &SessionHandler{svc: svc}
}

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

	if req.SessionID == "" || req.MachineID == "" || req.ProjectSlug == "" || req.TranscriptPath == "" {
		writeError(w, http.StatusBadRequest, "invalid_request",
			"session_id, machine_id, project_slug and transcript_path are required")
		return
	}

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

	if err := h.svc.Register(r.Context(), sess); err != nil {
		if err == store.ErrSessionExists {
			writeError(w, http.StatusConflict, "session_exists", "session already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to register session")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "registered"})
}

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

	if err := h.svc.Close(r.Context(), req.SessionID, req.Reason); err != nil {
		if err == store.ErrSessionNotFound {
			writeError(w, http.StatusNotFound, "not_found", "session not found or already closed")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to close session")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "closed"})
}

func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

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

	sessions, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list sessions")
		return
	}

	if sessions == nil {
		sessions = []store.Session{}
	}

	items := make([]SessionListItem, len(sessions))
	for i, s := range sessions {
		items[i] = toSessionListItem(s)
	}

	writeJSON(w, http.StatusOK, SessionListResponse{Sessions: items})
}

func (h *SessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	sessionID := r.URL.Path[len("/api/session/"):]
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session_id is required")
		return
	}

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

func toSessionListItem(s store.Session) SessionListItem {
	item := SessionListItem{
		SessionID:   s.SessionID,
		MachineID:   s.MachineID,
		ProjectSlug: s.ProjectSlug,
		Status:      s.Status,
		RegisteredAt: s.RegisteredAt,
	}
	if s.ClosedAt != nil {
		item.ClosedAt = s.ClosedAt
	}
	return item
}

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
