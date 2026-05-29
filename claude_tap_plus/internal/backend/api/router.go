package api

import "net/http"

type Handlers struct {
	Issue   *IssueHandler
	Session *SessionHandler
}

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", Health)

	// Issue routes.
	mux.HandleFunc("/api/issue/check", h.Issue.CheckIssues)
	mux.HandleFunc("/api/issue/claim", h.Issue.ClaimIssue)
	mux.HandleFunc("/api/issue/release", h.Issue.ReleaseIssue)
	mux.HandleFunc("/api/issue/release-session", h.Issue.ReleaseSession)
	mux.HandleFunc("/api/issue/status", h.Issue.UpdateStatus)

	// Session routes.
	mux.HandleFunc("/api/session/register", h.Session.Register)
	mux.HandleFunc("/api/session/close", h.Session.Close)
	mux.HandleFunc("/api/sessions", h.Session.List)
	mux.HandleFunc("/api/session/", h.Session.Get)

	return mux
}
