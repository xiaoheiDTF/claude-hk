package integration_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

type env struct {
	srv   *httptest.Server
	store *store.SQLiteStore
}

func setup(t *testing.T) *env {
	t.Helper()

	f, err := os.CreateTemp("", "test-sessionend-*.db")
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

	return &env{srv: srv, store: s}
}

func (e *env) post(t *testing.T, path, body string) *http.Response {
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

// claimIssue calls the claim API and returns (statusCode, success, claimed_by).
func claimIssue(t *testing.T, e *env, repo string, number int, sessionID string) (int, bool, string) {
	t.Helper()

	body := fmt.Sprintf(
		`{"repo_full_name":"%s","issue_number":%d,"session_id":"%s"}`,
		repo, number, sessionID)
	resp := e.post(t, "/api/issue/claim", body)

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

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// checkStatuses queries the check API and returns a map of issue_number → status.
func checkStatuses(t *testing.T, e *env, repo string, numbers []int) map[int]string {
	t.Helper()

	nums := make([]string, len(numbers))
	for i, n := range numbers {
		nums[i] = fmt.Sprintf("%d", n)
	}

	body := `{"repo_full_name":"` + repo + `","issue_numbers":[` + strings.Join(nums, ",") + `]}`
	resp := e.post(t, "/api/issue/check", body)

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

// --- B6-3 SessionEnd Hook 集成测试 ---
// 模拟 SessionEnd hook 调用 release-session API 的完整流程
// 验收标准：
//   - session 正常结束时，领取的 issue 被自动释放
//   - 已合并/打回的 issue 不被释放
//   - 无领取记录的 session 不报错

func TestSessionEndHook(t *testing.T) {
	t.Run("session_end_releases_claimed_issues", func(t *testing.T) {
		// 验收：session 正常结束时，领取的 issue 被自动释放
		// 模拟场景：sess_test 领取了 #10(claimed) 和 #11(fixing)
		// SessionEnd hook 调用 release-session → #10/#11 回到 idle
		e := setup(t)
		db := e.store.DB()

		seedIssue(db, "test/repo", 10, "claimed", "sess_test", "2026-05-26T10:00:00Z")
		seedIssue(db, "test/repo", 11, "fixing", "sess_test", "2026-05-26T10:00:00Z")

		// 模拟 hook 调用 release-session
		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_test"}`)

		var result struct {
			Released []int `json:"released"`
			Count    int   `json:"count"`
		}
		readJSON(t, resp, &result)

		if result.Count != 2 {
			t.Fatalf("expected 2 released, got %d", result.Count)
		}

		// 验证两个 issue 都回到了 idle
		statuses := checkStatuses(t, e, "test/repo", []int{10, 11})
		if statuses[10] != "idle" {
			t.Errorf("expected #10 idle, got %s", statuses[10])
		}
		if statuses[11] != "idle" {
			t.Errorf("expected #11 idle, got %s", statuses[11])
		}
	})

	t.Run("merged_and_rejected_not_released", func(t *testing.T) {
		// 验收：已合并/打回的 issue 不被释放
		// 模拟场景：sess_test 有 #10(claimed) #11(merged) #12(rejected)
		// SessionEnd → 只有 #10 被释放，#11/#12 不变
		e := setup(t)
		db := e.store.DB()

		seedIssue(db, "test/repo", 10, "claimed", "sess_test", "2026-05-26T10:00:00Z")
		seedIssue(db, "test/repo", 11, "merged", "sess_test", "2026-05-26T10:00:00Z")
		seedIssue(db, "test/repo", 12, "rejected", "sess_test", "2026-05-26T10:00:00Z")

		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_test"}`)

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

		// 验证终态不变
		statuses := checkStatuses(t, e, "test/repo", []int{10, 11, 12})
		if statuses[10] != "idle" {
			t.Errorf("expected #10 idle after release, got %s", statuses[10])
		}
		if statuses[11] != "merged" {
			t.Errorf("expected #11 still merged, got %s", statuses[11])
		}
		if statuses[12] != "rejected" {
			t.Errorf("expected #12 still rejected, got %s", statuses[12])
		}
	})

	t.Run("no_claims_returns_empty_no_error", func(t *testing.T) {
		// 验收：无领取记录的 session 不报错
		// 模拟场景：sess_empty 没有领取任何 issue
		// SessionEnd → released=[], count=0，无报错
		e := setup(t)

		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_empty"}`)

		var result struct {
			Released []int `json:"released"`
			Count    int   `json:"count"`
		}
		readJSON(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if result.Count != 0 {
			t.Errorf("expected 0, got %d", result.Count)
		}
		if len(result.Released) != 0 {
			t.Errorf("expected empty, got %v", result.Released)
		}
	})

	t.Run("only_releases_target_session", func(t *testing.T) {
		// 补充：只释放目标 session 的 issue，其他 session 的 issue 不受影响
		// 模拟场景：sess_a 领取 #10, sess_b 领取 #11, #12(merged)属于 sess_a
		// SessionEnd(sess_a) → 释放 #10，#11(属于sess_b)和 #12(merged)不变
		e := setup(t)
		db := e.store.DB()

		seedIssue(db, "test/repo", 10, "claimed", "sess_a", "2026-05-26T10:00:00Z")
		seedIssue(db, "test/repo", 11, "claimed", "sess_b", "2026-05-26T10:00:00Z")
		seedIssue(db, "test/repo", 12, "merged", "sess_a", "2026-05-26T10:00:00Z")

		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_a"}`)

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

		statuses := checkStatuses(t, e, "test/repo", []int{10, 11, 12})
		if statuses[10] != "idle" {
			t.Errorf("expected #10 idle, got %s", statuses[10])
		}
		if statuses[11] != "claimed" {
			t.Errorf("expected #11 still claimed (sess_b), got %s", statuses[11])
		}
		if statuses[12] != "merged" {
			t.Errorf("expected #12 still merged, got %s", statuses[12])
		}
	})

	t.Run("full_flow_claim_then_session_end", func(t *testing.T) {
		// 端到端流程：先 check（自动创建 idle）→ 模拟 claim → SessionEnd 释放
		// 模拟完整生命周期：idle → claimed → SessionEnd → idle
		e := setup(t)

		// 1. check 创建 idle 记录
		e.post(t, "/api/issue/check",
			`{"repo_full_name":"test/repo","issue_numbers":[20]}`)

		statuses := checkStatuses(t, e, "test/repo", []int{20})
		if statuses[20] != "idle" {
			t.Fatalf("step 1: expected idle, got %s", statuses[20])
		}

		// 2. 模拟 claim（直接 SQL，claim API 后续实现）
		e.store.DB().Exec(
			`UPDATE issue_claims SET status='claimed', session_id='sess_e2e', claimed_at=CURRENT_TIMESTAMP
			 WHERE repo_full_name='test/repo' AND issue_number=20`)

		statuses = checkStatuses(t, e, "test/repo", []int{20})
		if statuses[20] != "claimed" {
			t.Fatalf("step 2: expected claimed, got %s", statuses[20])
		}

		// 3. SessionEnd 触发释放
		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_e2e"}`)

		var result struct {
			Count int `json:"count"`
		}
		readJSON(t, resp, &result)
		if result.Count != 1 {
			t.Fatalf("step 3: expected 1 released, got %d", result.Count)
		}

		// 4. 验证回到 idle
		statuses = checkStatuses(t, e, "test/repo", []int{20})
		if statuses[20] != "idle" {
			t.Fatalf("step 4: expected idle after release, got %s", statuses[20])
		}
	})
}

// --- B3: Claim API 集成测试 ---
// 验收标准：
//   - 空闲 issue 可被成功领取
//   - 已被其他 session 领取的 issue 返回 already_claimed
//   - 同一 session 重复领取同一 issue 返回成功（幂等）
//   - 已合并/打回的 issue 不可领取
//   - 首次出现的 issue 自动创建记录后领取
//   - 并发领取时只有一个成功（原子性）

func TestClaimAPI(t *testing.T) {
	t.Run("claim_idle_issue_success", func(t *testing.T) {
		// 验收：空闲 issue 可被成功领取
		e := setup(t)
		db := e.store.DB()

		seedIssue(db, "test/repo", 10, "idle", "", "")

		code, success, _ := claimIssue(t, e, "test/repo", 10, "sess_a")
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		if !success {
			t.Fatal("expected success=true")
		}

		statuses := checkStatuses(t, e, "test/repo", []int{10})
		if statuses[10] != "claimed" {
			t.Errorf("expected claimed, got %s", statuses[10])
		}
	})

	t.Run("claim_already_claimed_by_other", func(t *testing.T) {
		// 验收：已被其他 session 领取的 issue 返回 already_claimed
		e := setup(t)
		db := e.store.DB()

		seedIssue(db, "test/repo", 10, "claimed", "sess_a", "2026-05-26T10:00:00Z")

		code, success, claimedBy := claimIssue(t, e, "test/repo", 10, "sess_b")
		if code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", code)
		}
		if success {
			t.Fatal("expected success=false")
		}
		if claimedBy != "sess_a" {
			t.Errorf("expected claimed_by=sess_a, got %s", claimedBy)
		}
	})

	t.Run("claim_idempotent_same_session", func(t *testing.T) {
		// 验收：同一 session 重复领取同一 issue 返回成功（幂等）
		e := setup(t)
		db := e.store.DB()

		seedIssue(db, "test/repo", 10, "idle", "", "")

		code1, ok1, _ := claimIssue(t, e, "test/repo", 10, "sess_a")
		if code1 != http.StatusOK || !ok1 {
			t.Fatalf("first claim should succeed: code=%d ok=%v", code1, ok1)
		}

		code2, ok2, _ := claimIssue(t, e, "test/repo", 10, "sess_a")
		if code2 != http.StatusOK || !ok2 {
			t.Fatalf("idempotent claim should succeed: code=%d ok=%v", code2, ok2)
		}
	})

	t.Run("claim_merged_rejected_blocked", func(t *testing.T) {
		// 验收：已合并/打回的 issue 不可领取
		e := setup(t)
		db := e.store.DB()

		seedIssue(db, "test/repo", 10, "merged", "sess_old", "2026-05-26T10:00:00Z")
		seedIssue(db, "test/repo", 11, "rejected", "sess_old", "2026-05-26T10:00:00Z")

		code10, ok10, _ := claimIssue(t, e, "test/repo", 10, "sess_new")
		if code10 != http.StatusConflict || ok10 {
			t.Errorf("merged issue should not be claimable: code=%d ok=%v", code10, ok10)
		}

		code11, ok11, _ := claimIssue(t, e, "test/repo", 11, "sess_new")
		if code11 != http.StatusConflict || ok11 {
			t.Errorf("rejected issue should not be claimable: code=%d ok=%v", code11, ok11)
		}
	})

	t.Run("claim_new_issue_auto_create", func(t *testing.T) {
		// 验收：首次出现的 issue 自动创建记录后领取
		e := setup(t)

		code, success, _ := claimIssue(t, e, "test/repo", 99, "sess_new")
		if code != http.StatusOK || !success {
			t.Fatalf("first-time claim should succeed: code=%d ok=%v", code, success)
		}

		statuses := checkStatuses(t, e, "test/repo", []int{99})
		if statuses[99] != "claimed" {
			t.Errorf("expected claimed, got %s", statuses[99])
		}
	})

	t.Run("claim_missing_params_returns_error", func(t *testing.T) {
		// 补充：缺少必填参数时返回 400
		e := setup(t)

		resp := e.post(t, "/api/issue/claim", `{"repo_full_name":"test/repo","issue_number":10}`)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("claim_after_release_succeeds", func(t *testing.T) {
		// 补充：issue 被释放后可被重新领取
		e := setup(t)
		db := e.store.DB()

		seedIssue(db, "test/repo", 10, "claimed", "sess_a", "2026-05-26T10:00:00Z")

		// 释放
		e.post(t, "/api/issue/release",
			`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_a"}`)

		// 新 session 领取
		code, success, _ := claimIssue(t, e, "test/repo", 10, "sess_b")
		if code != http.StatusOK || !success {
			t.Fatalf("claim after release should succeed: code=%d ok=%v", code, success)
		}
	})
}
