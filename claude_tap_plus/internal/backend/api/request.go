package api

type CheckIssuesRequest struct {
	RepoFullName string `json:"repo_full_name"`
	IssueNumbers []int  `json:"issue_numbers"`
}

type ClaimIssueRequest struct {
	RepoFullName string `json:"repo_full_name"`
	IssueNumber  int    `json:"issue_number"`
	SessionID    string `json:"session_id"`
	IssueTitle   string `json:"issue_title"`
}

type ReleaseIssueRequest struct {
	RepoFullName string `json:"repo_full_name"`
	IssueNumber  int    `json:"issue_number"`
	SessionID    string `json:"session_id"`
}

type ReleaseSessionRequest struct {
	SessionID string `json:"session_id"`
}

type UpdateStatusRequest struct {
	RepoFullName string `json:"repo_full_name"`
	IssueNumber  int    `json:"issue_number"`
	SessionID    string `json:"session_id"`
	Status       string `json:"status"`
}
