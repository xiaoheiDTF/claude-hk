// Package api_test 包含 GET /api/session/{id}/issues 接口的 BDD + TDD 验收测试。
// 覆盖：获取会话关联 Issue、会话无 Issue 返回空、不存在会话 404、方法限制。
// 所有测试数据均通过 API 接口产生（session register + issue claim/status）。
package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// --- response types for session issues API ---

// sessionIssuesResponse 是 GET /api/session/{id}/issues 的响应结构。
type sessionIssuesResponse struct {
	Issues []struct {
		ID           int64   `json:"id"`
		RepoFullName string  `json:"repo_full_name"`
		IssueNumber  int     `json:"issue_number"`
		IssueTitle   *string `json:"issue_title"`
		Status       string  `json:"status"`
		SessionID    *string `json:"session_id"`
		ClaimedAt    *string `json:"claimed_at"`
		UpdatedAt    string  `json:"updated_at"`
	} `json:"issues"`
}

// --- helpers ---

// readSessionIssuesResponse 读取并解析会话 Issue 列表响应。
func readSessionIssuesResponse(t *testing.T, resp *http.Response) sessionIssuesResponse {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result sessionIssuesResponse
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("parse JSON: %v, body: %s", err, b)
	}
	return result
}

// seedSessionIssuesData 通过 API 预置 BDD Background 数据：
//   - 注册 session-abc123 和 session-def456
//   - #42 claimed by session-abc123
//   - #45 fixing by session-abc123
//   - #10 claimed by session-def456
func seedSessionIssuesData(t *testing.T, env *testEnv) {
	t.Helper()
	// 注册会话（session register 要求 session_id 唯一）
	env.post(t, "/api/session/register",
		`{"session_id":"session-abc123","machine_id":"user@host","os":"linux","project_slug":"proj/x","project_cwd":"/tmp","transcript_path":"/tmp/t.jsonl"}`)
	env.post(t, "/api/session/register",
		`{"session_id":"session-def456","machine_id":"dev@host","os":"linux","project_slug":"proj/y","project_cwd":"/tmp","transcript_path":"/tmp/t.jsonl"}`)

	// #42 claimed by session-abc123
	apiClaim(t, env, "xiaoheiDTF/claude-hk", 42, "session-abc123")
	// #45 claimed by session-abc123 → fixing
	apiClaim(t, env, "xiaoheiDTF/claude-hk", 45, "session-abc123")
	apiUpdateStatus(t, env, "xiaoheiDTF/claude-hk", 45, "session-abc123", "fixing")
	// #10 claimed by session-def456
	apiClaim(t, env, "other/repo", 10, "session-def456")
}

// --- BDD Scenario tests ---

// TestSessionIssues_GetAll 验证：获取会话领取的所有 Issue。
// BDD: @positive Scenario: 获取会话领取的所有 Issue
func TestSessionIssues_GetAll(t *testing.T) {
	env := setupTest(t)
	seedSessionIssuesData(t, env)

	resp := env.get(t, "/api/session/session-abc123/issues")
	result := readSessionIssuesResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(result.Issues))
	}

	// 验证所有 Issue 的 session_id
	for _, issue := range result.Issues {
		if issue.SessionID == nil || *issue.SessionID != "session-abc123" {
			t.Errorf("expected session_id=session-abc123, got %v", issue.SessionID)
		}
	}

	// 验证具体 Issue 状态
	byNumber := map[int]string{}
	for _, issue := range result.Issues {
		byNumber[issue.IssueNumber] = issue.Status
	}
	if byNumber[42] != "claimed" {
		t.Errorf("expected #42 status=claimed, got %s", byNumber[42])
	}
	if byNumber[45] != "fixing" {
		t.Errorf("expected #45 status=fixing, got %s", byNumber[45])
	}
}

// TestSessionIssues_EmptySession 验证：会话无领取 Issue 返回空数组。
// BDD: @positive Scenario: 会话无领取 Issue
func TestSessionIssues_EmptySession(t *testing.T) {
	env := setupTest(t)

	// 注册一个没有任何 issue 的会话
	env.post(t, "/api/session/register",
		fmt.Sprintf(`{"session_id":"session-empty","machine_id":"user@h","os":"linux","project_slug":"p","project_cwd":"/tmp","transcript_path":"/tmp/t.jsonl"}`))

	resp := env.get(t, "/api/session/session-empty/issues")
	result := readSessionIssuesResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected empty issues array, got %d items", len(result.Issues))
	}
}

// TestSessionIssues_SessionNotFound 验证：获取不存在的会话返回 404。
// BDD: @negative Scenario: 获取不存在的会话
func TestSessionIssues_SessionNotFound(t *testing.T) {
	env := setupTest(t)

	resp := env.get(t, "/api/session/session-not-exist/issues")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var errResp struct {
		Code    string `json:"error"`
		Message string `json:"message"`
	}
	readJSON(t, resp, &errResp)
	if errResp.Code != "not_found" {
		t.Errorf("expected error=not_found, got %s", errResp.Code)
	}
}

// TestSessionIssues_MethodNotAllowed 验证：使用非 GET 方法请求返回 405。
// BDD: @negative Scenario: 使用非 GET 方法请求
func TestSessionIssues_MethodNotAllowed(t *testing.T) {
	env := setupTest(t)
	seedSessionIssuesData(t, env)

	resp := env.post(t, "/api/session/session-abc123/issues", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}

	var errResp struct {
		Code    string `json:"error"`
		Message string `json:"message"`
	}
	readJSON(t, resp, &errResp)
	if errResp.Code != "method_not_allowed" {
		t.Errorf("expected error=method_not_allowed, got %s", errResp.Code)
	}
}
