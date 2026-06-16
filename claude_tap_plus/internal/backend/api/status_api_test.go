// Package api_test 包含 GET /api/status 接口的 BDD 验收测试。
// 覆盖：获取系统状态（含统计数据）、无活跃会话时状态、方法限制。
package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/api"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
)

// --- response types for status API ---

// statusResponse 是 GET /api/status 的响应结构。
type statusResponse struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	Stats         struct {
		ActiveSessions int64 `json:"active_sessions"`
		ActiveProxies  int64 `json:"active_proxies"`
		PendingIssues  int64 `json:"pending_issues"`
		TotalMachines  int64 `json:"total_machines"`
		TotalProjects  int64 `json:"total_projects"`
	} `json:"stats"`
	Timestamp string `json:"timestamp"`
}

// --- helpers ---

// statusTestEnv 是状态测试环境。
type statusTestEnv struct {
	srv       *httptest.Server
	store     *store.SQLiteStore
	startTime time.Time
}

// setupStatusTest 创建状态测试环境，包含 Status handler。
func setupStatusTest(t *testing.T) *statusTestEnv {
	t.Helper()

	f, err := os.CreateTemp("", "test-status-*.db")
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

	startTime := time.Now().Add(-3600 * time.Second) // 模拟已运行 3600 秒

	issueSvc := service.NewIssueService(s.Issues())
	sessionSvc := service.NewSessionService(s.Sessions())
	machineSvc := service.NewMachineService(s.Machines())
	projectSvc := service.NewProjectService(s.Projects())
	proxySvc := service.NewProxyService(s.Proxies())
	statusSvc := service.NewStatusService(
		s.Sessions(), s.Proxies(),
		s.Issues(), s.Machines(), s.Projects(),
		startTime,
	)

	router := api.NewRouter(api.Handlers{
		Issue:   api.NewIssueHandler(issueSvc),
		Session: api.NewSessionHandler(sessionSvc, issueSvc, service.NewTokenService(s.Sessions()), service.NewTraceService(s.Sessions())),
		Machine: api.NewMachineHandler(machineSvc),
		Project: api.NewProjectHandler(projectSvc),
		Proxy:   api.NewProxyHandlerWithService(proxySvc),
		Status:  api.NewStatusHandler(statusSvc),
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &statusTestEnv{srv: srv, store: s, startTime: startTime}
}

func (e *statusTestEnv) get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(e.srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func (e *statusTestEnv) post(t *testing.T, path, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(e.srv.URL+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// readStatusResponse 读取并解析状态响应。
func readStatusResponse(t *testing.T, resp *http.Response) statusResponse {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result statusResponse
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("parse JSON: %v, body: %s", err, b)
	}
	return result
}

// seedStatusData 通过各 API 预置 BDD Background 中的数据：
//   3 活跃会话、2 活跃代理、5 待处理 Issue、4 机器、2 项目
func seedStatusData(t *testing.T, env *statusTestEnv) {
	t.Helper()
	db := env.store.DB()
	now := time.Now().Format("2006-01-02 15:04:05")

	// 插入 4 台机器
	for _, m := range []struct {
		mid, os, host, user string
	}{
		{"user@host-1", "linux", "host-1", "user"},
		{"dev@host-2", "windows", "host-2", "dev"},
		{"admin@host-3", "macos", "host-3", "admin"},
		{"extra@host-4", "linux", "host-4", "extra"},
	} {
		_, err := db.Exec(
			`INSERT OR IGNORE INTO machines (machine_id, os, hostname, username, first_seen_at) VALUES (?, ?, ?, ?, ?)`,
			m.mid, m.os, m.host, m.user, now)
		if err != nil {
			t.Fatalf("seed machine %s: %v", m.mid, err)
		}
	}

	// 插入 2 个项目
	for _, p := range []struct {
		slug, cwd string
	}{
		{"proj-a", "/tmp/proj-a"},
		{"proj-b", "/tmp/proj-b"},
	} {
		_, err := db.Exec(
			`INSERT OR IGNORE INTO projects (project_slug, project_cwd, first_seen_at, last_seen_at) VALUES (?, ?, ?, ?)`,
			p.slug, p.cwd, now, now)
		if err != nil {
			t.Fatalf("seed project %s: %v", p.slug, err)
		}
	}

	// 插入 3 个活跃会话
	for i, s := range []struct {
		sid, mid, os, proj string
	}{
		{"sess-s1", "user@host-1", "linux", "proj-a"},
		{"sess-s2", "dev@host-2", "windows", "proj-a"},
		{"sess-s3", "admin@host-3", "macos", "proj-b"},
	} {
		_, err := db.Exec(
			`INSERT OR IGNORE INTO sessions (session_id, machine_id, os, project_slug, project_cwd, transcript_path, status, registered_at)
			 VALUES (?, ?, ?, ?, '/tmp', '/tmp/t.jsonl', 'active', ?)`,
			s.sid, s.mid, s.os, s.proj, now)
		if err != nil {
			t.Fatalf("seed session %d: %v", i, err)
		}
	}

	// 插入 2 个活跃代理
	for _, p := range []struct {
		id, proj, status string
	}{
		{"proxy-1", "proj-a", "active"},
		{"proxy-2", "proj-b", "active"},
	} {
		_, err := db.Exec(
			`INSERT INTO proxies (proxy_id, project_slug, status, registered_at) VALUES (?, ?, ?, ?)`,
			p.id, p.proj, p.status, time.Now().Format("2006-01-02 15:04:05"))
		if err != nil {
			t.Fatalf("seed proxy %s: %v", p.id, err)
		}
	}

	// 插入 5 个待处理 Issue (status=idle)
	for i := 1; i <= 5; i++ {
		_, err := db.Exec(
			`INSERT INTO issue_claims (repo_full_name, issue_number, status, updated_at) VALUES (?, ?, 'idle', CURRENT_TIMESTAMP)`,
			fmt.Sprintf("org/repo-%d", (i-1)/3+1), i)
		if err != nil {
			t.Fatalf("seed issue %d: %v", i, err)
		}
	}
}

// --- BDD Scenario tests ---

// TestGetStatus_WithData 验证：获取系统状态（含统计数据）。
// BDD: @positive Scenario: 获取系统状态
func TestGetStatus_WithData(t *testing.T) {
	env := setupStatusTest(t)
	seedStatusData(t, env)

	resp := env.get(t, "/api/status")
	result := readStatusResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if result.Status != "healthy" {
		t.Errorf("expected status=healthy, got %s", result.Status)
	}
	if result.Version == "" {
		t.Error("expected non-empty version")
	}
	if result.UptimeSeconds < 3590 { // 接近 3600 秒
		t.Errorf("expected uptime_seconds ~3600, got %d", result.UptimeSeconds)
	}
	if result.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}

	// 验证统计数据
	if result.Stats.ActiveSessions != 3 {
		t.Errorf("expected active_sessions=3, got %d", result.Stats.ActiveSessions)
	}
	if result.Stats.ActiveProxies != 2 {
		t.Errorf("expected active_proxies=2, got %d", result.Stats.ActiveProxies)
	}
	if result.Stats.PendingIssues != 5 {
		t.Errorf("expected pending_issues=5, got %d", result.Stats.PendingIssues)
	}
	if result.Stats.TotalMachines != 4 {
		t.Errorf("expected total_machines=4, got %d", result.Stats.TotalMachines)
	}
	if result.Stats.TotalProjects != 2 {
		t.Errorf("expected total_projects=2, got %d", result.Stats.TotalProjects)
	}
}

// TestGetStatus_NoActiveSessions 验证：无活跃会话时状态正常。
// BDD: @positive Scenario: 系统无活跃会话时状态正常
func TestGetStatus_NoActiveSessions(t *testing.T) {
	env := setupStatusTest(t)
	// 不插入任何数据

	resp := env.get(t, "/api/status")
	result := readStatusResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if result.Status != "healthy" {
		t.Errorf("expected status=healthy, got %s", result.Status)
	}
	if result.Stats.ActiveSessions != 0 {
		t.Errorf("expected active_sessions=0, got %d", result.Stats.ActiveSessions)
	}
	if result.Stats.ActiveProxies != 0 {
		t.Errorf("expected active_proxies=0, got %d", result.Stats.ActiveProxies)
	}
	if result.Stats.PendingIssues != 0 {
		t.Errorf("expected pending_issues=0, got %d", result.Stats.PendingIssues)
	}
	if result.Stats.TotalMachines != 0 {
		t.Errorf("expected total_machines=0, got %d", result.Stats.TotalMachines)
	}
	if result.Stats.TotalProjects != 0 {
		t.Errorf("expected total_projects=0, got %d", result.Stats.TotalProjects)
	}
}

// TestGetStatus_MethodNotAllowed 验证：使用非 GET 方法请求返回 405。
// BDD: @negative Scenario: 使用非 GET 方法请求
func TestGetStatus_MethodNotAllowed(t *testing.T) {
	env := setupStatusTest(t)

	resp := env.post(t, "/api/status", "")
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
