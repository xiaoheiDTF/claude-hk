package api

import (
	"encoding/json"
	"net/http"
)

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

type ReleaseSessionResponse struct {
	Released []int `json:"released"`
	Count    int   `json:"count"`
}

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
