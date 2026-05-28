package backend_test

import (
	"database/sql"
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

type testEnv struct {
	srv   *httptest.Server
	store *store.SQLiteStore
}

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
	router := api.NewRouter(api.Handlers{
		Issue: api.NewIssueHandler(issueSvc),
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &testEnv{srv: srv, store: s}
}

func (e *testEnv) post(t *testing.T, path, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(e.srv.URL+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

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

func seedIssue(db *sql.DB, repo string, number int, status, sessionID, claimedAt string) {
	db.Exec(
		`INSERT INTO issue_claims (repo_full_name, issue_number, status, session_id, claimed_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		repo, number, status,
		nilIfEmpty(sessionID), nilIfEmpty(claimedAt),
	)
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// --- tests ---

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

	// #9: new → idle
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

	// #12: new → idle
	if byNumber[12].Status != "idle" {
		t.Errorf("issue 12: expected idle, got %s", byNumber[12].Status)
	}
}

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

func TestCheckIssues_IdempotentOnRepeatedCalls(t *testing.T) {
	env := setupTest(t)

	// First call creates idle records
	env.post(t, "/api/issue/check",
		`{"repo_full_name":"test/repo","issue_numbers":[1,2]}`)

	// Second call should return the same idle records
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
