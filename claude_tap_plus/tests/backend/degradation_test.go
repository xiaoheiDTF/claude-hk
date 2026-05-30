// Package backend_test 包含后端 API 的降级与恢复测试，覆盖服务重启、异常请求等场景。
package backend_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/api"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
)

// --- helpers ---

// setupDegradationTest 创建带临时数据库的降级测试环境，返回测试环境和数据库路径。
func setupDegradationTest(t *testing.T) (*testEnv, string) {
	t.Helper()

	f, err := os.CreateTemp("", "test-degradation-*.db")
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

	issueSvc := service.NewIssueService(s.Issues())
	router := api.NewRouter(api.Handlers{
		Issue: api.NewIssueHandler(issueSvc),
	})

	srv := httptest.NewServer(router)

	return &testEnv{srv: srv, store: s}, dbPath
}

// setupDegradationTestWithDB 使用已有数据库路径创建测试环境，用于验证服务重启后状态保持。
func setupDegradationTestWithDB(t *testing.T, dbPath string) *testEnv {
	t.Helper()

	s, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	issueSvc := service.NewIssueService(s.Issues())
	router := api.NewRouter(api.Handlers{
		Issue: api.NewIssueHandler(issueSvc),
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &testEnv{srv: srv, store: s}
}

// claimViaAPI 通过 API 调用领取指定 issue，返回 HTTP 响应。
func claimViaAPI(t *testing.T, url, repo string, number int, sessionID string) *http.Response {
	t.Helper()
	body := `{"repo_full_name":"` + repo + `","issue_number":` + strconv.Itoa(number) + `,"session_id":"` + sessionID + `"}`
	resp, err := http.Post(url+"/api/issue/claim", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// statusViaAPI 通过 API 调用更新 issue 状态，返回 HTTP 响应。
func statusViaAPI(t *testing.T, url, repo string, number int, sessionID, status string) *http.Response {
	t.Helper()
	body := `{"repo_full_name":"` + repo + `","issue_number":` + strconv.Itoa(number) + `,"session_id":"` + sessionID + `","status":"` + status + `"}`
	resp, err := http.Post(url+"/api/issue/status", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// releaseSessionViaAPI 通过 API 调用按 session 批量释放 issue，返回 HTTP 响应。
func releaseSessionViaAPI(t *testing.T, url, sessionID string) *http.Response {
	t.Helper()
	body := `{"session_id":"` + sessionID + `"}`
	resp, err := http.Post(url+"/api/issue/release-session", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// --- T12/T13: Recovery tests ---

// TestRecovery_StatePreservedAcrossRestart 验证：服务重启后 issue 状态保持。
// 先创建并更新 issue 状态，关闭服务后使用同一数据库重新启动，验证状态未丢失。
func TestRecovery_StatePreservedAcrossRestart(t *testing.T) {
	// Phase 1: 创建数据库、写入数据、启动服务
	env, dbPath := setupDegradationTest(t)

	claimViaAPI(t, env.srv.URL, "test/repo", 10, "sess_a")
	statusViaAPI(t, env.srv.URL, "test/repo", 10, "sess_a", "fixing")

	// Phase 2: 关闭服务
	env.srv.Close()
	env.store.Close()

	// Phase 3: 使用同一数据库重新启动服务
	env2 := setupDegradationTestWithDB(t, dbPath)
	t.Cleanup(func() { os.Remove(dbPath) })

	// Phase 4: 验证状态保持
	db := env2.store.DB()
	var status, sid string
	err := db.QueryRow(
		`SELECT status, session_id FROM issue_claims WHERE repo_full_name = ? AND issue_number = ?`,
		"test/repo", 10,
	).Scan(&status, &sid)
	if err != nil {
		t.Fatalf("failed to query issue: %v", err)
	}
	if status != "fixing" {
		t.Errorf("expected status=fixing, got %s", status)
	}
	if sid != "sess_a" {
		t.Errorf("expected session_id=sess_a, got %s", sid)
	}
}

// TestRecovery_ClaimFailsForAlreadyClaimedAfterRestart 验证：重启后已被领取的 issue 无法重复领取。
func TestRecovery_ClaimFailsForAlreadyClaimedAfterRestart(t *testing.T) {
	env, dbPath := setupDegradationTest(t)

	claimViaAPI(t, env.srv.URL, "test/repo", 20, "sess_a")

	env.srv.Close()
	env.store.Close()

	env2 := setupDegradationTestWithDB(t, dbPath)
	t.Cleanup(func() { os.Remove(dbPath) })

	resp := claimViaAPI(t, env2.srv.URL, "test/repo", 20, "sess_b")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	readJSON(t, resp, &result)
	if result["success"] == true {
		t.Error("expected success=false for already claimed issue")
	}
}

// TestRecovery_ReleaseSessionAfterRestart 验证：重启后 release-session API 正常工作。
func TestRecovery_ReleaseSessionAfterRestart(t *testing.T) {
	env, dbPath := setupDegradationTest(t)

	claimViaAPI(t, env.srv.URL, "test/repo", 30, "sess_a")
	statusViaAPI(t, env.srv.URL, "test/repo", 30, "sess_a", "fixing")

	env.srv.Close()
	env.store.Close()

	env2 := setupDegradationTestWithDB(t, dbPath)
	t.Cleanup(func() { os.Remove(dbPath) })

	resp := releaseSessionViaAPI(t, env2.srv.URL, "sess_a")
	defer resp.Body.Close()

	var result struct {
		Released []int `json:"released"`
		Count    int   `json:"count"`
	}
	readJSON(t, resp, &result)

	if result.Count != 1 {
		t.Fatalf("expected 1 released, got %d", result.Count)
	}
	if len(result.Released) != 1 || result.Released[0] != 30 {
		t.Errorf("expected released=[30], got %v", result.Released)
	}
}

// --- T14: Malformed request tests ---

// TestMalformedRequest_InvalidJSON 验证：请求体不是合法 JSON 时返回 400。
func TestMalformedRequest_InvalidJSON(t *testing.T) {
	env, _ := setupDegradationTest(t)
	t.Cleanup(func() { env.store.Close() })

	resp, err := http.Post(env.srv.URL+"/api/issue/claim", "application/json", strings.NewReader("not json"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// TestMalformedRequest_EmptyRequiredFields 验证：必填字段为空时返回 400。
func TestMalformedRequest_EmptyRequiredFields(t *testing.T) {
	env, _ := setupDegradationTest(t)
	t.Cleanup(func() { env.store.Close() })

	tests := []struct {
		name string
		body string
	}{
		{"empty repo", `{"repo_full_name":"","issue_number":1,"session_id":"s1"}`},
		{"zero issue_number", `{"repo_full_name":"r","issue_number":0,"session_id":"s1"}`},
		{"empty session_id", `{"repo_full_name":"r","issue_number":1,"session_id":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(env.srv.URL+"/api/issue/claim", "application/json", strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

// TestMalformedRequest_NegativeIssueNumber 验证：负数 issue number 被后端优雅处理（创建 idle 记录）。
func TestMalformedRequest_NegativeIssueNumber(t *testing.T) {
	env, _ := setupDegradationTest(t)
	t.Cleanup(func() { env.store.Close() })

	resp, err := http.Post(env.srv.URL+"/api/issue/check", "application/json",
		strings.NewReader(`{"repo_full_name":"test/repo","issue_numbers":[-1,-2]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// 后端应优雅处理负数（创建 idle 记录）
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// TestMalformedRequest_StatusEmptyFields 验证：状态更新接口中必填字段为空时返回 400。
func TestMalformedRequest_StatusEmptyFields(t *testing.T) {
	env, _ := setupDegradationTest(t)
	t.Cleanup(func() { env.store.Close() })

	tests := []struct {
		name string
		body string
	}{
		{"empty status", `{"repo_full_name":"r","issue_number":1,"session_id":"s1","status":""}`},
		{"empty repo", `{"repo_full_name":"","issue_number":1,"session_id":"s1","status":"fixing"}`},
		{"empty session", `{"repo_full_name":"r","issue_number":1,"session_id":"","status":"fixing"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(env.srv.URL+"/api/issue/status", "application/json", strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

// TestMalformedRequest_ExtraFieldsIgnored 验证：请求体中的额外字段被静默忽略。
func TestMalformedRequest_ExtraFieldsIgnored(t *testing.T) {
	env, _ := setupDegradationTest(t)
	t.Cleanup(func() { env.store.Close() })

	// 额外字段应被静默忽略
	resp, err := http.Post(env.srv.URL+"/api/issue/check", "application/json",
		strings.NewReader(`{"repo_full_name":"test/repo","issue_numbers":[1],"extra":"ignored"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// TestMalformedRequest_InvalidMethod 验证：使用错误的 HTTP 方法（GET）访问 claim 接口返回 405。
func TestMalformedRequest_InvalidMethod(t *testing.T) {
	env, _ := setupDegradationTest(t)
	t.Cleanup(func() { env.store.Close() })

	// GET 而非 POST
	resp, err := http.Get(env.srv.URL + "/api/issue/claim")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
}

// TestMalformedRequest_ReleaseSessionEmptySessionID 验证：release-session 接口缺少 session_id 时返回 400。
func TestMalformedRequest_ReleaseSessionEmptySessionID(t *testing.T) {
	env, _ := setupDegradationTest(t)
	t.Cleanup(func() { env.store.Close() })

	resp, err := http.Post(env.srv.URL+"/api/issue/release-session", "application/json",
		strings.NewReader(`{"session_id":""}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// TestMalformedRequest_UnicodeInFields 验证：请求字段中包含 Unicode 字符（如中文）时后端正常处理。
func TestMalformedRequest_UnicodeInFields(t *testing.T) {
	env, _ := setupDegradationTest(t)
	t.Cleanup(func() { env.store.Close() })

	body := `{"repo_full_name":"测试/仓库","issue_numbers":[1]}`
	resp, err := http.Post(env.srv.URL+"/api/issue/check", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
