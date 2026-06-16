// Package api_test 包含 API 的并发测试，验证 issue 领取的原子性。
package api_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// --- B3: 并发领取测试 ---
// 验收标准：并发领取时只有一个成功（原子性）

// TestConcurrentClaim 验证 issue 并发领取的原子性。
// 多个 goroutine 同时尝试领取同一个 idle 状态的 issue，只有一个能成功，其余应返回 already_claimed。
func TestConcurrentClaim(t *testing.T) {
	t.Run("only_one_succeeds", func(t *testing.T) {
		env := setupTest(t)

		db := env.store.DB()
		seedIssue(db, "test/repo", 10, "idle", "", "")

		var wg sync.WaitGroup
		var successCount int32
		var results [10]bool

		concurrency := 10
		wg.Add(concurrency)

		// 启动 10 个 goroutine 并发领取同一 issue
		for i := 0; i < concurrency; i++ {
			go func(idx int) {
				defer wg.Done()
				sessionID := fmt.Sprintf("sess_%d", idx)
				body := fmt.Sprintf(
					`{"repo_full_name":"test/repo","issue_number":10,"session_id":"%s"}`,
					sessionID)
				resp := env.post(t, "/api/issue/claim", body)
				var result struct {
					Success bool `json:"success"`
				}
				readJSON(t, resp, &result)
				if result.Success {
					atomic.AddInt32(&successCount, 1)
				}
				results[idx] = result.Success
			}(i)
		}

		wg.Wait()

		// 验证恰好只有一个领取成功
		if successCount != 1 {
			t.Fatalf("expected exactly 1 successful claim, got %d (results: %v)", successCount, results)
		}

		// 验证 issue 状态已变为 claimed
		statuses := checkStatuses(t, env, "test/repo", []int{10})
		if statuses[10] != "claimed" {
			t.Errorf("expected claimed after concurrent claims, got %s", statuses[10])
		}
	})

	t.Run("all_others_get_already_claimed", func(t *testing.T) {
		env := setupTest(t)

		db := env.store.DB()
		seedIssue(db, "test/repo", 20, "idle", "", "")

		var wg sync.WaitGroup
		var failCount int32

		concurrency := 10
		wg.Add(concurrency)

		// 启动 10 个 goroutine 并发领取，验证失败者都返回 already_claimed
		for i := 0; i < concurrency; i++ {
			go func(idx int) {
				defer wg.Done()
				sessionID := fmt.Sprintf("sess_f_%d", idx)
				body := fmt.Sprintf(
					`{"repo_full_name":"test/repo","issue_number":20,"session_id":"%s"}`,
					sessionID)
				resp := env.post(t, "/api/issue/claim", body)
				var result struct {
					Success bool   `json:"success"`
					Error   string `json:"error"`
				}
				readJSON(t, resp, &result)
				if !result.Success {
					if result.Error != "already_claimed" {
						t.Errorf("sess %d: expected already_claimed, got %s", idx, result.Error)
					}
					atomic.AddInt32(&failCount, 1)
				}
			}(i)
		}

		wg.Wait()

		// 验证有 concurrency-1 个失败请求
		if failCount != int32(concurrency-1) {
			t.Fatalf("expected %d failures, got %d", concurrency-1, failCount)
		}
	})
}
