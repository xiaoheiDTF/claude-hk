// Package backend_test 包含后端 Issue API 的验收测试，覆盖 check、claim、release、status 等接口。
package backend_test

import (
	"database/sql"
	"fmt"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/api"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
)

// --- helpers ---

// testEnv 是后端测试的通用环境，包含测试服务器和 SQLite 存储。
type testEnv struct {
	srv   *httptest.Server
	store *store.SQLiteStore
}

// setupTest 创建后端测试环境，自动清理临时数据库。
func setupTest(t *testing.T) *testEnv {
	t.Helper()

	f, err := os.CreateTemp("", "test-check-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()

	s, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatal(err)
	}

	t.Cleanup(func() {
		s.Close()
		os.Remove(dbPath)
	})

	issueSvc := service.NewIssueService(s.Issues())
	sessionSvc := service.NewSessionService(s.Sessions())
	router := api.NewRouter(api.Handlers{
		Issue:   api.NewIssueHandler(issueSvc),
		Session: api.NewSessionHandler(sessionSvc),
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &testEnv{srv: srv, store: s}
}

// post 发送 POST 请求到测试环境。
func (e *testEnv) post(t *testing.T, path, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(e.srv.URL+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// readJSON 读取 HTTP 响应并解析为 JSON。
func readJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("parse JSON: %v, body: %s", err, b)
	}
}

// seedIssue 在数据库中预置一条 issue 记录。
func seedIssue(db *sql.DB, repo string, number int, status, sessionID, claimedAt string) {
	db.Exec(
		`INSERT INTO issue_claims (repo_full_name, issue_number, status, session_id, claimed_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		repo, number, status,
		nilIfEmpty(sessionID), nilIfEmpty(claimedAt),
	)
}

// nilIfEmpty 将空字符串转为 nil，用于数据库插入。
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// --- tests ---

// TestHealth 验证：/health 健康检查接口返回 200 和 {"status":"ok"}。
func TestHealth(t *testing.T) {
	env := setupTest(t)

	resp, err := http.Get(env.srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	readJSON(t, resp, &result)
	if result["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", result["status"])
	}
}

// TestCheckIssues_EmptyArray 验证：传入空 issue 数组时返回空结果。
func TestCheckIssues_EmptyArray(t *testing.T) {
	env := setupTest(t)

	resp := env.post(t, "/api/issue/check",
		`{"repo_full_name":"test/repo","issue_numbers":[]}`)

	var result struct {
		Issues []any `json:"issues"`
	}
	readJSON(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected empty issues, got %d", len(result.Issues))
	}
}

// TestCheckIssues_NewIssuesAutoCreatedAsIdle 验证：首次出现的 issue 自动创建为 idle 状态。
func TestCheckIssues_NewIssuesAutoCreatedAsIdle(t *testing.T) {
	env := setupTest(t)

	resp := env.post(t, "/api/issue/check",
		`{"repo_full_name":"test/repo","issue_numbers":[1,2,3]}`)

	var result struct {
		Issues []struct {
			Number    int     `json:"number"`
			Status    string  `json:"status"`
			SessionID *string `json:"session_id"`
			ClaimedAt *string `json:"claimed_at"`
		} `json:"issues"`
	}
	readJSON(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(result.Issues))
	}

	for _, issue := range result.Issues {
		if issue.Status != "idle" {
			t.Errorf("issue %d: expected idle, got %s", issue.Number, issue.Status)
		}
		if issue.SessionID != nil {
			t.Errorf("issue %d: expected null session_id, got %v", issue.Number, *issue.SessionID)
		}
		if issue.ClaimedAt != nil {
			t.Errorf("issue %d: expected null claimed_at, got %v", issue.Number, *issue.ClaimedAt)
		}
	}
}

// TestCheckIssues_ClaimedAndMerged 验证：check 接口返回不同状态 issue 的正确信息。
// 覆盖 idle、claimed（有 session_id 和 claimed_at）、merged 三种状态。
func TestCheckIssues_ClaimedAndMerged(t *testing.T) {
	env := setupTest(t)
	db := env.store.DB()

	seedIssue(db, "test/repo", 10, "claimed", "sess_abc", "2026-05-26T10:00:00Z")
	seedIssue(db, "test/repo", 11, "merged", "", "")

	resp := env.post(t, "/api/issue/check",
		`{"repo_full_name":"test/repo","issue_numbers":[9,10,11,12]}`)

	var result struct {
		Issues []struct {
			Number    int     `json:"number"`
			Status    string  `json:"status"`
			SessionID *string `json:"session_id"`
			ClaimedAt *string `json:"claimed_at"`
		} `json:"issues"`
	}
	readJSON(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 4 {
		t.Fatalf("expected 4 issues, got %d", len(result.Issues))
	}

	byNumber := map[int]*struct {
		Number    int     `json:"number"`
		Status    string  `json:"status"`
		SessionID *string `json:"session_id"`
		ClaimedAt *string `json:"claimed_at"`
	}{}
	for i := range result.Issues {
		byNumber[result.Issues[i].Number] = &result.Issues[i]
	}

	// #9: 新 issue → idle
	if byNumber[9].Status != "idle" {
		t.Errorf("issue 9: expected idle, got %s", byNumber[9].Status)
	}

	// #10: claimed
	if byNumber[10].Status != "claimed" {
		t.Errorf("issue 10: expected claimed, got %s", byNumber[10].Status)
	}
	if byNumber[10].SessionID == nil || *byNumber[10].SessionID != "sess_abc" {
		t.Errorf("issue 10: expected session_id=sess_abc, got %v", byNumber[10].SessionID)
	}
	if byNumber[10].ClaimedAt == nil {
		t.Error("issue 10: expected non-nil claimed_at")
	}

	// #11: merged
	if byNumber[11].Status != "merged" {
		t.Errorf("issue 11: expected merged, got %s", byNumber[11].Status)
	}
	if byNumber[11].SessionID != nil {
		t.Errorf("issue 11: expected null session_id, got %v", byNumber[11].SessionID)
	}

	// #12: 新 issue → idle
	if byNumber[12].Status != "idle" {
		t.Errorf("issue 12: expected idle, got %s", byNumber[12].Status)
	}
}

// TestCheckIssues_MissingRepoFullName 验证：缺少 repo_full_name 返回 400。
func TestCheckIssues_MissingRepoFullName(t *testing.T) {
	env := setupTest(t)

	resp := env.post(t, "/api/issue/check",
		`{"issue_numbers":[1]}`)

	var errResp struct {
		Code    string `json:"error"`
		Message string `json:"message"`
	}
	readJSON(t, resp, &errResp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if errResp.Code != "invalid_request" {
		t.Errorf("expected error=invalid_request, got %s", errResp.Code)
	}
}

// TestCheckIssues_MissingIssueNumbers 验证：缺少 issue_numbers 返回 400。
func TestCheckIssues_MissingIssueNumbers(t *testing.T) {
	env := setupTest(t)

	resp := env.post(t, "/api/issue/check",
		`{"repo_full_name":"test/repo"}`)

	var errResp struct {
		Code    string `json:"error"`
		Message string `json:"message"`
	}
	readJSON(t, resp, &errResp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if errResp.Code != "invalid_request" {
		t.Errorf("expected error=invalid_request, got %s", errResp.Code)
	}
}

// TestCheckIssues_InvalidJSON 验证：非法 JSON 请求体返回 400。
func TestCheckIssues_InvalidJSON(t *testing.T) {
	env := setupTest(t)

	resp := env.post(t, "/api/issue/check", `not json`)

	var errResp struct {
		Code    string `json:"error"`
		Message string `json:"message"`
	}
	readJSON(t, resp, &errResp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if errResp.Code != "invalid_request" {
		t.Errorf("expected error=invalid_request, got %s", errResp.Code)
	}
}

// TestCheckIssues_MethodNotAllowed 验证：GET 请求访问 check 接口返回 405。
func TestCheckIssues_MethodNotAllowed(t *testing.T) {
	env := setupTest(t)

	resp, err := http.Get(env.srv.URL + "/api/issue/check")
	if err != nil {
		t.Fatal(err)
	}

	var errResp struct {
		Code    string `json:"error"`
		Message string `json:"message"`
	}
	readJSON(t, resp, &errResp)

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
	if errResp.Code != "method_not_allowed" {
		t.Errorf("expected error=method_not_allowed, got %s", errResp.Code)
	}
}

// TestCheckIssues_IdempotentOnRepeatedCalls 验证：重复调用 check 接口是幂等的。
func TestCheckIssues_IdempotentOnRepeatedCalls(t *testing.T) {
	env := setupTest(t)

	// 第一次调用创建 idle 记录
	env.post(t, "/api/issue/check",
		`{"repo_full_name":"test/repo","issue_numbers":[1,2]}`)

	// 第二次调用应返回相同的 idle 记录
	resp := env.post(t, "/api/issue/check",
		`{"repo_full_name":"test/repo","issue_numbers":[1,2]}`)

	var result struct {
		Issues []struct {
			Number int    `json:"number"`
			Status string `json:"status"`
		} `json:"issues"`
	}
	readJSON(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(result.Issues))
	}
	for _, issue := range result.Issues {
		if issue.Status != "idle" {
			t.Errorf("issue %d: expected idle, got %s", issue.Number, issue.Status)
		}
	}
}

// --- Release Issue tests ---
// POST /api/issue/release — 单个 issue 释放
// 验收标准：
//   - 领取者可释放自己的 issue
//   - 非领取者无法释放（返回 not_owner）
//   - merged/rejected 终态不被释放
//   - 参数缺失时返回错误

// TestReleaseIssue 验证单个 issue 释放接口。
func TestReleaseIssue(t *testing.T) {
	t.Run("owner_can_release", func(t *testing.T) {
		// 验收：领取者可释放自己的 issue
		// 预置 #10 为 claimed（sess_abc 领取），用 sess_abc 调 release → success=true
		// 释放后再 check 验证 #10 回到 idle、session_id=null
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "claimed", "sess_abc", "2026-05-26T10:00:00Z")

		resp := env.post(t, "/api/issue/release",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_abc"}`)

		var result struct {
			Success  bool  `json:"success"`
			Released *bool `json:"released"`
			Error    string `json:"error"`
		}
		readJSON(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if !result.Success {
			t.Fatalf("expected success=true, got false (error: %s)", result.Error)
		}
		if result.Released == nil || !*result.Released {
			t.Fatal("expected released=true")
		}

		// 验证 issue 状态已回到 idle
		checkResp := env.post(t, "/api/issue/check",
			`{"repo_full_name":"test/repo","issue_numbers":[10]}`)
		var check struct {
			Issues []struct {
				Status    string  `json:"status"`
				SessionID *string `json:"session_id"`
			} `json:"issues"`
		}
		readJSON(t, checkResp, &check)
		if len(check.Issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(check.Issues))
		}
		if check.Issues[0].Status != "idle" {
			t.Errorf("expected idle after release, got %s", check.Issues[0].Status)
		}
		if check.Issues[0].SessionID != nil {
			t.Error("session_id should be null after release")
		}
	})

	t.Run("non_owner_cannot_release", func(t *testing.T) {
		// 验收：非领取者无法释放（返回 not_owner）
		// 预置 #10 为 claimed（sess_abc 领取），用 sess_other 调 release → success=false, error=not_owner
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "claimed", "sess_abc", "2026-05-26T10:00:00Z")

		resp := env.post(t, "/api/issue/release",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_other"}`)

		var result struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		readJSON(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if result.Success {
			t.Fatal("expected success=false for non-owner")
		}
		if result.Error != "not_owner" {
			t.Fatalf("expected error=not_owner, got %s", result.Error)
		}
	})

	t.Run("terminal_merged_not_released", func(t *testing.T) {
		// 验收：merged 状态的 issue 不被释放
		// 预置 #10 为 merged（sess_abc 领取），用 sess_abc 调 release → success=false
		// 即使是领取者本人，终态也不可释放
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "merged", "sess_abc", "2026-05-26T10:00:00Z")

		resp := env.post(t, "/api/issue/release",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_abc"}`)

		var result struct {
			Success bool `json:"success"`
		}
		readJSON(t, resp, &result)

		if result.Success {
			t.Fatal("merged issue should not be released")
		}
	})

	t.Run("terminal_rejected_not_released", func(t *testing.T) {
		// 验收：rejected 状态的 issue 不被释放
		// 预置 #10 为 rejected（sess_abc 领取），用 sess_abc 调 release → success=false
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "rejected", "sess_abc", "2026-05-26T10:00:00Z")

		resp := env.post(t, "/api/issue/release",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_abc"}`)

		var result struct {
			Success bool `json:"success"`
		}
		readJSON(t, resp, &result)

		if result.Success {
			t.Fatal("rejected issue should not be released")
		}
	})

	t.Run("non_terminal_statuses_releasable", func(t *testing.T) {
		// 补充：非终态（claimed/fixing/ready-for-pr/pr-created/testing/reviewing）都可释放
		// 预置 #10 为 fixing（sess_abc），释放后验证回到 idle
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "fixing", "sess_abc", "2026-05-26T10:00:00Z")

		resp := env.post(t, "/api/issue/release",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_abc"}`)

		var result struct {
			Success  bool  `json:"success"`
			Released *bool `json:"released"`
		}
		readJSON(t, resp, &result)

		if !result.Success {
			t.Fatal("fixing status should be releasable by owner")
		}
		if result.Released == nil || !*result.Released {
			t.Fatal("expected released=true")
		}
	})

	t.Run("nonexistent_issue_returns_not_owner", func(t *testing.T) {
		// 补充：issue 不存在时返回 success=false（无记录可释放）
		env := setupTest(t)

		resp := env.post(t, "/api/issue/release",
			`{"repo_full_name":"test/repo","issue_number":999,"session_id":"sess_abc"}`)

		var result struct {
			Success bool `json:"success"`
		}
		readJSON(t, resp, &result)

		if result.Success {
			t.Fatal("nonexistent issue should not be released")
		}
	})

	t.Run("missing_params_returns_400", func(t *testing.T) {
		// 验收：参数缺失时返回错误响应
		// 缺少 session_id → 400
		env := setupTest(t)

		resp := env.post(t, "/api/issue/release",
			`{"repo_full_name":"test/repo","issue_number":10}`)

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})
}

// --- Release Session tests ---
// POST /api/issue/release-session — 按 session 批量释放
// 验收标准：
//   - 释放该 session 所有未终态的 issue
//   - 已合并/打回的 issue 不受影响
//   - 无领取记录的 session 返回空列表

// TestReleaseSession 验证按 session 批量释放接口。
func TestReleaseSession(t *testing.T) {
	t.Run("releases_all_non_terminal", func(t *testing.T) {
		// 验收：释放该 session 所有未终态的 issue
		// 验收：已合并/打回的 issue 不受影响
		// 预置 sess_abc 领取 3 个 issue：
		//   #10 claimed（非终态，应释放）
		//   #11 fixing（非终态，应释放）
		//   #12 merged（终态，不应释放）
		// 调 release-session → released=[10,11], count=2
		// 再 check 验证 #10/#11 回到 idle，#12 仍为 merged
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "claimed", "sess_abc", "2026-05-26T10:00:00Z")
		seedIssue(db, "test/repo", 11, "fixing", "sess_abc", "2026-05-26T10:00:00Z")
		seedIssue(db, "test/repo", 12, "merged", "sess_abc", "2026-05-26T10:00:00Z")

		resp := env.post(t, "/api/issue/release-session",
			`{"session_id":"sess_abc"}`)

		var result struct {
			Released []int `json:"released"`
			Count    int   `json:"count"`
		}
		readJSON(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if result.Count != 2 {
			t.Fatalf("expected 2 released, got %d", result.Count)
		}

		releasedSet := map[int]bool{}
		for _, n := range result.Released {
			releasedSet[n] = true
		}
		if !releasedSet[10] || !releasedSet[11] {
			t.Fatalf("expected issues 10 and 11 in released, got %v", result.Released)
		}

		// 验证 #12（merged）不受影响
		checkResp := env.post(t, "/api/issue/check",
			`{"repo_full_name":"test/repo","issue_numbers":[10,11,12]}`)
		var check struct {
			Issues []struct {
				Number int    `json:"number"`
				Status string `json:"status"`
			} `json:"issues"`
		}
		readJSON(t, checkResp, &check)

		byNum := map[int]string{}
		for _, iss := range check.Issues {
			byNum[iss.Number] = iss.Status
		}
		if byNum[10] != "idle" || byNum[11] != "idle" {
			t.Fatalf("expected 10,11 idle after release, got %v", byNum)
		}
		if byNum[12] != "merged" {
			t.Fatalf("expected 12 still merged, got %s", byNum[12])
		}
	})

	t.Run("rejected_not_released", func(t *testing.T) {
		// 验收：已打回的 issue 不受影响
		// 预置 sess_abc 领取 #10（rejected），调 release-session → count=0
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "rejected", "sess_abc", "2026-05-26T10:00:00Z")

		resp := env.post(t, "/api/issue/release-session",
			`{"session_id":"sess_abc"}`)

		var result struct {
			Count int `json:"count"`
		}
		readJSON(t, resp, &result)

		if result.Count != 0 {
			t.Fatalf("rejected should not be released, got count=%d", result.Count)
		}
	})

	t.Run("no_claims_returns_empty", func(t *testing.T) {
		// 验收：无领取记录的 session 返回空列表
		// 无任何预置数据，调 release-session → released=[], count=0
		env := setupTest(t)

		resp := env.post(t, "/api/issue/release-session",
			`{"session_id":"sess_none"}`)

		var result struct {
			Released []int `json:"released"`
			Count    int   `json:"count"`
		}
		readJSON(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if result.Count != 0 {
			t.Errorf("expected 0 released, got %d", result.Count)
		}
		if len(result.Released) != 0 {
			t.Errorf("expected empty released array, got %v", result.Released)
		}
	})

	t.Run("only_releases_own_session", func(t *testing.T) {
		// 补充：只释放目标 session 的 issue，不影响其他 session
		// 预置 #10（sess_abc claimed）和 #11（sess_other claimed）
		// 调 release-session(sess_abc) → 只释放 #10，#11 不受影响
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "claimed", "sess_abc", "2026-05-26T10:00:00Z")
		seedIssue(db, "test/repo", 11, "claimed", "sess_other", "2026-05-26T10:00:00Z")

		resp := env.post(t, "/api/issue/release-session",
			`{"session_id":"sess_abc"}`)

		var result struct {
			Released []int `json:"released"`
			Count    int   `json:"count"`
		}
		readJSON(t, resp, &result)

		if result.Count != 1 {
			t.Fatalf("expected 1 released, got %d", result.Count)
		}
		if len(result.Released) != 1 || result.Released[0] != 10 {
			t.Fatalf("expected [10], got %v", result.Released)
		}

		// 验证 #11 仍为 claimed（sess_other 的）
		checkResp := env.post(t, "/api/issue/check",
			`{"repo_full_name":"test/repo","issue_numbers":[11]}`)
		var check struct {
			Issues []struct {
				Status    string  `json:"status"`
				SessionID *string `json:"session_id"`
			} `json:"issues"`
		}
		readJSON(t, checkResp, &check)
		if len(check.Issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(check.Issues))
		}
		if check.Issues[0].Status != "claimed" {
			t.Errorf("expected #11 still claimed, got %s", check.Issues[0].Status)
		}
		if check.Issues[0].SessionID == nil || *check.Issues[0].SessionID != "sess_other" {
			t.Error("expected #11 still owned by sess_other")
		}
	})

	t.Run("missing_session_id_returns_400", func(t *testing.T) {
		// 验收：参数缺失时返回错误响应
		// 缺少 session_id → 400
		env := setupTest(t)

		resp := env.post(t, "/api/issue/release-session", `{}`)

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})
}

// --- D2: Issue Status Update tests ---
// POST /api/issue/status — 更新 issue 状态（D2-D6 使用）
// 验收标准：
//   - 领取者可将 issue 状态从 claimed 更新为 fixing
//   - 非 owner session 无法更新状态
//   - 不存在的 issue 返回 not_found
//   - 参数缺失时返回 400
//   - 同一状态幂等更新返回成功
//   - 完整流程：claim → fixing 验证状态正确流转

// TestUpdateStatus 验证 Issue 状态更新接口。
func TestUpdateStatus(t *testing.T) {
	t.Run("owner_can_update_to_fixing", func(t *testing.T) {
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "claimed", "sess_abc", "2026-05-27T10:00:00Z")

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_abc","status":"fixing"}`)

		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
			Error          string `json:"error"`
		}
		readJSON(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if !result.Success {
			t.Fatalf("expected success=true, got false (error: %s)", result.Error)
		}
		if result.PreviousStatus != "claimed" {
			t.Errorf("expected previous_status=claimed, got %s", result.PreviousStatus)
		}
		if result.NewStatus != "fixing" {
			t.Errorf("expected new_status=fixing, got %s", result.NewStatus)
		}

		statuses := checkStatuses(t, env, "test/repo", []int{10})
		if statuses[10] != "fixing" {
			t.Errorf("expected fixing after update, got %s", statuses[10])
		}
	})

	t.Run("non_owner_cannot_update", func(t *testing.T) {
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "claimed", "sess_abc", "2026-05-27T10:00:00Z")

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_other","status":"fixing"}`)

		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			Error          string `json:"error"`
		}
		readJSON(t, resp, &result)

		if result.Success {
			t.Fatal("expected success=false for non-owner")
		}
		if result.Error != "not_owner" {
			t.Errorf("expected error=not_owner, got %s", result.Error)
		}

		statuses := checkStatuses(t, env, "test/repo", []int{10})
		if statuses[10] != "claimed" {
			t.Errorf("expected status unchanged (claimed), got %s", statuses[10])
		}
	})

	t.Run("nonexistent_issue_returns_not_found", func(t *testing.T) {
		env := setupTest(t)

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":999,"session_id":"sess_abc","status":"fixing"}`)

		var result struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		readJSON(t, resp, &result)

		if result.Success {
			t.Fatal("expected success=false for nonexistent issue")
		}
		if result.Error != "not_found" {
			t.Errorf("expected error=not_found, got %s", result.Error)
		}
	})

	t.Run("missing_params_returns_400", func(t *testing.T) {
		env := setupTest(t)

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_abc"}`)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 missing status, got %d", resp.StatusCode)
		}

		resp = env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"status":"fixing"}`)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 missing session_id, got %d", resp.StatusCode)
		}

		resp = env.post(t, "/api/issue/status",
			`{"issue_number":10,"session_id":"sess_abc","status":"fixing"}`)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 missing repo, got %d", resp.StatusCode)
		}
	})

	t.Run("idempotent_same_status", func(t *testing.T) {
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "claimed", "sess_abc", "2026-05-27T10:00:00Z")

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_abc","status":"fixing"}`)
		var r1 struct{ Success bool `json:"success"` }
		readJSON(t, resp, &r1)
		if !r1.Success {
			t.Fatal("first update should succeed")
		}

		resp = env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_abc","status":"fixing"}`)
		var r2 struct {
			Success   bool   `json:"success"`
			NewStatus string `json:"new_status"`
		}
		readJSON(t, resp, &r2)
		if !r2.Success {
			t.Fatal("idempotent update should succeed")
		}
		if r2.NewStatus != "fixing" {
			t.Errorf("expected new_status=fixing, got %s", r2.NewStatus)
		}
	})

	t.Run("full_claim_to_fix_flow", func(t *testing.T) {
		env := setupTest(t)

		// 1. check 创建 idle
		env.post(t, "/api/issue/check",
			`{"repo_full_name":"test/repo","issue_numbers":[20]}`)
		s := checkStatuses(t, env, "test/repo", []int{20})
		if s[20] != "idle" {
			t.Fatalf("step 1: expected idle, got %s", s[20])
		}

		// 2. claim
		code, ok, _ := claimIssue(t, env, "test/repo", 20, "sess_fix")
		if code != http.StatusOK || !ok {
			t.Fatalf("step 2: claim failed")
		}
		s = checkStatuses(t, env, "test/repo", []int{20})
		if s[20] != "claimed" {
			t.Fatalf("step 2: expected claimed, got %s", s[20])
		}

		// 3. 标记 fixing
		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":20,"session_id":"sess_fix","status":"fixing"}`)
		var r struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &r)
		if !r.Success {
			t.Fatal("step 3: update to fixing failed")
		}
		if r.PreviousStatus != "claimed" {
			t.Errorf("step 3: expected previous=claimed, got %s", r.PreviousStatus)
		}

		// 4. 验证最终状态
		s = checkStatuses(t, env, "test/repo", []int{20})
		if s[20] != "fixing" {
			t.Fatalf("step 4: expected fixing, got %s", s[20])
		}
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		env := setupTest(t)

		resp, err := http.Get(env.srv.URL + "/api/issue/status")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", resp.StatusCode)
		}
	})
}

// claimIssue 调用 claim API 并返回 (statusCode, success, claimed_by)。
func claimIssue(t *testing.T, env *testEnv, repo string, number int, sessionID string) (int, bool, string) {
	t.Helper()

	body := fmt.Sprintf(
		`{"repo_full_name":"%s","issue_number":%d,"session_id":"%s"}`,
		repo, number, sessionID)
	resp := env.post(t, "/api/issue/claim", body)

	var result struct {
		Success   bool    `json:"success"`
		Error     string  `json:"error"`
		ClaimedBy *string `json:"claimed_by"`
		Status    string  `json:"status"`
	}
	readJSON(t, resp, &result)

	claimedBy := ""
	if result.ClaimedBy != nil {
		claimedBy = *result.ClaimedBy
	}
	return resp.StatusCode, result.Success, claimedBy
}

// checkStatuses 调用 check API 返回 issue_number → status 的映射。
func checkStatuses(t *testing.T, env *testEnv, repo string, numbers []int) map[int]string {
	t.Helper()

	nums := make([]string, len(numbers))
	for i, n := range numbers {
		nums[i] = fmt.Sprintf("%d", n)
	}

	body := `{"repo_full_name":"` + repo + `","issue_numbers":[` + strings.Join(nums, ",") + `]}`
	resp := env.post(t, "/api/issue/check", body)

	var result struct {
		Issues []struct {
			Number int    `json:"number"`
			Status string `json:"status"`
		} `json:"issues"`
	}
	readJSON(t, resp, &result)

	m := map[int]string{}
	for _, iss := range result.Issues {
		m[iss.Number] = iss.Status
	}
	return m
}

// --- D3: Issue Status Update → ready-for-pr tests ---
// 验收标准：
//   - fixing → ready-for-pr 状态更新成功
//   - 非 owner 无法更新
//   - 完整流程：claimed → fixing → ready-for-pr

// TestUpdateStatus_ReadyForPR 验证 fixing → ready-for-pr 状态流转。
func TestUpdateStatus_ReadyForPR(t *testing.T) {
	t.Run("fixing_to_ready_for_pr", func(t *testing.T) {
		// D3 验收：fixing 状态可更新为 ready-for-pr
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "fixing", "sess_abc", "2026-05-27T10:00:00Z")

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_abc","status":"ready-for-pr"}`)

		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if !result.Success {
			t.Fatal("expected success=true")
		}
		if result.PreviousStatus != "fixing" {
			t.Errorf("expected previous=fixing, got %s", result.PreviousStatus)
		}
		if result.NewStatus != "ready-for-pr" {
			t.Errorf("expected new=ready-for-pr, got %s", result.NewStatus)
		}

		statuses := checkStatuses(t, env, "test/repo", []int{10})
		if statuses[10] != "ready-for-pr" {
			t.Errorf("expected ready-for-pr, got %s", statuses[10])
		}
	})

	t.Run("non_owner_cannot_mark_ready", func(t *testing.T) {
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "fixing", "sess_abc", "2026-05-27T10:00:00Z")

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_other","status":"ready-for-pr"}`)

		var result struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		readJSON(t, resp, &result)

		if result.Success {
			t.Fatal("expected success=false for non-owner")
		}
		if result.Error != "not_owner" {
			t.Errorf("expected not_owner, got %s", result.Error)
		}
	})

	t.Run("full_claim_fix_done_flow", func(t *testing.T) {
		// 端到端：idle → claimed → fixing → ready-for-pr
		env := setupTest(t)

		// 1. idle
		env.post(t, "/api/issue/check",
			`{"repo_full_name":"test/repo","issue_numbers":[30]}`)
		s := checkStatuses(t, env, "test/repo", []int{30})
		if s[30] != "idle" {
			t.Fatalf("step 1: expected idle, got %s", s[30])
		}

		// 2. claim
		code, ok, _ := claimIssue(t, env, "test/repo", 30, "sess_done")
		if code != http.StatusOK || !ok {
			t.Fatalf("step 2: claim failed")
		}

		// 3. fixing
		env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":30,"session_id":"sess_done","status":"fixing"}`)
		s = checkStatuses(t, env, "test/repo", []int{30})
		if s[30] != "fixing" {
			t.Fatalf("step 3: expected fixing, got %s", s[30])
		}

		// 4. ready-for-pr
		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":30,"session_id":"sess_done","status":"ready-for-pr"}`)
		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)
		if !result.Success {
			t.Fatal("step 4: ready-for-pr update failed")
		}
		if result.PreviousStatus != "fixing" {
			t.Errorf("step 4: expected previous=fixing, got %s", result.PreviousStatus)
		}

		// 5. 验证最终状态
		s = checkStatuses(t, env, "test/repo", []int{30})
		if s[30] != "ready-for-pr" {
			t.Fatalf("step 5: expected ready-for-pr, got %s", s[30])
		}
	})
}

// --- D4: Issue Status Update → pr-created tests ---
// 验收标准：
//   - ready-for-pr → pr-created 状态更新成功
//   - 非 owner 无法更新
//   - 完整流程：claimed → fixing → ready-for-pr → pr-created

// TestUpdateStatus_PRCreated 验证 ready-for-pr → pr-created 状态流转。
func TestUpdateStatus_PRCreated(t *testing.T) {
	t.Run("ready_for_pr_to_pr_created", func(t *testing.T) {
		// D4 验收：ready-for-pr 状态可更新为 pr-created
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "ready-for-pr", "sess_abc", "2026-05-27T10:00:00Z")

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_abc","status":"pr-created"}`)

		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if !result.Success {
			t.Fatal("expected success=true")
		}
		if result.PreviousStatus != "ready-for-pr" {
			t.Errorf("expected previous=ready-for-pr, got %s", result.PreviousStatus)
		}
		if result.NewStatus != "pr-created" {
			t.Errorf("expected new=pr-created, got %s", result.NewStatus)
		}

		statuses := checkStatuses(t, env, "test/repo", []int{10})
		if statuses[10] != "pr-created" {
			t.Errorf("expected pr-created, got %s", statuses[10])
		}
	})

	t.Run("non_owner_cannot_create_pr_status", func(t *testing.T) {
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "ready-for-pr", "sess_abc", "2026-05-27T10:00:00Z")

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_other","status":"pr-created"}`)

		var result struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		readJSON(t, resp, &result)

		if result.Success {
			t.Fatal("expected success=false for non-owner")
		}
		if result.Error != "not_owner" {
			t.Errorf("expected not_owner, got %s", result.Error)
		}
	})

	t.Run("full_claim_fix_done_pr_flow", func(t *testing.T) {
		// 端到端：idle → claimed → fixing → ready-for-pr → pr-created
		env := setupTest(t)

		// 1. idle
		env.post(t, "/api/issue/check",
			`{"repo_full_name":"test/repo","issue_numbers":[40]}`)

		// 2. claim
		code, ok, _ := claimIssue(t, env, "test/repo", 40, "sess_pr")
		if code != http.StatusOK || !ok {
			t.Fatalf("step 2: claim failed")
		}

		// 3. fixing
		env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":40,"session_id":"sess_pr","status":"fixing"}`)

		// 4. ready-for-pr
		env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":40,"session_id":"sess_pr","status":"ready-for-pr"}`)

		// 5. pr-created
		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":40,"session_id":"sess_pr","status":"pr-created"}`)
		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)
		if !result.Success {
			t.Fatal("step 5: pr-created update failed")
		}
		if result.PreviousStatus != "ready-for-pr" {
			t.Errorf("step 5: expected previous=ready-for-pr, got %s", result.PreviousStatus)
		}

		// 6. 验证最终状态
		s := checkStatuses(t, env, "test/repo", []int{40})
		if s[40] != "pr-created" {
			t.Fatalf("step 6: expected pr-created, got %s", s[40])
		}
	})
}

// --- D5: Issue Status Update → testing tests ---
// 验收标准：
//   - pr-created → testing 状态更新成功
//   - 非 owner 无法更新
//   - 完整流程：claimed → fixing → ready-for-pr → pr-created → testing

// TestUpdateStatus_Testing 验证 pr-created → testing 状态流转。
func TestUpdateStatus_Testing(t *testing.T) {
	t.Run("pr_created_to_testing", func(t *testing.T) {
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "pr-created", "sess_abc", "2026-05-27T10:00:00Z")

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_abc","status":"testing"}`)

		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if !result.Success {
			t.Fatal("expected success=true")
		}
		if result.PreviousStatus != "pr-created" {
			t.Errorf("expected previous=pr-created, got %s", result.PreviousStatus)
		}
		if result.NewStatus != "testing" {
			t.Errorf("expected new=testing, got %s", result.NewStatus)
		}

		statuses := checkStatuses(t, env, "test/repo", []int{10})
		if statuses[10] != "testing" {
			t.Errorf("expected testing, got %s", statuses[10])
		}
	})

	t.Run("non_owner_cannot_mark_testing", func(t *testing.T) {
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "pr-created", "sess_abc", "2026-05-27T10:00:00Z")

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_other","status":"testing"}`)

		var result struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		readJSON(t, resp, &result)

		if result.Success {
			t.Fatal("expected success=false for non-owner")
		}
		if result.Error != "not_owner" {
			t.Errorf("expected not_owner, got %s", result.Error)
		}
	})

	t.Run("full_flow_to_testing", func(t *testing.T) {
		// 端到端：idle → claimed → fixing → ready-for-pr → pr-created → testing
		env := setupTest(t)

		env.post(t, "/api/issue/check",
			`{"repo_full_name":"test/repo","issue_numbers":[50]}`)

		code, ok, _ := claimIssue(t, env, "test/repo", 50, "sess_test")
		if code != http.StatusOK || !ok {
			t.Fatalf("claim failed")
		}

		env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":50,"session_id":"sess_test","status":"fixing"}`)
		env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":50,"session_id":"sess_test","status":"ready-for-pr"}`)
		env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":50,"session_id":"sess_test","status":"pr-created"}`)

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":50,"session_id":"sess_test","status":"testing"}`)
		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)
		if !result.Success {
			t.Fatal("testing update failed")
		}
		if result.PreviousStatus != "pr-created" {
			t.Errorf("expected previous=pr-created, got %s", result.PreviousStatus)
		}

		s := checkStatuses(t, env, "test/repo", []int{50})
		if s[50] != "testing" {
			t.Fatalf("expected testing, got %s", s[50])
		}
	})
}

// --- D6: Issue Status Update → reviewing/merged/rejected tests ---
// 验收标准：
//   - testing → reviewing 状态更新成功
//   - reviewing → merged 终态更新成功
//   - reviewing → rejected 状态更新成功
//   - merged 不被 session end 释放
//   - rejected 不被 session end 释放
//   - 完整流程：idle → ... → testing → reviewing → merged/rejected

// TestUpdateStatus_Review 验证 testing → reviewing → merged/rejected 状态流转。
func TestUpdateStatus_Review(t *testing.T) {
	t.Run("testing_to_reviewing", func(t *testing.T) {
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "testing", "sess_abc", "2026-05-27T10:00:00Z")

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_abc","status":"reviewing"}`)

		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)

		if !result.Success {
			t.Fatalf("expected success=true, got error: %s", result.NewStatus)
		}
		if result.PreviousStatus != "testing" {
			t.Errorf("expected previous=testing, got %s", result.PreviousStatus)
		}
		if result.NewStatus != "reviewing" {
			t.Errorf("expected new=reviewing, got %s", result.NewStatus)
		}
	})

	t.Run("reviewing_to_merged", func(t *testing.T) {
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "reviewing", "sess_abc", "2026-05-27T10:00:00Z")

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_abc","status":"merged"}`)

		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)

		if !result.Success {
			t.Fatal("expected success=true")
		}
		if result.PreviousStatus != "reviewing" {
			t.Errorf("expected previous=reviewing, got %s", result.PreviousStatus)
		}
		if result.NewStatus != "merged" {
			t.Errorf("expected new=merged, got %s", result.NewStatus)
		}
	})

	t.Run("reviewing_to_rejected", func(t *testing.T) {
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "reviewing", "sess_abc", "2026-05-27T10:00:00Z")

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_abc","status":"rejected"}`)

		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)

		if !result.Success {
			t.Fatal("expected success=true")
		}
		if result.PreviousStatus != "reviewing" {
			t.Errorf("expected previous=reviewing, got %s", result.PreviousStatus)
		}
		if result.NewStatus != "rejected" {
			t.Errorf("expected new=rejected, got %s", result.NewStatus)
		}
	})

	t.Run("non_owner_cannot_review", func(t *testing.T) {
		env := setupTest(t)
		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "testing", "sess_abc", "2026-05-27T10:00:00Z")

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_other","status":"reviewing"}`)

		var result struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		readJSON(t, resp, &result)

		if result.Success {
			t.Fatal("expected success=false for non-owner")
		}
		if result.Error != "not_owner" {
			t.Errorf("expected not_owner, got %s", result.Error)
		}
	})

	t.Run("full_merge_flow", func(t *testing.T) {
		// 端到端：idle → claimed → fixing → ready-for-pr → pr-created → testing → reviewing → merged
		env := setupTest(t)

		env.post(t, "/api/issue/check",
			`{"repo_full_name":"test/repo","issue_numbers":[60]}`)

		code, ok, _ := claimIssue(t, env, "test/repo", 60, "sess_merge")
		if code != http.StatusOK || !ok {
			t.Fatalf("claim failed")
		}

		for _, status := range []string{"fixing", "ready-for-pr", "pr-created", "testing", "reviewing"} {
			resp := env.post(t, "/api/issue/status",
				fmt.Sprintf(`{"repo_full_name":"test/repo","issue_number":60,"session_id":"sess_merge","status":"%s"}`, status))
			var r struct{ Success bool `json:"success"` }
			readJSON(t, resp, &r)
			if !r.Success {
				t.Fatalf("update to %s failed", status)
			}
		}

		// merged
		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":60,"session_id":"sess_merge","status":"merged"}`)
		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)
		if !result.Success {
			t.Fatal("merged update failed")
		}
		if result.PreviousStatus != "reviewing" {
			t.Errorf("expected previous=reviewing, got %s", result.PreviousStatus)
		}

		s := checkStatuses(t, env, "test/repo", []int{60})
		if s[60] != "merged" {
			t.Fatalf("expected merged, got %s", s[60])
		}
	})

	t.Run("full_reject_flow", func(t *testing.T) {
		// 端到端：idle → ... → testing → reviewing → rejected
		env := setupTest(t)

		env.post(t, "/api/issue/check",
			`{"repo_full_name":"test/repo","issue_numbers":[70]}`)

		claimIssue(t, env, "test/repo", 70, "sess_reject")
		for _, status := range []string{"fixing", "ready-for-pr", "pr-created", "testing", "reviewing"} {
			env.post(t, "/api/issue/status",
				fmt.Sprintf(`{"repo_full_name":"test/repo","issue_number":70,"session_id":"sess_reject","status":"%s"}`, status))
		}

		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":70,"session_id":"sess_reject","status":"rejected"}`)
		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)
		if !result.Success {
			t.Fatal("rejected update failed")
		}
		if result.PreviousStatus != "reviewing" {
			t.Errorf("expected previous=reviewing, got %s", result.PreviousStatus)
		}

		s := checkStatuses(t, env, "test/repo", []int{70})
		if s[70] != "rejected" {
			t.Fatalf("expected rejected, got %s", s[70])
		}
	})
}
