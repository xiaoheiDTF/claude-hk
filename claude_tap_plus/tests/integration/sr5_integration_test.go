// Package integration_test 包含 SR-5（配置与集成）的端到端测试。
// 验证会话注册、机器去重、状态保持、端到端完整流程等功能。
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

// --- SR-5: 配置与集成测试 ---
// 验证矩阵覆盖：
//   - 会话注册 → 机器/项目自动创建
//   - 多会话同机器 → machine 去重
//   - 会话注销 → closed 状态
//   - 后端重启 → 状态保持
//   - 端到端完整流程：register → 操作 → close

// sr5Env 是 SR-5 集成测试环境，支持 session 相关路由。
type sr5Env struct {
	srv   *httptest.Server
	store *store.SQLiteStore
	db    *sql.DB
}

// setupSR5 创建支持 session 的集成测试环境。
func setupSR5(t *testing.T) *sr5Env {
	t.Helper()

	f, err := os.CreateTemp("", "sr5-test-*.db")
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
		Session: api.NewSessionHandler(sessionSvc, service.NewIssueService(s.Issues()), service.NewTokenService(s.Sessions()), service.NewTraceService(s.Sessions())),
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &sr5Env{srv: srv, store: s, db: s.DB()}
}

// post 发送 POST 请求到 SR-5 测试环境。
func (e *sr5Env) post(t *testing.T, path, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(e.srv.URL+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// get 发送 GET 请求到 SR-5 测试环境。
func (e *sr5Env) get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(e.srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// sr5ReadJSON 读取 HTTP 响应并解析为 JSON。
func sr5ReadJSON(t *testing.T, resp *http.Response, v any) {
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

// --- 验证：会话注册 + 机器/项目自动创建 ---

// TestSR5_RegisterCreatesMachineAndProject 验证：注册 session 时自动创建机器和项目记录。
func TestSR5_RegisterCreatesMachineAndProject(t *testing.T) {
	env := setupSR5(t)

	resp := env.post(t, "/api/session/register", `{
		"session_id": "sr5-sess-1",
		"machine_id": "alice@devbox",
		"os": "linux",
		"project_slug": "my-app",
		"project_cwd": "/home/alice/my-app",
		"transcript_path": "/home/alice/.claude/transcripts/sr5-sess-1.jsonl",
		"model": "GLM-5.1",
		"source": "startup"
	}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register: expected 200, got %d", resp.StatusCode)
	}

	// 验证机器自动创建
	var machineCount int
	env.db.QueryRow("SELECT COUNT(*) FROM machines WHERE machine_id = 'alice@devbox'").Scan(&machineCount)
	if machineCount != 1 {
		t.Errorf("expected 1 machine, got %d", machineCount)
	}

	// 验证项目自动创建
	var projectCount int
	env.db.QueryRow("SELECT COUNT(*) FROM projects WHERE project_slug = 'my-app'").Scan(&projectCount)
	if projectCount != 1 {
		t.Errorf("expected 1 project, got %d", projectCount)
	}

	// 验证 session 状态活跃且元数据正确
	var status, model, source string
	env.db.QueryRow("SELECT status, model, source FROM sessions WHERE session_id = 'sr5-sess-1'").Scan(&status, &model, &source)
	if status != "active" {
		t.Errorf("expected active, got %s", status)
	}
	if model != "GLM-5.1" {
		t.Errorf("expected model=GLM-5.1, got %s", model)
	}
	if source != "startup" {
		t.Errorf("expected source=startup, got %s", source)
	}
}

// --- 验证：多会话同机器 → machine 去重，project 累积 ---

// TestSR5_MultipleSessionsSameMachine 验证：同一机器注册多个 session 时机器表去重，项目表累积。
func TestSR5_MultipleSessionsSameMachine(t *testing.T) {
	env := setupSR5(t)

	for i := 0; i < 3; i++ {
		body := fmt.Sprintf(`{
			"session_id": "sr5-sess-%d",
			"machine_id": "bob@laptop",
			"os": "windows",
			"project_slug": "proj-%d",
			"project_cwd": "/proj-%d",
			"transcript_path": "/t/%d.jsonl"
		}`, i, i, i, i)
		resp := env.post(t, "/api/session/register", body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("register %d: expected 200, got %d", i, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// 机器表应只有 1 行（去重）
	var machineCount int
	env.db.QueryRow("SELECT COUNT(*) FROM machines WHERE machine_id = 'bob@laptop'").Scan(&machineCount)
	if machineCount != 1 {
		t.Errorf("expected 1 machine (deduplicated), got %d", machineCount)
	}

	// 项目表应有 3 行
	var projectCount int
	env.db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&projectCount)
	if projectCount != 3 {
		t.Errorf("expected 3 projects, got %d", projectCount)
	}

	// sessions 表应有 3 行
	var sessionCount int
	env.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE machine_id = 'bob@laptop'").Scan(&sessionCount)
	if sessionCount != 3 {
		t.Errorf("expected 3 sessions, got %d", sessionCount)
	}
}

// --- 验证：端到端完整流程（register → issue 操作 → close）---

// TestSR5_EndToEndSessionWithIssueOps 验证：从注册到 issue 领取、状态更新、session 释放、关闭的完整流程。
func TestSR5_EndToEndSessionWithIssueOps(t *testing.T) {
	env := setupSR5(t)

	// 1. 注册 session
	resp := env.post(t, "/api/session/register", `{
		"session_id": "sr5-e2e",
		"machine_id": "e2e@test",
		"os": "linux",
		"project_slug": "e2e-proj",
		"project_cwd": "/e2e",
		"transcript_path": "/e2e.jsonl",
		"model": "GLM-5.1",
		"source": "startup"
	}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("step 1 register: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. 领取 issue（模拟 003-4-issue-claim）
	resp = env.post(t, "/api/issue/claim", `{
		"repo_full_name": "test/sr5",
		"issue_number": 1,
		"session_id": "sr5-e2e"
	}`)
	var claimResult struct {
		Success bool `json:"success"`
	}
	sr5ReadJSON(t, resp, &claimResult)
	if !claimResult.Success {
		t.Fatal("step 2 claim: expected success")
	}

	// 3. 更新状态（模拟 003-5-issue-fix）
	resp = env.post(t, "/api/issue/status", `{
		"repo_full_name": "test/sr5",
		"issue_number": 1,
		"session_id": "sr5-e2e",
		"status": "fixing"
	}`)
	var statusResult struct {
		Success bool `json:"success"`
	}
	sr5ReadJSON(t, resp, &statusResult)
	if !statusResult.Success {
		t.Fatal("step 3 status: expected success")
	}

	// 4. 验证 session 仍活跃
	resp = env.get(t, "/api/session/sr5-e2e")
	var sessionDetail struct {
		Status string `json:"status"`
	}
	sr5ReadJSON(t, resp, &sessionDetail)
	if sessionDetail.Status != "active" {
		t.Fatalf("step 4: expected active, got %s", sessionDetail.Status)
	}

	// 5. 释放 session 关联的 issue（模拟 SessionEnd）
	resp = env.post(t, "/api/issue/release-session", `{"session_id": "sr5-e2e"}`)
	var releaseResult struct {
		Count int `json:"count"`
	}
	sr5ReadJSON(t, resp, &releaseResult)
	if releaseResult.Count != 1 {
		t.Fatalf("step 5 release: expected 1, got %d", releaseResult.Count)
	}

	// 6. 关闭 session（模拟 SR-3 unregister）
	resp = env.post(t, "/api/session/close", `{
		"session_id": "sr5-e2e",
		"reason": "prompt_input_exit"
	}`)
	var closeResult struct {
		Status string `json:"status"`
	}
	sr5ReadJSON(t, resp, &closeResult)
	if closeResult.Status != "closed" {
		t.Fatalf("step 6 close: expected closed, got %s", closeResult.Status)
	}

	// 7. 验证最终状态
	resp = env.get(t, "/api/session/sr5-e2e")
	var finalDetail struct {
		Status      string  `json:"status"`
		CloseReason string  `json:"close_reason"`
		ClosedAt    *string `json:"closed_at"`
	}
	sr5ReadJSON(t, resp, &finalDetail)
	if finalDetail.Status != "closed" {
		t.Errorf("step 7: expected closed, got %s", finalDetail.Status)
	}
	if finalDetail.CloseReason != "prompt_input_exit" {
		t.Errorf("step 7: expected reason=prompt_input_exit, got %s", finalDetail.CloseReason)
	}
	if finalDetail.ClosedAt == nil {
		t.Error("step 7: expected closed_at to be set")
	}

	// 8. 验证 issue 已回到 idle
	resp = env.post(t, "/api/issue/check", `{"repo_full_name":"test/sr5","issue_numbers":[1]}`)
	var checkResult struct {
		Issues []struct {
			Number int    `json:"number"`
			Status string `json:"status"`
		} `json:"issues"`
	}
	sr5ReadJSON(t, resp, &checkResult)
	if len(checkResult.Issues) != 1 || checkResult.Issues[0].Status != "idle" {
		t.Errorf("step 8: expected issue idle, got %v", checkResult.Issues)
	}
}

// --- 验证：后端重启后状态保持 ---

// TestSR5_StatePreservedAcrossRestart 验证：关闭后重新启动服务，session 状态保持不变。
func TestSR5_StatePreservedAcrossRestart(t *testing.T) {
	// 阶段 1：创建数据库、注册 session、关闭服务
	f, err := os.CreateTemp("", "sr5-restart-*.db")
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
	sessionSvc := service.NewSessionService(s.Sessions())
	router := api.NewRouter(api.Handlers{
		Issue:   api.NewIssueHandler(issueSvc),
		Session: api.NewSessionHandler(sessionSvc, service.NewIssueService(s.Issues()), service.NewTokenService(s.Sessions()), service.NewTraceService(s.Sessions())),
	})
	srv := httptest.NewServer(router)

	// 注册并关闭 session
	resp, _ := http.Post(srv.URL+"/api/session/register", "application/json",
		strings.NewReader(`{"session_id":"sr5-restart","machine_id":"test@test","os":"linux","project_slug":"proj","project_cwd":"/p","transcript_path":"/t.jsonl"}`))
	resp.Body.Close()

	resp, _ = http.Post(srv.URL+"/api/session/close", "application/json",
		strings.NewReader(`{"session_id":"sr5-restart","reason":"prompt_input_exit"}`))
	resp.Body.Close()

	// 关闭服务
	srv.Close()
	s.Close()

	// 阶段 2：使用同一数据库重新启动服务
	s2, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		s2.Close()
		os.Remove(dbPath)
	})

	sessionSvc2 := service.NewSessionService(s2.Sessions())
	router2 := api.NewRouter(api.Handlers{
		Issue:   api.NewIssueHandler(service.NewIssueService(s2.Issues())),
		Session: api.NewSessionHandler(sessionSvc2, service.NewIssueService(s2.Issues()), service.NewTokenService(s2.Sessions()), service.NewTraceService(s2.Sessions())),
	})
	srv2 := httptest.NewServer(router2)
	t.Cleanup(srv2.Close)

	// 验证 session 仍关闭
	resp, _ = http.Get(srv2.URL + "/api/session/sr5-restart")
	if err != nil {
		t.Fatal(err)
	}
	var detail struct {
		Status      string  `json:"status"`
		CloseReason string  `json:"close_reason"`
		ClosedAt    *string `json:"closed_at"`
	}
	sr5ReadJSON(t, resp, &detail)

	if detail.Status != "closed" {
		t.Errorf("expected closed after restart, got %s", detail.Status)
	}
	if detail.CloseReason != "prompt_input_exit" {
		t.Errorf("expected reason preserved, got %s", detail.CloseReason)
	}
	if detail.ClosedAt == nil {
		t.Error("expected closed_at preserved")
	}
}

// --- 验证：重复注册返回 409 ---

// TestSR5_DuplicateRegisterReturns409 验证：重复注册返回 409 Conflict。
func TestSR5_DuplicateRegisterReturns409(t *testing.T) {
	env := setupSR5(t)

	body := `{"session_id":"sr5-dup","machine_id":"test@test","os":"linux","project_slug":"proj","project_cwd":"/p","transcript_path":"/t.jsonl"}`
	resp := env.post(t, "/api/session/register", body)
	resp.Body.Close()

	resp = env.post(t, "/api/session/register", body)
	var errResp struct {
		Code string `json:"error"`
	}
	sr5ReadJSON(t, resp, &errResp)

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if errResp.Code != "session_exists" {
		t.Errorf("expected session_exists, got %s", errResp.Code)
	}
}

// --- 验证：session list 过滤 ---

// TestSR5_SessionListFilters 验证：session 列表支持按状态和 machine_id 过滤。
func TestSR5_SessionListFilters(t *testing.T) {
	env := setupSR5(t)

	// 准备数据：2 个活跃 session（machine_a），1 个已关闭（machine_b）
	for i, s := range []struct {
		sid, machine, slug, cwd string
	}{
		{"sa1", "ma", "proj-a", "/a"},
		{"sa2", "ma", "proj-b", "/b"},
		{"sb1", "mb", "proj-c", "/c"},
	} {
		body := fmt.Sprintf(`{"session_id":"%s","machine_id":"%s","os":"linux","project_slug":"%s","project_cwd":"%s","transcript_path":"/t%d.jsonl"}`,
			s.sid, s.machine, s.slug, s.cwd, i)
		resp := env.post(t, "/api/session/register", body)
		resp.Body.Close()
	}

	// 关闭 sb1
	resp := env.post(t, "/api/session/close", `{"session_id":"sb1","reason":"done"}`)
	resp.Body.Close()

	// 过滤：status=active
	resp = env.get(t, "/api/sessions?status=active")
	var activeList struct {
		Sessions []struct {
			SessionID string `json:"session_id"`
		} `json:"sessions"`
	}
	sr5ReadJSON(t, resp, &activeList)
	if len(activeList.Sessions) != 2 {
		t.Errorf("active filter: expected 2, got %d", len(activeList.Sessions))
	}

	// 过滤：machine_id=ma
	resp = env.get(t, "/api/sessions?machine_id=ma")
	var machineList struct {
		Sessions []struct {
			SessionID string `json:"session_id"`
		} `json:"sessions"`
	}
	sr5ReadJSON(t, resp, &machineList)
	if len(machineList.Sessions) != 2 {
		t.Errorf("machine filter: expected 2, got %d", len(machineList.Sessions))
	}

	// 过滤：status=closed
	resp = env.get(t, "/api/sessions?status=closed")
	var closedList struct {
		Sessions []struct {
			SessionID string `json:"session_id"`
		} `json:"sessions"`
	}
	sr5ReadJSON(t, resp, &closedList)
	if len(closedList.Sessions) != 1 {
		t.Errorf("closed filter: expected 1, got %d", len(closedList.Sessions))
	}
	if len(closedList.Sessions) > 0 && closedList.Sessions[0].SessionID != "sb1" {
		t.Errorf("closed filter: expected sb1, got %s", closedList.Sessions[0].SessionID)
	}
}
