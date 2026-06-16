// Package api_test 包含 /api/proxy/trace-init 精确路由的验收测试。
// 覆盖：PID 精确路由、注销清理映射、参数校验。
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

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/api"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
)

// traceInitResponse 是 /_internal/trace-init 的响应。
type traceInitResponse struct {
	Status    string `json:"status"`
	TracePath string `json:"trace_path"`
}

// traceInitTestEnv 是 trace-init 路由测试环境。
type traceInitTestEnv struct {
	srv     *httptest.Server
	handler *api.ProxyHandler
	proxyA  *httptest.Server
	proxyB  *httptest.Server
	aCalls  []string
	bCalls  []string
}

// setupTraceInitTest 创建带 mock proxy A/B 的测试环境。
func setupTraceInitTest(t *testing.T) *traceInitTestEnv {
	t.Helper()

	f, err := os.CreateTemp("", "test-trace-init-*.db")
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
	proxySvc := service.NewProxyService(s.Proxies())
	proxyHandler := api.NewProxyHandlerWithService(proxySvc)

	router := api.NewRouter(api.Handlers{
		Issue:   api.NewIssueHandler(issueSvc),
		Session: api.NewSessionHandler(sessionSvc, issueSvc, service.NewTokenService(s.Sessions()), service.NewTraceService(s.Sessions())),
		Proxy:   proxyHandler,
	})

	env := &traceInitTestEnv{
		handler: proxyHandler,
	}

	// mock proxy A：记录每次收到的 session_id，返回 trace_path_a
	env.proxyA = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_internal/trace-init" {
			http.NotFound(w, r)
			return
		}
		b, _ := io.ReadAll(r.Body)
		var body struct {
			SessionID string `json:"session_id"`
		}
		json.Unmarshal(b, &body)
		env.aCalls = append(env.aCalls, body.SessionID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(traceInitResponse{Status: "ok", TracePath: "/traces/a_" + body.SessionID + ".jsonl"})
	}))

	// mock proxy B：记录每次收到的 session_id，返回 trace_path_b
	env.proxyB = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_internal/trace-init" {
			http.NotFound(w, r)
			return
		}
		b, _ := io.ReadAll(r.Body)
		var body struct {
			SessionID string `json:"session_id"`
		}
		json.Unmarshal(b, &body)
		env.bCalls = append(env.bCalls, body.SessionID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(traceInitResponse{Status: "ok", TracePath: "/traces/b_" + body.SessionID + ".jsonl"})
	}))

	env.srv = httptest.NewServer(router)
	t.Cleanup(func() {
		env.srv.Close()
		env.proxyA.Close()
		env.proxyB.Close()
	})

	return env
}

// post 发送 POST 请求到测试环境。
func (e *traceInitTestEnv) post(t *testing.T, path, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(e.srv.URL+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// registerProxy 向 backend 注册一个 mock proxy。
func (e *traceInitTestEnv) registerProxy(t *testing.T, pid, url string) {
	t.Helper()
	resp := e.post(t, "/api/proxy/register", fmt.Sprintf(`{"pid":"%s","proxy_url":"%s"}`, pid, url))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register proxy %s: expected 200, got %d", pid, resp.StatusCode)
	}
}

// unregisterProxy 向 backend 注销一个 mock proxy。
func (e *traceInitTestEnv) unregisterProxy(t *testing.T, pid string) {
	t.Helper()
	resp := e.post(t, "/api/proxy/unregister", fmt.Sprintf(`{"pid":"%s"}`, pid))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unregister proxy %s: expected 200, got %d", pid, resp.StatusCode)
	}
}

// traceInitWithPID 使用指定 proxy_pid 请求 /api/proxy/trace-init。
func (e *traceInitTestEnv) traceInitWithPID(t *testing.T, sessionID, proxyPID string) traceInitResponse {
	t.Helper()
	body := fmt.Sprintf(`{"session_id":"%s","proxy_pid":"%s"}`, sessionID, proxyPID)
	resp := e.post(t, "/api/proxy/trace-init", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("trace-init %s: expected 200, got %d, body=%s", sessionID, resp.StatusCode, b)
	}
	var result traceInitResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

// TestTraceInit_PIDRoute 验证：通过 proxy_pid 精确路由到目标代理。
func TestTraceInit_PIDRoute(t *testing.T) {
	env := setupTraceInitTest(t)

	env.registerProxy(t, "pid-A", env.proxyA.URL)
	env.registerProxy(t, "pid-B", env.proxyB.URL)

	// 指定 pid-B，应该路由到 proxy B
	resp := env.traceInitWithPID(t, "session-S", "pid-B")
	if resp.TracePath != "/traces/b_session-S.jsonl" {
		t.Fatalf("expected proxy B, got %s", resp.TracePath)
	}

	// 验证只有 B 被调用
	if len(env.aCalls) != 0 {
		t.Fatalf("expected proxy A not called, got %d calls", len(env.aCalls))
	}
	if len(env.bCalls) != 1 {
		t.Fatalf("expected proxy B called once, got %d calls", len(env.bCalls))
	}
}

// TestTraceInit_PIDRouteToA 验证：两个 proxy 注册，指定 pid-A 路由到 A。
func TestTraceInit_PIDRouteToA(t *testing.T) {
	env := setupTraceInitTest(t)

	env.registerProxy(t, "pid-A", env.proxyA.URL)
	env.registerProxy(t, "pid-B", env.proxyB.URL)

	resp := env.traceInitWithPID(t, "session-X", "pid-A")
	if resp.TracePath != "/traces/a_session-X.jsonl" {
		t.Fatalf("expected proxy A, got %s", resp.TracePath)
	}
}

// TestTraceInit_UnregisterThenRoute 验证：注销 proxy 后，再路由到该 PID 返回 404。
func TestTraceInit_UnregisterThenRoute(t *testing.T) {
	env := setupTraceInitTest(t)

	env.registerProxy(t, "pid-A", env.proxyA.URL)
	env.traceInitWithPID(t, "session-S", "pid-A")

	env.unregisterProxy(t, "pid-A")

	// pid-A 已注销，路由应返回 404
	body := `{"session_id":"session-S","proxy_pid":"pid-A"}`
	resp := env.post(t, "/api/proxy/trace-init", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after unregister, got %d", resp.StatusCode)
	}
}

// TestTraceInit_MissingProxyPID 验证：缺少 proxy_pid 返回 400。
func TestTraceInit_MissingProxyPID(t *testing.T) {
	env := setupTraceInitTest(t)
	env.registerProxy(t, "pid-A", env.proxyA.URL)

	resp := env.post(t, "/api/proxy/trace-init", `{"session_id":"session-S"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing proxy_pid, got %d", resp.StatusCode)
	}
}

// TestTraceInit_MissingSessionID 验证：缺少 session_id 返回 400。
func TestTraceInit_MissingSessionID(t *testing.T) {
	env := setupTraceInitTest(t)
	env.registerProxy(t, "pid-A", env.proxyA.URL)

	resp := env.post(t, "/api/proxy/trace-init", `{"proxy_pid":"pid-A"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing session_id, got %d", resp.StatusCode)
	}
}

// TestTraceInit_PIDNotFound 验证：proxy_pid 未注册返回 404。
func TestTraceInit_PIDNotFound(t *testing.T) {
	env := setupTraceInitTest(t)
	env.registerProxy(t, "pid-A", env.proxyA.URL)

	// 请求路由到未注册的 pid-C
	resp := env.post(t, "/api/proxy/trace-init", `{"session_id":"session-S","proxy_pid":"pid-C"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown PID, got %d", resp.StatusCode)
	}
}
