// Package backend_test 包含后端 API 的验收测试，覆盖 Issue 状态流转相关接口。
package backend_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// --- B4: 状态流转 API ---
//
// 验收标准覆盖：
//   - 合法状态可成功更新
//   - 非领取者 session 无法更新状态（返回 not_owner）
//   - updated_at 自动更新
//   - 响应中包含 previous_status 字段
//   - 不存在的 issue 返回错误

// TestB4_UpdatedAtAutoRefresh 验收：updated_at 在状态更新后自动变化。
// 验证更新 issue 状态后，数据库中的 updated_at 字段会被自动刷新为当前时间。
func TestB4_UpdatedAtAutoRefresh(t *testing.T) {
	env := setupTest(t)
	db := env.store.DB()

	// 在数据库中预置一个已知 updated_at 的 issue 记录
	seedIssue(db, "test/repo", 10, "claimed", "sess_b4", "2026-05-27T10:00:00Z")
	db.Exec(`UPDATE issue_claims SET updated_at = datetime('now', '-1 hour') WHERE repo_full_name = 'test/repo' AND issue_number = 10`)

	var before string
	db.QueryRow(`SELECT updated_at FROM issue_claims WHERE repo_full_name = 'test/repo' AND issue_number = 10`).Scan(&before)

	// 更新状态
	resp := env.post(t, "/api/issue/status",
		`{"repo_full_name":"test/repo","issue_number":10,"session_id":"sess_b4","status":"fixing"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// 验证 updated_at 已发生变化
	var after string
	db.QueryRow(`SELECT updated_at FROM issue_claims WHERE repo_full_name = 'test/repo' AND issue_number = 10`).Scan(&after)

	if after == before {
		t.Errorf("updated_at should have changed: before=%s after=%s", before, after)
	}

	// 验证 updated_at 是最近的时间（5 秒内）
	parsed, err := time.Parse(time.RFC3339, after)
	if err != nil {
		parsed, err = time.Parse("2006-01-02 15:04:05", after)
	}
	if err != nil {
		t.Fatalf("parse updated_at %q: %v", after, err)
	}
	if time.Since(parsed) > 5*time.Second {
		t.Errorf("updated_at not recent: %s (now: %s)", after, time.Now().Format("2006-01-02 15:04:05"))
	}
}

// TestB4_FullStatusTransitionChain 验收：完整状态流转链 claimed → fixing → ready-for-pr → pr-created → testing → reviewing → merged。
// 验证 issue 可以从初始 claimed 状态一步步流转到最终 merged 状态，每一步都返回正确的 previous_status。
func TestB4_FullStatusTransitionChain(t *testing.T) {
	env := setupTest(t)

	// 预置一个 claimed 状态的 issue
	seedIssue(env.store.DB(), "test/repo", 100, "claimed", "sess_chain", "2026-05-27T10:00:00Z")

	transitions := []struct {
		to   string
		from string
	}{
		{"fixing", "claimed"},
		{"ready-for-pr", "fixing"},
		{"pr-created", "ready-for-pr"},
		{"testing", "pr-created"},
		{"reviewing", "testing"},
		{"merged", "reviewing"},
	}

	// 遍历所有状态流转步骤
	for _, tr := range transitions {
		resp := env.post(t, "/api/issue/status",
			`{"repo_full_name":"test/repo","issue_number":100,"session_id":"sess_chain","status":"`+tr.to+`"}`)

		var result struct {
			Success        bool   `json:"success"`
			PreviousStatus string `json:"previous_status"`
			NewStatus      string `json:"new_status"`
		}
		readJSON(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("transition to %s: expected 200, got %d", tr.to, resp.StatusCode)
		}
		if !result.Success {
			t.Fatalf("transition to %s: expected success=true", tr.to)
		}
		if result.PreviousStatus != tr.from {
			t.Errorf("transition to %s: expected previous=%s, got %s", tr.to, tr.from, result.PreviousStatus)
		}
		if result.NewStatus != tr.to {
			t.Errorf("transition to %s: expected new=%s, got %s", tr.to, tr.to, result.NewStatus)
		}
	}

	// 验证最终状态为 merged
	statuses := checkStatuses(t, env, "test/repo", []int{100})
	if statuses[100] != "merged" {
		t.Fatalf("expected merged, got %s", statuses[100])
	}
}

// TestB4_FullStatusTransitionChain_Reject 验收：完整状态流转链 reviewing → rejected。
// 验证 issue 可以从 reviewing 状态流转到 rejected 终态。
func TestB4_FullStatusTransitionChain_Reject(t *testing.T) {
	env := setupTest(t)

	seedIssue(env.store.DB(), "test/repo", 101, "reviewing", "sess_reject", "2026-05-27T10:00:00Z")

	resp := env.post(t, "/api/issue/status",
		`{"repo_full_name":"test/repo","issue_number":101,"session_id":"sess_reject","status":"rejected"}`)

	var result struct {
		Success        bool   `json:"success"`
		PreviousStatus string `json:"previous_status"`
		NewStatus      string `json:"new_status"`
	}
	readJSON(t, resp, &result)

	if !result.Success {
		t.Fatal("reject should succeed")
	}
	if result.PreviousStatus != "reviewing" {
		t.Errorf("expected previous=reviewing, got %s", result.PreviousStatus)
	}
	if result.NewStatus != "rejected" {
		t.Errorf("expected new=rejected, got %s", result.NewStatus)
	}
}

// TestB4_PreviousStatusInResponse 验收：previous_status 在响应中正确返回（每个阶段）。
// 验证从 claimed 更新到 fixing，以及从 fixing 更新到 ready-for-pr 时，响应都包含正确的前一状态。
func TestB4_PreviousStatusInResponse(t *testing.T) {
	env := setupTest(t)
	seedIssue(env.store.DB(), "test/repo", 200, "claimed", "sess_prev", "2026-05-27T10:00:00Z")

	// 验证 claimed → fixing 的 previous_status
	resp := env.post(t, "/api/issue/status",
		`{"repo_full_name":"test/repo","issue_number":200,"session_id":"sess_prev","status":"fixing"}`)
	var r struct {
		Success        bool   `json:"success"`
		PreviousStatus string `json:"previous_status"`
	}
	readJSON(t, resp, &r)
	if !r.Success || r.PreviousStatus != "claimed" {
		t.Fatalf("fixing: success=%v previous=%s", r.Success, r.PreviousStatus)
	}

	// 验证 fixing → ready-for-pr 的 previous_status
	resp = env.post(t, "/api/issue/status",
		`{"repo_full_name":"test/repo","issue_number":200,"session_id":"sess_prev","status":"ready-for-pr"}`)
	readJSON(t, resp, &r)
	if !r.Success || r.PreviousStatus != "fixing" {
		t.Fatalf("ready-for-pr: success=%v previous=%s", r.Success, r.PreviousStatus)
	}
}

// TestB4_NonexistentIssueReturnsNotFound 验收：不存在的 issue 返回 not_found。
// 当尝试更新一个不存在的 issue 状态时，API 应返回 success=false 且 error=not_found。
func TestB4_NonexistentIssueReturnsNotFound(t *testing.T) {
	env := setupTest(t)

	resp := env.post(t, "/api/issue/status",
		`{"repo_full_name":"test/repo","issue_number":99999,"session_id":"sess_x","status":"fixing"}`)

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
}

// TestB4_NonOwnerBlockedAcrossStatuses 验收：非领取者无法更新（跨所有状态阶段）。
// 遍历所有状态阶段，验证非 issue 拥有者尝试更新状态时都会被拒绝，返回 not_owner 错误。
func TestB4_NonOwnerBlockedAcrossStatuses(t *testing.T) {
	env := setupTest(t)
	db := env.store.DB()

	statuses := []string{"claimed", "fixing", "ready-for-pr", "pr-created", "testing", "reviewing"}
	targets := []string{"fixing", "ready-for-pr", "pr-created", "testing", "reviewing", "merged"}

	for i, from := range statuses {
		t.Run(from+"_blocked", func(t *testing.T) {
			seedIssue(db, "test/repo", 300+i, from, "sess_owner", "2026-05-27T10:00:00Z")

			resp := env.post(t, "/api/issue/status",
				`{"repo_full_name":"test/repo","issue_number":`+fmt.Sprintf("%d", 300+i)+`,"session_id":"sess_other","status":"`+targets[i]+`"}`)

			var result struct {
				Success bool   `json:"success"`
				Error   string `json:"error"`
			}
			readJSON(t, resp, &result)

			if result.Success {
				t.Errorf("%s→%s: non-owner should not be able to update", from, targets[i])
			}
			if result.Error != "not_owner" {
				t.Errorf("%s→%s: expected not_owner, got %s", from, targets[i], result.Error)
			}
		})
	}
}

// TestB4_StatusMethodNotAllowed 验收：POST /api/issue/status 非 POST 方法返回 405。
// 验证使用 GET 请求访问状态更新接口时，服务器返回 405 Method Not Allowed。
func TestB4_StatusMethodNotAllowed(t *testing.T) {
	env := setupTest(t)

	resp, err := http.Get(env.srv.URL + "/api/issue/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
}
