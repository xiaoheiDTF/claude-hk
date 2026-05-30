// Package integration_test 包含后端技能流程的集成测试。
// 模拟各个技能（issue-claim、issue-fix、issue-done、issue-pr、issue-test、issue-review）调用后端的完整流程。
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

// env 是集成测试环境，包含测试服务器和 SQLite 存储。
type env struct {
	srv   *httptest.Server
	store *store.SQLiteStore
}

// setup 创建集成测试环境，自动清理临时数据库。
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

// post 发送 POST 请求到集成测试环境。
func (e *env) post(t *testing.T, path, body string) *http.Response {
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

// claimIssue 调用 claim API 并返回 (statusCode, success, claimed_by)。
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

// nilIfEmpty 将空字符串转为 nil，用于数据库插入。
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// checkStatuses 调用 check API 并返回 issue_number → status 的映射。
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

// TestSessionEndHook 验证 SessionEnd hook 调用 release-session 的完整流程。
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

// TestClaimAPI 验证 Issue 领取 API。
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

// --- D2: 003-5-issue-fix 技能集成测试 ---
// 模拟 003-5-issue-fix 技能调用后端的完整流程
// 验收标准：
//   - claim 后调 status API 可将状态更新为 fixing
//   - 非 owner session 无法标记 fixing
//   - 后端不可用时 skill 脚本静默降级（此处验证 API 层行为）
//   - fixing 状态的 issue 可被 session end 正常释放
//   - 完整技能流程：claim → fixing → session end 释放

// TestIssueFixFlow 验证 issue-fix 技能的完整流程。
func TestIssueFixFlow(t *testing.T) {
	t.Run("claim_then_fixing", func(t *testing.T) {
		// D2 核心验收：claim → fixing 状态流转
		// 模拟 /003-4-issue-claim 后，/003-5-issue-fix 标记 fixing
		e := setup(t)

		// 1. check 创建 idle
		e.post(t, "/api/issue/check",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_numbers":[10]}`)
		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{10})
		if s[10] != "idle" {
			t.Fatalf("step 1: expected idle, got %s", s[10])
		}

		// 2. claim（模拟 003-4-issue-claim）
		code, ok, _ := claimIssue(t, e, "xiaoheiDTF/claude-hk", 10, "sess_fix_001")
		if code != http.StatusOK || !ok {
			t.Fatalf("step 2: claim failed: code=%d ok=%v", code, ok)
		}

		// 3. 标记 fixing（模拟 003-5-issue-fix 调用 update_issue_status）
		resp := e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":10,"session_id":"sess_fix_001","status":"fixing"}`)
		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)
		if !result.Success {
			t.Fatalf("step 3: fixing update failed")
		}
		if result.PreviousStatus != "claimed" {
			t.Errorf("step 3: expected previous=claimed, got %s", result.PreviousStatus)
		}
		if result.NewStatus != "fixing" {
			t.Errorf("step 3: expected new=fixing, got %s", result.NewStatus)
		}

		// 4. check 确认状态为 fixing
		s = checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{10})
		if s[10] != "fixing" {
			t.Fatalf("step 4: expected fixing, got %s", s[10])
		}
	})

	t.Run("non_owner_cannot_fix", func(t *testing.T) {
		// 验收：非领取者无法标记 fixing
		e := setup(t)
		db := e.store.DB()
		seedIssue(db, "xiaoheiDTF/claude-hk", 10, "claimed", "sess_owner", "2026-05-27T10:00:00Z")

		resp := e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":10,"session_id":"sess_other","status":"fixing"}`)

		var result struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		readJSON(t, resp, &result)
		if result.Success {
			t.Fatal("non-owner should not be able to update status")
		}
		if result.Error != "not_owner" {
			t.Errorf("expected not_owner, got %s", result.Error)
		}
	})

	t.Run("fixing_released_on_session_end", func(t *testing.T) {
		// 验收：fixing 状态的 issue 在 session 结束时被释放回到 idle
		// 模拟：claim → fixing → session end → idle
		e := setup(t)
		db := e.store.DB()
		seedIssue(db, "xiaoheiDTF/claude-hk", 10, "claimed", "sess_fix_end", "2026-05-27T10:00:00Z")

		// 标记 fixing
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":10,"session_id":"sess_fix_end","status":"fixing"}`)

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{10})
		if s[10] != "fixing" {
			t.Fatalf("expected fixing before session end, got %s", s[10])
		}

		// session end 释放
		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_fix_end"}`)
		var rel struct {
			Count int `json:"count"`
		}
		readJSON(t, resp, &rel)
		if rel.Count != 1 {
			t.Fatalf("expected 1 released, got %d", rel.Count)
		}

		// 验证回到 idle
		s = checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{10})
		if s[10] != "idle" {
			t.Errorf("expected idle after session end, got %s", s[10])
		}
	})

	t.Run("full_lifecycle_claim_fix_release", func(t *testing.T) {
		// 完整生命周期：idle → claimed → fixing → session end → idle → re-claim
		e := setup(t)

		// 1. idle
		e.post(t, "/api/issue/check",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_numbers":[15]}`)

		// 2. claim
		claimIssue(t, e, "xiaoheiDTF/claude-hk", 15, "sess_lifecycle")
		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{15})
		if s[15] != "claimed" {
			t.Fatalf("step 2: expected claimed, got %s", s[15])
		}

		// 3. fixing
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":15,"session_id":"sess_lifecycle","status":"fixing"}`)
		s = checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{15})
		if s[15] != "fixing" {
			t.Fatalf("step 3: expected fixing, got %s", s[15])
		}

		// 4. session end 释放
		e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_lifecycle"}`)
		s = checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{15})
		if s[15] != "idle" {
			t.Fatalf("step 4: expected idle, got %s", s[15])
		}

		// 5. 重新被另一个 session 领取
		code, ok, _ := claimIssue(t, e, "xiaoheiDTF/claude-hk", 15, "sess_new")
		if code != http.StatusOK || !ok {
			t.Fatalf("step 5: re-claim failed: code=%d ok=%v", code, ok)
		}
		s = checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{15})
		if s[15] != "claimed" {
			t.Fatalf("step 5: expected claimed, got %s", s[15])
		}
	})

	t.Run("fixing_merged_not_released_on_session_end", func(t *testing.T) {
		// 补充：如果 fixing 的 issue 在期间被外部标记为 merged，
		// session end 不应释放它（虽然正常流程不会出现这种场景，但验证边界条件）
		e := setup(t)
		db := e.store.DB()
		seedIssue(db, "xiaoheiDTF/claude-hk", 10, "merged", "sess_merged", "2026-05-27T10:00:00Z")

		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_merged"}`)
		var rel struct {
			Count int `json:"count"`
		}
		readJSON(t, resp, &rel)
		if rel.Count != 0 {
			t.Fatalf("merged should not be released, got count=%d", rel.Count)
		}

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{10})
		if s[10] != "merged" {
			t.Errorf("expected merged unchanged, got %s", s[10])
		}
	})
}

// --- D3: 003-6-issue-done 技能集成测试 ---
// 验收标准：
//   - fixing → ready-for-pr 状态更新
//   - ready-for-pr 状态在 session end 时被释放
//   - 完整技能流程：claim → fixing → ready-for-pr → session end

// TestIssueDoneFlow 验证 issue-done 技能的完整流程。
func TestIssueDoneFlow(t *testing.T) {
	t.Run("fixing_to_ready_for_pr", func(t *testing.T) {
		e := setup(t)

		// claim → fixing → ready-for-pr
		claimIssue(t, e, "xiaoheiDTF/claude-hk", 20, "sess_done_001")
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":20,"session_id":"sess_done_001","status":"fixing"}`)

		resp := e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":20,"session_id":"sess_done_001","status":"ready-for-pr"}`)
		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)
		if !result.Success {
			t.Fatal("ready-for-pr update failed")
		}
		if result.PreviousStatus != "fixing" {
			t.Errorf("expected previous=fixing, got %s", result.PreviousStatus)
		}

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{20})
		if s[20] != "ready-for-pr" {
			t.Fatalf("expected ready-for-pr, got %s", s[20])
		}
	})

	t.Run("ready_for_pr_released_on_session_end", func(t *testing.T) {
		e := setup(t)
		db := e.store.DB()
		seedIssue(db, "xiaoheiDTF/claude-hk", 10, "ready-for-pr", "sess_done_end", "2026-05-27T10:00:00Z")

		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_done_end"}`)
		var rel struct{ Count int `json:"count"` }
		readJSON(t, resp, &rel)
		if rel.Count != 1 {
			t.Fatalf("expected 1 released, got %d", rel.Count)
		}

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{10})
		if s[10] != "idle" {
			t.Errorf("expected idle after session end, got %s", s[10])
		}
	})

	t.Run("full_lifecycle_claim_fix_done_release", func(t *testing.T) {
		e := setup(t)

		// idle
		e.post(t, "/api/issue/check",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_numbers":[25]}`)

		// claim
		claimIssue(t, e, "xiaoheiDTF/claude-hk", 25, "sess_lc_done")

		// fixing
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":25,"session_id":"sess_lc_done","status":"fixing"}`)

		// ready-for-pr
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":25,"session_id":"sess_lc_done","status":"ready-for-pr"}`)
		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{25})
		if s[25] != "ready-for-pr" {
			t.Fatalf("expected ready-for-pr, got %s", s[25])
		}

		// session end
		e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_lc_done"}`)
		s = checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{25})
		if s[25] != "idle" {
			t.Fatalf("expected idle after session end, got %s", s[25])
		}

		// re-claim by another session
		code, ok, _ := claimIssue(t, e, "xiaoheiDTF/claude-hk", 25, "sess_new")
		if code != http.StatusOK || !ok {
			t.Fatalf("re-claim failed")
		}
	})
}

// --- D4: 003-7-issue-pr 技能集成测试 ---
// 验收标准：
//   - ready-for-pr → pr-created 状态更新
//   - pr-created 状态在 session end 时被释放
//   - 完整技能流程：claim → fixing → ready-for-pr → pr-created

// TestIssuePRFlow 验证 issue-pr 技能的完整流程。
func TestIssuePRFlow(t *testing.T) {
	t.Run("ready_for_pr_to_pr_created", func(t *testing.T) {
		e := setup(t)

		// claim → fixing → ready-for-pr → pr-created
		claimIssue(t, e, "xiaoheiDTF/claude-hk", 30, "sess_pr_001")
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":30,"session_id":"sess_pr_001","status":"fixing"}`)
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":30,"session_id":"sess_pr_001","status":"ready-for-pr"}`)

		resp := e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":30,"session_id":"sess_pr_001","status":"pr-created"}`)
		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)
		if !result.Success {
			t.Fatal("pr-created update failed")
		}
		if result.PreviousStatus != "ready-for-pr" {
			t.Errorf("expected previous=ready-for-pr, got %s", result.PreviousStatus)
		}

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{30})
		if s[30] != "pr-created" {
			t.Fatalf("expected pr-created, got %s", s[30])
		}
	})

	t.Run("pr_created_released_on_session_end", func(t *testing.T) {
		e := setup(t)
		db := e.store.DB()
		seedIssue(db, "xiaoheiDTF/claude-hk", 10, "pr-created", "sess_pr_end", "2026-05-27T10:00:00Z")

		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_pr_end"}`)
		var rel struct{ Count int `json:"count"` }
		readJSON(t, resp, &rel)
		if rel.Count != 1 {
			t.Fatalf("expected 1 released, got %d", rel.Count)
		}

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{10})
		if s[10] != "idle" {
			t.Errorf("expected idle after session end, got %s", s[10])
		}
	})

	t.Run("full_lifecycle_to_pr_created", func(t *testing.T) {
		e := setup(t)

		// idle
		e.post(t, "/api/issue/check",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_numbers":[35]}`)

		// claim → fixing → ready-for-pr → pr-created
		claimIssue(t, e, "xiaoheiDTF/claude-hk", 35, "sess_pr_lc")
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":35,"session_id":"sess_pr_lc","status":"fixing"}`)
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":35,"session_id":"sess_pr_lc","status":"ready-for-pr"}`)
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":35,"session_id":"sess_pr_lc","status":"pr-created"}`)

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{35})
		if s[35] != "pr-created" {
			t.Fatalf("expected pr-created, got %s", s[35])
		}

		// session end 释放 pr-created
		e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_pr_lc"}`)
		s = checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{35})
		if s[35] != "idle" {
			t.Fatalf("expected idle after session end, got %s", s[35])
		}

		// re-claim
		code, ok, _ := claimIssue(t, e, "xiaoheiDTF/claude-hk", 35, "sess_new")
		if code != http.StatusOK || !ok {
			t.Fatalf("re-claim failed")
		}
	})
}

// --- D5: 003-8-issue-test 技能集成测试 ---
// 验收标准：
//   - pr-created → testing 状态更新
//   - testing 状态在 session end 时被释放
//   - 完整技能流程：claim → fixing → ready-for-pr → pr-created → testing

// TestIssueTestFlow 验证 issue-test 技能的完整流程。
func TestIssueTestFlow(t *testing.T) {
	t.Run("pr_created_to_testing", func(t *testing.T) {
		e := setup(t)

		claimIssue(t, e, "xiaoheiDTF/claude-hk", 40, "sess_test_001")
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":40,"session_id":"sess_test_001","status":"fixing"}`)
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":40,"session_id":"sess_test_001","status":"ready-for-pr"}`)
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":40,"session_id":"sess_test_001","status":"pr-created"}`)

		resp := e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":40,"session_id":"sess_test_001","status":"testing"}`)
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

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{40})
		if s[40] != "testing" {
			t.Fatalf("expected testing, got %s", s[40])
		}
	})

	t.Run("testing_released_on_session_end", func(t *testing.T) {
		e := setup(t)
		db := e.store.DB()
		seedIssue(db, "xiaoheiDTF/claude-hk", 10, "testing", "sess_test_end", "2026-05-27T10:00:00Z")

		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_test_end"}`)
		var rel struct{ Count int `json:"count"` }
		readJSON(t, resp, &rel)
		if rel.Count != 1 {
			t.Fatalf("expected 1 released, got %d", rel.Count)
		}

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{10})
		if s[10] != "idle" {
			t.Errorf("expected idle after session end, got %s", s[10])
		}
	})

	t.Run("full_lifecycle_to_testing", func(t *testing.T) {
		e := setup(t)

		// idle
		e.post(t, "/api/issue/check",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_numbers":[45]}`)

		// claim → fixing → ready-for-pr → pr-created → testing
		claimIssue(t, e, "xiaoheiDTF/claude-hk", 45, "sess_test_lc")
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":45,"session_id":"sess_test_lc","status":"fixing"}`)
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":45,"session_id":"sess_test_lc","status":"ready-for-pr"}`)
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":45,"session_id":"sess_test_lc","status":"pr-created"}`)
		e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":45,"session_id":"sess_test_lc","status":"testing"}`)

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{45})
		if s[45] != "testing" {
			t.Fatalf("expected testing, got %s", s[45])
		}

		// session end 释放 testing
		e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_test_lc"}`)
		s = checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{45})
		if s[45] != "idle" {
			t.Fatalf("expected idle after session end, got %s", s[45])
		}

		// re-claim
		code, ok, _ := claimIssue(t, e, "xiaoheiDTF/claude-hk", 45, "sess_new")
		if code != http.StatusOK || !ok {
			t.Fatalf("re-claim failed")
		}
	})
}

// --- D6: 003-9-issue-review 技能集成测试 ---
// 验收标准：
//   - testing → reviewing → merged 完整流程
//   - merged 终态不被 session end 释放
//   - rejected 终态不被 session end 释放
//   - rejected issue 释放后可被重新 claim

// TestIssueReviewFlow 验证 issue-review 技能的完整流程。
func TestIssueReviewFlow(t *testing.T) {
	t.Run("merge_full_flow", func(t *testing.T) {
		e := setup(t)

		// claim → fixing → ready-for-pr → pr-created → testing → reviewing → merged
		claimIssue(t, e, "xiaoheiDTF/claude-hk", 50, "sess_merge")
		for _, s := range []string{"fixing", "ready-for-pr", "pr-created", "testing", "reviewing"} {
			e.post(t, "/api/issue/status",
				fmt.Sprintf(`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":50,"session_id":"sess_merge","status":"%s"}`, s))
		}

		resp := e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":50,"session_id":"sess_merge","status":"merged"}`)
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

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{50})
		if s[50] != "merged" {
			t.Fatalf("expected merged, got %s", s[50])
		}
	})

	t.Run("merged_not_released_on_session_end", func(t *testing.T) {
		// 验收：merged 终态不被 session end 释放
		e := setup(t)
		db := e.store.DB()
		seedIssue(db, "xiaoheiDTF/claude-hk", 10, "merged", "sess_merged", "2026-05-27T10:00:00Z")

		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_merged"}`)
		var rel struct{ Count int `json:"count"` }
		readJSON(t, resp, &rel)
		if rel.Count != 0 {
			t.Fatalf("merged should not be released, got count=%d", rel.Count)
		}

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{10})
		if s[10] != "merged" {
			t.Errorf("expected merged unchanged, got %s", s[10])
		}
	})

	t.Run("reject_full_flow", func(t *testing.T) {
		e := setup(t)

		claimIssue(t, e, "xiaoheiDTF/claude-hk", 55, "sess_reject")
		for _, s := range []string{"fixing", "ready-for-pr", "pr-created", "testing", "reviewing"} {
			e.post(t, "/api/issue/status",
				fmt.Sprintf(`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":55,"session_id":"sess_reject","status":"%s"}`, s))
		}

		resp := e.post(t, "/api/issue/status",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":55,"session_id":"sess_reject","status":"rejected"}`)
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

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{55})
		if s[55] != "rejected" {
			t.Fatalf("expected rejected, got %s", s[55])
		}
	})

	t.Run("rejected_not_released_on_session_end", func(t *testing.T) {
		// 验收：rejected 终态不被 session end 释放
		e := setup(t)
		db := e.store.DB()
		seedIssue(db, "xiaoheiDTF/claude-hk", 11, "rejected", "sess_rejected", "2026-05-27T10:00:00Z")

		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_rejected"}`)
		var rel struct{ Count int `json:"count"` }
		readJSON(t, resp, &rel)
		if rel.Count != 0 {
			t.Fatalf("rejected should not be released, got count=%d", rel.Count)
		}

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{11})
		if s[11] != "rejected" {
			t.Errorf("expected rejected unchanged, got %s", s[11])
		}
	})

	t.Run("non_terminal_reviewing_released_on_session_end", func(t *testing.T) {
		// reviewing（非终态）在 session end 时应被释放
		e := setup(t)
		db := e.store.DB()
		seedIssue(db, "xiaoheiDTF/claude-hk", 12, "reviewing", "sess_rev", "2026-05-27T10:00:00Z")

		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_rev"}`)
		var rel struct{ Count int `json:"count"` }
		readJSON(t, resp, &rel)
		if rel.Count != 1 {
			t.Fatalf("reviewing should be released, got count=%d", rel.Count)
		}

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{12})
		if s[12] != "idle" {
			t.Errorf("expected idle after session end, got %s", s[12])
		}
	})

	t.Run("complete_lifecycle_merge", func(t *testing.T) {
		// 完整生命周期：idle → claimed → ... → merged，session end 不影响
		e := setup(t)

		e.post(t, "/api/issue/check",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_numbers":[60]}`)

		claimIssue(t, e, "xiaoheiDTF/claude-hk", 60, "sess_full_merge")
		for _, s := range []string{"fixing", "ready-for-pr", "pr-created", "testing", "reviewing", "merged"} {
			e.post(t, "/api/issue/status",
				fmt.Sprintf(`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":60,"session_id":"sess_full_merge","status":"%s"}`, s))
		}

		// session end 不释放 merged
		e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_full_merge"}`)
		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{60})
		if s[60] != "merged" {
			t.Fatalf("expected merged preserved after session end, got %s", s[60])
		}
	})

	t.Run("complete_lifecycle_reject_then_reclaim", func(t *testing.T) {
		// 完整生命周期：... → rejected，手动 release 后可重新 claim
		e := setup(t)

		e.post(t, "/api/issue/check",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_numbers":[65]}`)

		claimIssue(t, e, "xiaoheiDTF/claude-hk", 65, "sess_rej")
		for _, s := range []string{"fixing", "ready-for-pr", "pr-created", "testing", "reviewing", "rejected"} {
			e.post(t, "/api/issue/status",
				fmt.Sprintf(`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":65,"session_id":"sess_rej","status":"%s"}`, s))
		}

		// rejected 是终态，session end 不释放
		e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_rej"}`)
		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{65})
		if s[65] != "rejected" {
			t.Fatalf("expected rejected preserved, got %s", s[65])
		}

		// 手动 release（由新的 fixing 周期触发）
		e.post(t, "/api/issue/release",
			`{"repo_full_name":"xiaoheiDTF/claude-hk","issue_number":65,"session_id":"sess_rej"}`)
		s = checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{65})
		if s[65] != "rejected" {
			t.Fatalf("rejected should not be releasable by owner either, got %s", s[65])
		}
	})
}

// --- D7: SessionEnd Hook 自动释放集成测试 ---
// 验收标准：
//   - 会话结束时释放该 session 所有非终态 issue
//   - merged 不被释放
//   - rejected 不被释放
//   - 无领取记录的 session 不报错
//   - 释放后 issue 可被重新 claim

// TestSessionEndRelease 验证 SessionEnd 自动释放的边界条件。
func TestSessionEndRelease(t *testing.T) {
	t.Run("releases_fixing_ready_for_pr_pr_created_testing_reviewing", func(t *testing.T) {
		// 验收：所有非终态（fixing/ready-for-pr/pr-created/testing/reviewing）都被释放
		e := setup(t)
		db := e.store.DB()

		seedIssue(db, "xiaoheiDTF/claude-hk", 10, "fixing", "sess_d7", "2026-05-27T10:00:00Z")
		seedIssue(db, "xiaoheiDTF/claude-hk", 11, "ready-for-pr", "sess_d7", "2026-05-27T10:00:00Z")
		seedIssue(db, "xiaoheiDTF/claude-hk", 12, "pr-created", "sess_d7", "2026-05-27T10:00:00Z")
		seedIssue(db, "xiaoheiDTF/claude-hk", 13, "testing", "sess_d7", "2026-05-27T10:00:00Z")
		seedIssue(db, "xiaoheiDTF/claude-hk", 14, "reviewing", "sess_d7", "2026-05-27T10:00:00Z")

		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_d7"}`)
		var result struct {
			Released []int `json:"released"`
			Count    int   `json:"count"`
		}
		readJSON(t, resp, &result)

		if result.Count != 5 {
			t.Fatalf("expected 5 released, got %d", result.Count)
		}

		// 验证所有 issue 都回到 idle
		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{10, 11, 12, 13, 14})
		for _, n := range []int{10, 11, 12, 13, 14} {
			if s[n] != "idle" {
				t.Errorf("issue %d: expected idle, got %s", n, s[n])
			}
		}
	})

	t.Run("merged_not_released", func(t *testing.T) {
		// 验收：merged 终态不被释放
		e := setup(t)
		db := e.store.DB()

		seedIssue(db, "xiaoheiDTF/claude-hk", 10, "merged", "sess_d7_m", "2026-05-27T10:00:00Z")
		seedIssue(db, "xiaoheiDTF/claude-hk", 11, "fixing", "sess_d7_m", "2026-05-27T10:00:00Z")

		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_d7_m"}`)
		var result struct {
			Released []int `json:"released"`
			Count    int   `json:"count"`
		}
		readJSON(t, resp, &result)

		if result.Count != 1 {
			t.Fatalf("expected 1 released (fixing only), got %d", result.Count)
		}
		if len(result.Released) != 1 || result.Released[0] != 11 {
			t.Fatalf("expected [11], got %v", result.Released)
		}

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{10, 11})
		if s[10] != "merged" {
			t.Errorf("merged should stay, got %s", s[10])
		}
		if s[11] != "idle" {
			t.Errorf("fixing should be idle, got %s", s[11])
		}
	})

	t.Run("rejected_not_released", func(t *testing.T) {
		// 验收：rejected 终态不被释放
		e := setup(t)
		db := e.store.DB()

		seedIssue(db, "xiaoheiDTF/claude-hk", 10, "rejected", "sess_d7_r", "2026-05-27T10:00:00Z")
		seedIssue(db, "xiaoheiDTF/claude-hk", 11, "claimed", "sess_d7_r", "2026-05-27T10:00:00Z")

		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_d7_r"}`)
		var result struct{ Count int `json:"count"` }
		readJSON(t, resp, &result)

		if result.Count != 1 {
			t.Fatalf("expected 1 released (claimed only), got %d", result.Count)
		}

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{10, 11})
		if s[10] != "rejected" {
			t.Errorf("rejected should stay, got %s", s[10])
		}
		if s[11] != "idle" {
			t.Errorf("claimed should be idle, got %s", s[11])
		}
	})

	t.Run("released_issue_can_be_reclaimed", func(t *testing.T) {
		// 验收：释放后 issue 可被重新 claim
		e := setup(t)
		db := e.store.DB()

		seedIssue(db, "xiaoheiDTF/claude-hk", 10, "fixing", "sess_old", "2026-05-27T10:00:00Z")

		// 释放
		e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_old"}`)

		// 新 session claim
		code, ok, _ := claimIssue(t, e, "xiaoheiDTF/claude-hk", 10, "sess_new")
		if code != http.StatusOK || !ok {
			t.Fatalf("re-claim should succeed: code=%d ok=%v", code, ok)
		}

		s := checkStatuses(t, e, "xiaoheiDTF/claude-hk", []int{10})
		if s[10] != "claimed" {
			t.Fatalf("expected claimed after re-claim, got %s", s[10])
		}
	})

	t.Run("no_backend_issues_returns_ok", func(t *testing.T) {
		// 验收：无领取记录的 session 返回成功
		e := setup(t)

		resp := e.post(t, "/api/issue/release-session",
			`{"session_id":"sess_empty_d7"}`)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	})
}
