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
