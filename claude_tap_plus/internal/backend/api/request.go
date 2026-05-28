package api

type CheckIssuesRequest struct {
	RepoFullName string `json:"repo_full_name"`
	IssueNumbers []int  `json:"issue_numbers"`
}
