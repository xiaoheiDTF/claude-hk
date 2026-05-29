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

// --- Session request types ---

type RegisterSessionRequest struct {
	SessionID      string `json:"session_id"`
	MachineID      string `json:"machine_id"`
	OS             string `json:"os"`
	ProjectSlug    string `json:"project_slug"`
	ProjectCwd     string `json:"project_cwd"`
	TranscriptPath string `json:"transcript_path"`
	LocalTracePath string `json:"local_trace_path"`
	Model          string `json:"model"`
	Source         string `json:"source"`
}

type CloseSessionRequest struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason"`
}
