package api

import (
	"encoding/json"
	"net/http"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
)

type IssueHandler struct {
	svc *service.IssueService
}

func NewIssueHandler(svc *service.IssueService) *IssueHandler {
	return &IssueHandler{svc: svc}
}

func (h *IssueHandler) CheckIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}

	var req CheckIssuesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.RepoFullName == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "repo_full_name is required")
		return
	}
	if req.IssueNumbers == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "issue_numbers is required")
		return
	}

	results, err := h.svc.Check(r.Context(), req.RepoFullName, req.IssueNumbers)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to check issues")
		return
	}

	items := make([]IssueStatusItem, len(results))
	for i, r := range results {
		items[i] = IssueStatusItem{
			Number:    r.Number,
			Status:    r.Status,
			SessionID: r.SessionID,
			ClaimedAt: r.ClaimedAt,
		}
	}

	if items == nil {
		items = []IssueStatusItem{}
	}

	writeJSON(w, http.StatusOK, CheckIssuesResponse{Issues: items})
}

func (h *IssueHandler) ClaimIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}

	var req ClaimIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.RepoFullName == "" || req.IssueNumber == 0 || req.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "repo_full_name, issue_number and session_id are required")
		return
	}

	result, err := h.svc.Claim(r.Context(), req.RepoFullName, req.IssueNumber, req.SessionID, req.IssueTitle)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to claim issue")
		return
	}

	if result.Success {
		writeJSON(w, http.StatusOK, ClaimIssueResponse{
			Success:   true,
			Status:    result.Status,
			ClaimedAt: result.ClaimedAt,
		})
		return
	}

	writeJSON(w, http.StatusConflict, ClaimIssueResponse{
		Success:   false,
		Error:     "already_claimed",
		ClaimedBy: result.ClaimedBy,
		ClaimedAt: result.ClaimedAt,
	})
}

func (h *IssueHandler) ReleaseIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}

	var req ReleaseIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.RepoFullName == "" || req.IssueNumber == 0 || req.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "repo_full_name, issue_number and session_id are required")
		return
	}

	released, err := h.svc.Release(r.Context(), req.RepoFullName, req.IssueNumber, req.SessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to release issue")
		return
	}

	if !released {
		writeJSON(w, http.StatusOK, ReleaseIssueResponse{Success: false, Error: "not_owner"})
		return
	}

	writeJSON(w, http.StatusOK, ReleaseIssueResponse{Success: true, Released: boolPtr(true)})
}

func (h *IssueHandler) ReleaseSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}

	var req ReleaseSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.SessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session_id is required")
		return
	}

	numbers, err := h.svc.ReleaseSession(r.Context(), req.SessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to release session issues")
		return
	}

	if numbers == nil {
		numbers = []int{}
	}

	writeJSON(w, http.StatusOK, ReleaseSessionResponse{Released: numbers, Count: len(numbers)})
}

func (h *IssueHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.RepoFullName == "" || req.IssueNumber == 0 || req.SessionID == "" || req.Status == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "repo_full_name, issue_number, session_id and status are required")
		return
	}

	result, err := h.svc.UpdateStatus(r.Context(), req.RepoFullName, req.IssueNumber, req.SessionID, req.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update status")
		return
	}

	if !result.Updated {
		status := http.StatusOK
		errMsg := ""
		if result.PreviousStatus == "" {
			errMsg = "not_found"
		} else {
			errMsg = "not_owner"
		}
		writeJSON(w, status, UpdateStatusResponse{
			Success:        false,
			PreviousStatus: result.PreviousStatus,
			Error:          errMsg,
		})
		return
	}

	writeJSON(w, http.StatusOK, UpdateStatusResponse{
		Success:        true,
		PreviousStatus: result.PreviousStatus,
		NewStatus:      result.NewStatus,
	})
}

func boolPtr(b bool) *bool { return &b }
