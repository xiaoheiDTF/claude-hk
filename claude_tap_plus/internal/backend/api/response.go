package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// --- Issue response types ---

type CheckIssuesResponse struct {
	Issues []IssueStatusItem `json:"issues"`
}

type IssueStatusItem struct {
	Number    int     `json:"number"`
	Status    string  `json:"status"`
	SessionID *string `json:"session_id"`
	ClaimedAt *string `json:"claimed_at"`
}

type ReleaseIssueResponse struct {
	Success  bool   `json:"success"`
	Released *bool  `json:"released,omitempty"`
	Error    string `json:"error,omitempty"`
}

type ClaimIssueResponse struct {
	Success   bool    `json:"success"`
	Status    string  `json:"status,omitempty"`
	ClaimedAt *string `json:"claimed_at,omitempty"`
	Error     string  `json:"error,omitempty"`
	ClaimedBy *string `json:"claimed_by,omitempty"`
	Message   string  `json:"message,omitempty"`
}

type ReleaseSessionResponse struct {
	Released []int `json:"released"`
	Count    int   `json:"count"`
}

type UpdateStatusResponse struct {
	Success       bool   `json:"success"`
	PreviousStatus string `json:"previous_status,omitempty"`
	NewStatus     string `json:"new_status,omitempty"`
	Error         string `json:"error,omitempty"`
}

// --- Session response types ---

type SessionListResponse struct {
	Sessions []SessionListItem `json:"sessions"`
}

type SessionListItem struct {
	SessionID    string     `json:"session_id"`
	MachineID    string     `json:"machine_id"`
	ProjectSlug  string     `json:"project_slug"`
	Status       string     `json:"status"`
	RegisteredAt time.Time  `json:"registered_at"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
}

type SessionDetail struct {
	SessionID      string     `json:"session_id"`
	MachineID      string     `json:"machine_id"`
	OS             string     `json:"os"`
	ProjectSlug    string     `json:"project_slug"`
	ProjectCwd     string     `json:"project_cwd"`
	TranscriptPath string     `json:"transcript_path"`
	LocalTracePath string     `json:"local_trace_path"`
	Model          string     `json:"model"`
	Source         string     `json:"source"`
	Status         string     `json:"status"`
	RegisteredAt   time.Time  `json:"registered_at"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	CloseReason    string     `json:"close_reason,omitempty"`
}

// --- Shared helpers ---

type APIError struct {
	Code    string `json:"error"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIError{Code: code, Message: message})
}
