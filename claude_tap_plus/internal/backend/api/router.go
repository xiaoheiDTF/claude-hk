package api

import "net/http"

type Handlers struct {
	Issue *IssueHandler
}

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", Health)
	mux.HandleFunc("/api/issue/check", h.Issue.CheckIssues)
	mux.HandleFunc("/api/issue/release", h.Issue.ReleaseIssue)
	mux.HandleFunc("/api/issue/release-session", h.Issue.ReleaseSession)
	return mux
}
