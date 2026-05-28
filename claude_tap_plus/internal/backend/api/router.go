package api

import "net/http"

type Handlers struct {
	Issue *IssueHandler
}

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", Health)
	mux.HandleFunc("/api/issue/check", h.Issue.CheckIssues)
	return mux
}
