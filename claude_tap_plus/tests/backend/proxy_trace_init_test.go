// Package backend_test 包含 /api/proxy/trace-init 精确路由的验收测试。
// 覆盖：多 proxy 场景下按 session_id 精确路由、resume 重新绑定、注销清理映射。
package backend_test

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
	Status     string `json:"status"`
	TracePath  string `json:"trace_path"`
}

// traceInitTestEnv 是 trace-init 路由测试环境。
type traceInitTestEnv struct {
	srv        *httptest.Server
	handler    *api.ProxyHandler
	proxyA     *httptest.Server
	proxyB     *httptest.Server
	aCalls     []string
	bCalls     []string
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

// traceInit 请求 /api/proxy/trace-init 并返回响应。
func (e *traceInitTestEnv) traceInit(t *testing.T, sessionID string) traceInitResponse {
	t.Helper()
	resp := e.post(t, "/api/proxy/trace-init", fmt.Sprintf(`{"session_id":"%s","machine_id":"u@h","project_slug":"proj"}`, sessionID))
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

// TestTraceInit_DirectRouteAfterFirstHit 验证：首次广播探测后，同一 session_id 直接精确路由。
func TestTraceInit_DirectRouteAfterFirstHit(t *testing.T) {
	env := setupTraceInitTest(t)

	env.registerProxy(t, "pid-A", env.proxyA.URL)
	env.registerProxy(t, "pid-B", env.proxyB.URL)

	// 第一次：广播探测（map 随机，假设 A 被选中；无论选中谁都会被记录）
	resp1 := env.traceInit(t, "session-S")
	expected := "/traces/a_session-S.jsonl"
	if resp1.TracePath != expected && resp1.TracePath != "/traces/b_session-S.jsonl" {
		t.Fatalf("unexpected trace path: %s", resp1.TracePath)
	}

	// 清除计数
	beforeA, beforeB := len(env.aCalls), len(env.bCalls)

	// 第二次：应该直接路由到同一代理，不广播
	resp2 := env.traceInit(t, "session-S")
	if resp2.TracePath != resp1.TracePath {
		t.Fatalf("expected same proxy on second init, got %s vs %s", resp2.TracePath, resp1.TracePath)
	}

	// 确认只有一个额外调用落在已绑定代理上
	afterA, afterB := len(env.aCalls), len(env.bCalls)
	if resp1.TracePath == expected {
		if afterA != beforeA+1 {
			t.Fatalf("expected proxy A called once on second init, A=%d->%d B=%d->%d", beforeA, afterA, beforeB, afterB)
		}
		if afterB != beforeB {
			t.Fatalf("expected proxy B not called on second init, A=%d->%d B=%d->%d", beforeA, afterA, beforeB, afterB)
		}
	} else {
		if afterB != beforeB+1 {
			t.Fatalf("expected proxy B called once on second init, A=%d->%d B=%d->%d", beforeA, afterA, beforeB, afterB)
		}
		if afterA != beforeA {
			t.Fatalf("expected proxy A not called on second init, A=%d->%d B=%d->%d", beforeA, afterA, beforeB, afterB)
		}
	}
}

// TestTraceInit_RebindOnResume 验证：注销旧 proxy 后同一 session_id 可重新绑定到新 proxy。
func TestTraceInit_RebindOnResume(t *testing.T) {
	env := setupTraceInitTest(t)

	env.registerProxy(t, "pid-A", env.proxyA.URL)
	env.registerProxy(t, "pid-B", env.proxyB.URL)

	resp1 := env.traceInit(t, "session-S")
	if resp1.TracePath == "" {
		t.Fatal("expected non-empty trace path")
	}

	// 模拟 resume：从 A 注销，B 仍在
	env.unregisterProxy(t, "pid-A")

	// 现在只有 B 可用，session-S 应该被重新绑定到 B
	resp2 := env.traceInit(t, "session-S")
	expectedB := "/traces/b_session-S.jsonl"
	if resp2.TracePath != expectedB {
		t.Fatalf("expected resume to route to proxy B (%s), got %s", expectedB, resp2.TracePath)
	}
}

// TestTraceInit_UnregisterClearsBinding 验证：注销 proxy 会清理该 proxy 下的 session 绑定。
func TestTraceInit_UnregisterClearsBinding(t *testing.T) {
	env := setupTraceInitTest(t)

	env.registerProxy(t, "pid-A", env.proxyA.URL)
	resp := env.traceInit(t, "session-S")
	if resp.TracePath != "/traces/a_session-S.jsonl" {
		t.Fatalf("expected proxy A, got %s", resp.TracePath)
	}

	// 再注册 B，但 A 仍存活；注销 A 后再发一次
	env.registerProxy(t, "pid-B", env.proxyB.URL)
	env.unregisterProxy(t, "pid-A")

	resp2 := env.traceInit(t, "session-S")
	if resp2.TracePath != "/traces/b_session-S.jsonl" {
		t.Fatalf("expected fallback to proxy B after A unregister, got %s", resp2.TracePath)
	}
}

// TestTraceInit_NoProxiesRegistered 验证：无 proxy 时返回 503。
func TestTraceInit_NoProxiesRegistered(t *testing.T) {
	env := setupTraceInitTest(t)

	resp := env.post(t, "/api/proxy/trace-init", `{"session_id":"session-S","machine_id":"u@h","project_slug":"proj"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}
}

// TestTraceInit_MissingSessionID 验证：缺少 session_id 返回 400。
func TestTraceInit_MissingSessionID(t *testing.T) {
	env := setupTraceInitTest(t)
	env.registerProxy(t, "pid-A", env.proxyA.URL)

	resp := env.post(t, "/api/proxy/trace-init", `{"machine_id":"u@h","project_slug":"proj"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
