// Package backend_test 包含 GET /api/proxies 接口的 BDD 验收测试。
// 覆盖：获取全部、按 status/project 过滤、组合过滤、空列表、方法限制。
package backend_test

import (
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

// --- response types for proxy API ---

// proxyListResponse 是 GET /api/proxies 的响应结构。
type proxyListResponse struct {
	Proxies []proxyItem `json:"proxies"`
	Total   int         `json:"total"`
}

// proxyItem 是代理列表中的单个条目。
type proxyItem struct {
	ProxyID      string `json:"proxy_id"`
	ProjectSlug  string `json:"project_slug"`
	Status       string `json:"status"`
	RegisteredAt string `json:"registered_at"`
	LastPingAt   string `json:"last_ping_at"`
}

// --- helpers ---

// proxyTestEnv 是代理测试环境。
type proxyTestEnv struct {
	srv   *httptest.Server
	store *store.SQLiteStore
}

// setupProxyTest 创建代理测试环境，包含 Proxy handler。
func setupProxyTest(t *testing.T) *proxyTestEnv {
	t.Helper()

	f, err := os.CreateTemp("", "test-proxy-*.db")
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
	machineSvc := service.NewMachineService(s.Machines())
	projectSvc := service.NewProjectService(s.Projects())
	proxySvc := service.NewProxyService(s.Proxies())

	router := api.NewRouter(api.Handlers{
		Issue:   api.NewIssueHandler(issueSvc),
		Session: api.NewSessionHandler(sessionSvc, issueSvc, service.NewTokenService(s.Sessions()), service.NewTraceService(s.Sessions())),
		Machine: api.NewMachineHandler(machineSvc),
		Project: api.NewProjectHandler(projectSvc),
		Proxy:   api.NewProxyHandlerWithService(proxySvc),
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &proxyTestEnv{srv: srv, store: s}
}

// get 发送 GET 请求到测试环境。
func (e *proxyTestEnv) get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(e.srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// post 发送 POST 请求到测试环境。
func (e *proxyTestEnv) post(t *testing.T, path, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(e.srv.URL+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// readProxyListResponse 读取并解析代理列表响应。
func readProxyListResponse(t *testing.T, resp *http.Response) proxyListResponse {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result proxyListResponse
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("parse JSON: %v, body: %s", err, b)
	}
	return result
}

// seedProxyData 在数据库中预置 BDD Background 中的 3 条代理数据。
//   proxy-1 | project-a | active  | 2026-06-01T10:00:00Z
//   proxy-2 | project-b | active  | 2026-06-01T11:00:00Z
//   proxy-3 | project-a | offline | 2026-06-01T09:00:00Z
func seedProxyData(t *testing.T, env *proxyTestEnv) {
	t.Helper()
	db := env.store.DB()
	inserts := []struct {
		id, project, status, registeredAt string
	}{
		{"proxy-1", "project-a", "active", "2026-06-01 10:00:00"},
		{"proxy-2", "project-b", "active", "2026-06-01 11:00:00"},
		{"proxy-3", "project-a", "offline", "2026-06-01 09:00:00"},
	}
	for _, ins := range inserts {
		_, err := db.Exec(
			`INSERT INTO proxies (proxy_id, project_slug, status, registered_at) VALUES (?, ?, ?, ?)`,
			ins.id, ins.project, ins.status, ins.registeredAt)
		if err != nil {
			t.Fatalf("seed proxy %s: %v", ins.id, err)
		}
	}
}

// --- BDD Scenario tests ---

// TestListProxies_GetAll 验证：获取所有代理列表。
// BDD: @positive Scenario: 获取所有代理列表
func TestListProxies_GetAll(t *testing.T) {
	env := setupProxyTest(t)
	seedProxyData(t, env)

	resp := env.get(t, "/api/proxies")
	result := readProxyListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if result.Total != 3 {
		t.Fatalf("expected total=3, got %d", result.Total)
	}
	if len(result.Proxies) != 3 {
		t.Fatalf("expected 3 proxies, got %d", len(result.Proxies))
	}

	// 验证每个代理包含必要字段
	for _, p := range result.Proxies {
		if p.ProxyID == "" {
			t.Error("expected non-empty proxy_id")
		}
		if p.ProjectSlug == "" {
			t.Error("expected non-empty project_slug")
		}
		if p.Status == "" {
			t.Error("expected non-empty status")
		}
		if p.RegisteredAt == "" {
			t.Error("expected non-empty registered_at")
		}
	}
}

// TestListProxies_FilterByStatus 验证：按状态过滤代理。
// BDD: @positive Scenario: 按状态过滤代理
func TestListProxies_FilterByStatus(t *testing.T) {
	env := setupProxyTest(t)
	seedProxyData(t, env)

	resp := env.get(t, "/api/proxies?status=active")
	result := readProxyListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Proxies) != 2 {
		t.Fatalf("expected 2 proxies, got %d", len(result.Proxies))
	}
	for _, p := range result.Proxies {
		if p.Status != "active" {
			t.Errorf("expected status=active, got %s", p.Status)
		}
	}
}

// TestListProxies_FilterByProject 验证：按项目过滤代理。
// BDD: @positive Scenario: 按项目过滤代理
func TestListProxies_FilterByProject(t *testing.T) {
	env := setupProxyTest(t)
	seedProxyData(t, env)

	resp := env.get(t, "/api/proxies?project=project-a")
	result := readProxyListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Proxies) != 2 {
		t.Fatalf("expected 2 proxies, got %d", len(result.Proxies))
	}
	for _, p := range result.Proxies {
		if p.ProjectSlug != "project-a" {
			t.Errorf("expected project_slug=project-a, got %s", p.ProjectSlug)
		}
	}
}

// TestListProxies_CombinedFilter 验证：组合过滤代理。
// BDD: @positive Scenario: 组合过滤代理
func TestListProxies_CombinedFilter(t *testing.T) {
	env := setupProxyTest(t)
	seedProxyData(t, env)

	resp := env.get(t, "/api/proxies?status=active&project=project-a")
	result := readProxyListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Proxies) != 1 {
		t.Fatalf("expected 1 proxy, got %d", len(result.Proxies))
	}
	if result.Proxies[0].ProxyID != "proxy-1" {
		t.Errorf("expected proxy_id=proxy-1, got %s", result.Proxies[0].ProxyID)
	}
}

// TestListProxies_EmptyResult 验证：无代理时返回空列表。
// BDD: @positive Scenario: 无代理时返回空列表
func TestListProxies_EmptyResult(t *testing.T) {
	env := setupProxyTest(t)

	resp := env.get(t, "/api/proxies")
	result := readProxyListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if result.Total != 0 {
		t.Fatalf("expected total=0, got %d", result.Total)
	}
	if len(result.Proxies) != 0 {
		t.Fatalf("expected empty proxies array, got %d items", len(result.Proxies))
	}
}

// TestListProxies_MethodNotAllowed 验证：使用非 GET 方法请求返回 405。
// BDD: @negative Scenario: 使用非 GET 方法请求
func TestListProxies_MethodNotAllowed(t *testing.T) {
	env := setupProxyTest(t)

	resp := env.post(t, "/api/proxies", "")
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
