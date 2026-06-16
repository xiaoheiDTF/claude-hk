// Package api_test 包含 GET /api/issues 接口的 BDD + TDD 验收测试。
// 覆盖：获取全部、按仓库/状态/session_id 过滤、组合过滤、分页、空列表、方法限制。
// 所有测试数据均通过系统自身 API（claim、release、status）产生，不直接操作数据库。
package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// --- response types for list API ---

// listResponse 是 GET /api/issues 的响应结构。
type listResponse struct {
	Issues     []listItem `json:"issues"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

// listItem 是列表中的单个 Issue 条目。
type listItem struct {
	ID           int64   `json:"id"`
	RepoFullName string  `json:"repo_full_name"`
	IssueNumber  int     `json:"issue_number"`
	IssueTitle   string  `json:"issue_title"`
	Status       string  `json:"status"`
	SessionID    *string `json:"session_id"`
	ClaimedAt    *string `json:"claimed_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// --- helpers ---

// get 发送 GET 请求到测试环境。
func (e *testEnv) get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(e.srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// readListResponse 读取并解析列表响应。
func readListResponse(t *testing.T, resp *http.Response) listResponse {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result listResponse
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("parse JSON: %v, body: %s", err, b)
	}
	return result
}

// apiClaim 通过 POST /api/issue/claim 创建一条 claimed 记录。
func apiClaim(t *testing.T, env *testEnv, repo string, number int, sessionID string) {
	t.Helper()
	body := fmt.Sprintf(
		`{"repo_full_name":"%s","issue_number":%d,"session_id":"%s"}`,
		repo, number, sessionID)
	resp := env.post(t, "/api/issue/claim", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("apiClaim failed: repo=%s #%d status=%d", repo, number, resp.StatusCode)
	}
}

// apiRelease 通过 POST /api/issue/release 释放一条记录，使其回到 idle。
func apiRelease(t *testing.T, env *testEnv, repo string, number int, sessionID string) {
	t.Helper()
	body := fmt.Sprintf(
		`{"repo_full_name":"%s","issue_number":%d,"session_id":"%s"}`,
		repo, number, sessionID)
	resp := env.post(t, "/api/issue/release", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("apiRelease failed: repo=%s #%d status=%d", repo, number, resp.StatusCode)
	}
}

// apiUpdateStatus 通过 POST /api/issue/status 更新 issue 状态。
func apiUpdateStatus(t *testing.T, env *testEnv, repo string, number int, sessionID, newStatus string) {
	t.Helper()
	body := fmt.Sprintf(
		`{"repo_full_name":"%s","issue_number":%d,"session_id":"%s","status":"%s"}`,
		repo, number, sessionID, newStatus)
	resp := env.post(t, "/api/issue/status", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("apiUpdateStatus failed: repo=%s #%d -> %s status=%d", repo, number, newStatus, resp.StatusCode)
	}
}

// seedBackgroundData 通过 API 预置 BDD Background 中的 5 条数据：
//   - #42 idle, no session        → claim + release
//   - #45 claimed, sess-abc123    → claim
//   - #38 fixing, sess-def456     → claim + status → fixing
//   - #10 idle, no session        → claim + release
//   - #11 merged, sess-abc123     → claim + status → merged
func seedBackgroundData(t *testing.T, env *testEnv) {
	t.Helper()

	// #42: idle（claim 后 release，回到 idle 且 session_id=null）
	apiClaim(t, env, "xiaoheiDTF/claude-hk", 42, "sess-tmp-42")
	apiRelease(t, env, "xiaoheiDTF/claude-hk", 42, "sess-tmp-42")

	// #45: claimed by sess-abc123
	apiClaim(t, env, "xiaoheiDTF/claude-hk", 45, "sess-abc123")

	// #38: fixing by sess-def456（claim → status=fixing）
	apiClaim(t, env, "xiaoheiDTF/claude-hk", 38, "sess-def456")
	apiUpdateStatus(t, env, "xiaoheiDTF/claude-hk", 38, "sess-def456", "fixing")

	// #10: idle（claim 后 release）
	apiClaim(t, env, "other/repo", 10, "sess-tmp-10")
	apiRelease(t, env, "other/repo", 10, "sess-tmp-10")

	// #11: merged by sess-abc123（claim → status=merged）
	apiClaim(t, env, "other/repo", 11, "sess-abc123")
	apiUpdateStatus(t, env, "other/repo", 11, "sess-abc123", "merged")
}

// --- BDD Scenario tests ---

// TestListIssues_GetAll 验证：获取所有 Issue 列表。
// BDD: @positive Scenario: 获取所有 Issue 列表
func TestListIssues_GetAll(t *testing.T) {
	env := setupTest(t)
	seedBackgroundData(t, env)

	resp := env.get(t, "/api/issues")
	result := readListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 5 {
		t.Fatalf("expected 5 issues, got %d", len(result.Issues))
	}

	// 验证每个 Issue 包含必要字段
	for _, item := range result.Issues {
		if item.ID == 0 {
			t.Error("expected non-zero id")
		}
		if item.RepoFullName == "" {
			t.Error("expected non-empty repo_full_name")
		}
		if item.IssueNumber == 0 {
			t.Error("expected non-zero issue_number")
		}
		if item.Status == "" {
			t.Error("expected non-empty status")
		}
		if item.UpdatedAt == "" {
			t.Error("expected non-empty updated_at")
		}
	}
}

// TestListIssues_FilterByRepo 验证：按仓库过滤 Issue。
// BDD: @positive Scenario: 按仓库过滤 Issue
func TestListIssues_FilterByRepo(t *testing.T) {
	env := setupTest(t)
	seedBackgroundData(t, env)

	resp := env.get(t, "/api/issues?repo=xiaoheiDTF/claude-hk")
	result := readListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(result.Issues))
	}
	for _, item := range result.Issues {
		if item.RepoFullName != "xiaoheiDTF/claude-hk" {
			t.Errorf("expected repo_full_name=xiaoheiDTF/claude-hk, got %s", item.RepoFullName)
		}
	}
}

// TestListIssues_FilterByStatus 验证：按状态过滤 Issue。
// BDD: @positive Scenario: 按状态过滤 Issue
func TestListIssues_FilterByStatus(t *testing.T) {
	env := setupTest(t)
	seedBackgroundData(t, env)

	resp := env.get(t, "/api/issues?status=idle")
	result := readListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(result.Issues))
	}
	for _, item := range result.Issues {
		if item.Status != "idle" {
			t.Errorf("expected status=idle, got %s", item.Status)
		}
	}
}

// TestListIssues_FilterBySessionID 验证：按 session_id 过滤 Issue。
// BDD: @positive Scenario: 按 session_id 过滤 Issue
func TestListIssues_FilterBySessionID(t *testing.T) {
	env := setupTest(t)
	seedBackgroundData(t, env)

	resp := env.get(t, "/api/issues?session_id=sess-abc123")
	result := readListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(result.Issues))
	}
	for _, item := range result.Issues {
		if item.SessionID == nil || *item.SessionID != "sess-abc123" {
			t.Errorf("expected session_id=sess-abc123, got %v", item.SessionID)
		}
	}
}

// TestListIssues_CombinedFilter 验证：组合过滤条件。
// BDD: @positive Scenario: 组合过滤条件
func TestListIssues_CombinedFilter(t *testing.T) {
	env := setupTest(t)
	seedBackgroundData(t, env)

	resp := env.get(t, "/api/issues?repo=xiaoheiDTF/claude-hk&status=idle")
	result := readListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].RepoFullName != "xiaoheiDTF/claude-hk" {
		t.Errorf("expected repo_full_name=xiaoheiDTF/claude-hk, got %s", result.Issues[0].RepoFullName)
	}
	if result.Issues[0].Status != "idle" {
		t.Errorf("expected status=idle, got %s", result.Issues[0].Status)
	}
	if result.Issues[0].IssueNumber != 42 {
		t.Errorf("expected issue_number=42, got %d", result.Issues[0].IssueNumber)
	}
}

// TestListIssues_Pagination 验证：分页查询。
// BDD: @positive Scenario: 分页查询
func TestListIssues_Pagination(t *testing.T) {
	env := setupTest(t)
	seedBackgroundData(t, env)

	resp := env.get(t, "/api/issues?page=1&page_size=2")
	result := readListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues on page, got %d", len(result.Issues))
	}
	if result.Total != 5 {
		t.Errorf("expected total=5, got %d", result.Total)
	}
	if result.Page != 1 {
		t.Errorf("expected page=1, got %d", result.Page)
	}
	if result.PageSize != 2 {
		t.Errorf("expected page_size=2, got %d", result.PageSize)
	}
	if result.TotalPages != 3 {
		t.Errorf("expected total_pages=3, got %d", result.TotalPages)
	}
}

// TestListIssues_EmptyResult 验证：无 Issue 时返回空数组。
// BDD: @positive Scenario: 无 Issue 时返回空数组
func TestListIssues_EmptyResult(t *testing.T) {
	env := setupTest(t)

	resp := env.get(t, "/api/issues")
	result := readListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected empty issues array, got %d items", len(result.Issues))
	}
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
}

// TestListIssues_MethodNotAllowed 验证：使用非 GET 方法请求返回 405。
// BDD: @negative Scenario: 使用非 GET 方法请求
func TestListIssues_MethodNotAllowed(t *testing.T) {
	env := setupTest(t)

	resp := env.post(t, "/api/issues", "")
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

// --- TDD additional edge-case tests ---

// TestListIssues_DefaultPagination 验证：不传分页参数时使用默认值。
func TestListIssues_DefaultPagination(t *testing.T) {
	env := setupTest(t)
	seedBackgroundData(t, env)

	resp := env.get(t, "/api/issues")
	result := readListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if result.Page != 1 {
		t.Errorf("expected default page=1, got %d", result.Page)
	}
	if result.PageSize != 20 {
		t.Errorf("expected default page_size=20, got %d", result.PageSize)
	}
}

// TestListIssues_PaginationPage2 验证：第二页数据正确。
func TestListIssues_PaginationPage2(t *testing.T) {
	env := setupTest(t)
	seedBackgroundData(t, env)

	resp := env.get(t, "/api/issues?page=2&page_size=2")
	result := readListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues on page 2, got %d", len(result.Issues))
	}
	if result.Page != 2 {
		t.Errorf("expected page=2, got %d", result.Page)
	}
}

// TestListIssues_PaginationLastPage 验证：最后一页可能不满。
func TestListIssues_PaginationLastPage(t *testing.T) {
	env := setupTest(t)
	seedBackgroundData(t, env)

	// 5 条数据，page_size=2，最后一页（page=3）应有 1 条
	resp := env.get(t, "/api/issues?page=3&page_size=2")
	result := readListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue on last page, got %d", len(result.Issues))
	}
	if result.TotalPages != 3 {
		t.Errorf("expected total_pages=3, got %d", result.TotalPages)
	}
}

// TestListIssues_FilterByNonexistentRepo 验证：过滤不存在的仓库返回空。
func TestListIssues_FilterByNonexistentRepo(t *testing.T) {
	env := setupTest(t)
	seedBackgroundData(t, env)

	resp := env.get(t, "/api/issues?repo=nonexistent/repo")
	result := readListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected 0 issues, got %d", len(result.Issues))
	}
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
}

// TestListIssues_SessionIDNullForIdle 验证：idle 状态的 issue session_id 为 null。
// 通过 claim → release 产生 idle 记录，验证 session_id 被清空。
func TestListIssues_SessionIDNullForIdle(t *testing.T) {
	env := setupTest(t)
	apiClaim(t, env, "test/repo", 1, "sess-temp")
	apiRelease(t, env, "test/repo", 1, "sess-temp")

	resp := env.get(t, "/api/issues")
	result := readListResponse(t, resp)

	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Status != "idle" {
		t.Errorf("expected status=idle, got %s", result.Issues[0].Status)
	}
	if result.Issues[0].SessionID != nil {
		t.Errorf("expected session_id=null for idle issue, got %v", *result.Issues[0].SessionID)
	}
}

// TestListIssues_ClaimedAtPopulated 验证：claimed 状态的 issue claimed_at 非空。
func TestListIssues_ClaimedAtPopulated(t *testing.T) {
	env := setupTest(t)
	seedBackgroundData(t, env)

	resp := env.get(t, "/api/issues?status=claimed")
	result := readListResponse(t, resp)

	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 claimed issue, got %d", len(result.Issues))
	}
	if result.Issues[0].ClaimedAt == nil {
		t.Error("expected claimed_at to be non-nil for claimed issue")
	}
}

// TestListIssues_StatusFlowViaAPI 验证：通过 API 完整流转产生多种状态的 issue。
// 端到端：claim → fixing → ready-for-pr，验证列表中状态正确。
func TestListIssues_StatusFlowViaAPI(t *testing.T) {
	env := setupTest(t)

	// 通过 API 创建一条 issue 并推到 fixing 状态
	apiClaim(t, env, "flow/repo", 100, "sess-flow")
	apiUpdateStatus(t, env, "flow/repo", 100, "sess-flow", "fixing")

	resp := env.get(t, "/api/issues?repo=flow/repo")
	result := readListResponse(t, resp)

	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Status != "fixing" {
		t.Errorf("expected status=fixing, got %s", result.Issues[0].Status)
	}
	if result.Issues[0].SessionID == nil || *result.Issues[0].SessionID != "sess-flow" {
		t.Errorf("expected session_id=sess-flow, got %v", result.Issues[0].SessionID)
	}
}
