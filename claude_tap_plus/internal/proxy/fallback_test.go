package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestUpstreamFallback_ConnectionFailure 测试连接失败时触发兜底
// 来源：契约 4 — 上游可用性检测与全量兜底
func TestUpstreamFallback_ConnectionFailure(t *testing.T) {
	// 创建 fallback 上游服务器
	var fallbackMu sync.Mutex
	var fallbackRequests []map[string]any
	fallbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]any
		json.Unmarshal(body, &parsed)

		fallbackMu.Lock()
		fallbackRequests = append(fallbackRequests, parsed)
		fallbackMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_fallback","type":"message","role":"assistant","content":[],"model":"fallback-model","stop_reason":"end_turn","usage":{"input_tokens":0,"output_tokens":0}}`))
	}))
	defer fallbackServer.Close()

	// 创建一个会立即关闭的服务器模拟连接失败
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 使用 503 表示上游不可用
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"service unavailable"}`))
	}))
	defer badServer.Close()

	traceDir := t.TempDir()
	rp := NewReverseProxy(badServer.URL, traceDir)
	rp.model = "glm-5.1"
	rp.retryInterval = 50 * time.Millisecond
	rp.SetFallbackConfigs([]*FallbackConfig{
		{
			BaseURL:   fallbackServer.URL,
			Model:     "claude-sonnet-4-6",
			AuthToken: "tok-fallback",
		},
	})

	_, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	// 请求 1：上游返回 503，应触发 fallback
	body1 := `{"model":"claude-sonnet-4-6","stream":false}`
	req1, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader([]byte(body1)))
	req1.Header.Set("Content-Type", "application/json")
	resp1, err := http.DefaultClient.Do(req1)
	if err != nil {
		t.Fatalf("request 1: %v", err)
	}
	resp1.Body.Close()

	// 请求 2：应走 fallback
	body2 := `{"model":"claude-sonnet-4-6","stream":false}`
	req2, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader([]byte(body2)))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("request 2: %v", err)
	}
	resp2.Body.Close()

	// 验证 fallback 服务器收到了请求
	fallbackMu.Lock()
	defer fallbackMu.Unlock()
	if len(fallbackRequests) == 0 {
		t.Fatal("fallback server received no requests")
	}

	// 验证 fallback 请求的 model 被替换为 fallback model
	lastReq := fallbackRequests[len(fallbackRequests)-1]
	gotModel, _ := lastReq["model"].(string)
	if gotModel != "claude-sonnet-4-6" {
		t.Errorf("fallback model = %q, want %q", gotModel, "claude-sonnet-4-6")
	}

}

// TestUpstreamFallback_ResponseError 测试上游响应错误时触发兜底
// 来源：契约 4 — 响应错误（4xx/5xx）触发切换
func TestUpstreamFallback_ResponseError(t *testing.T) {
	var primaryMu sync.Mutex
	primaryRequestCount := 0

	// 上游返回 500
	primaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryMu.Lock()
		primaryRequestCount++
		primaryMu.Unlock()
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer primaryServer.Close()

	// fallback 服务器
	fallbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_fallback","type":"message","role":"assistant","content":[],"model":"fallback-model","stop_reason":"end_turn","usage":{"input_tokens":0,"output_tokens":0}}`))
	}))
	defer fallbackServer.Close()

	traceDir := t.TempDir()
	rp := NewReverseProxy(primaryServer.URL, traceDir)
	rp.model = "glm-5.1"
	rp.retryInterval = 50 * time.Millisecond
	rp.SetFallbackConfigs([]*FallbackConfig{
		{
			BaseURL:   fallbackServer.URL,
			Model:     "claude-sonnet-4-6",
			AuthToken: "tok-fallback",
		},
	})

	_, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	// 请求 1：上游 500
	req1, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader([]byte(`{"model":"glm-5.1","stream":false}`)))
	req1.Header.Set("Content-Type", "application/json")
	resp1, _ := http.DefaultClient.Do(req1)
	resp1.Body.Close()

	// 请求 2：应走 fallback
	req2, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader([]byte(`{"model":"glm-5.1","stream":false}`)))
	req2.Header.Set("Content-Type", "application/json")
	resp2, _ := http.DefaultClient.Do(req2)
	resp2.Body.Close()

	// 上游应该只收到 1 个请求（第一次失败后就不再请求了）
	primaryMu.Lock()
	defer primaryMu.Unlock()
	if primaryRequestCount != 1 {
		t.Errorf("primary received %d requests, want 1", primaryRequestCount)
	}

}

// TestUpstreamFallback_NoRecovery 测试切换后不恢复
// 来源：契约 4 — 进程生命周期内保持不可用
func TestUpstreamFallback_NoRecovery(t *testing.T) {
	// 上游：第一次失败，后续都成功（模拟恢复）
	requestCount := 0
	primaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// 后续请求返回成功（但 fallback 不应该再来这里了）
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_ok","type":"message","role":"assistant","content":[],"model":"primary","stop_reason":"end_turn","usage":{"input_tokens":0,"output_tokens":0}}`))
	}))
	defer primaryServer.Close()

	// fallback
	fallbackRequestCount := 0
	fallbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackRequestCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_fallback","type":"message","role":"assistant","content":[],"model":"fallback","stop_reason":"end_turn","usage":{"input_tokens":0,"output_tokens":0}}`))
	}))
	defer fallbackServer.Close()

	traceDir := t.TempDir()
	rp := NewReverseProxy(primaryServer.URL, traceDir)
	rp.model = "glm-5.1"
	rp.retryInterval = 50 * time.Millisecond
	rp.SetFallbackConfigs([]*FallbackConfig{
		{
			BaseURL:   fallbackServer.URL,
			Model:     "claude-sonnet-4-6",
			AuthToken: "tok-fallback",
		},
	})

	_, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	// 发送 3 个请求
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader([]byte(`{"model":"glm-5.1","stream":false}`)))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := http.DefaultClient.Do(req)
		resp.Body.Close()
	}

	// 上游只收到 1 个请求（第一次失败），后续不会重试
	if requestCount != 1 {
		t.Errorf("primary received %d requests, want 1 (no recovery)", requestCount)
	}
	// fallback 收到 2 个请求（第 2、3 个请求）
	if fallbackRequestCount != 2 {
		t.Errorf("fallback received %d requests, want 2", fallbackRequestCount)
	}
}

// TestUpstreamFallback_NoFallbackConfig 测试没有 fallback 配置时不崩溃
// 来源：契约 4 — fallback 为空时的行为
func TestUpstreamFallback_NoFallbackConfig(t *testing.T) {
	primaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal error"}`))
	}))
	defer primaryServer.Close()

	traceDir := t.TempDir()
	rp := NewReverseProxy(primaryServer.URL, traceDir)
	rp.model = "glm-5.1"
	rp.retryInterval = 50 * time.Millisecond
	// 不设置 fallbackConfig

	_, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	// 发送请求，不应 panic
	req, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader([]byte(`{"model":"glm-5.1","stream":false}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	resp.Body.Close()

	// 即使上游失败且无 fallback，代理透传上游的 500 响应
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d (upstream error forwarded)", resp.StatusCode, http.StatusInternalServerError)
	}
}

// TestUpstreamFallback_MultipleProfiles 测试多 fallback profile 轮询
func TestUpstreamFallback_MultipleProfiles(t *testing.T) {
	var mu sync.Mutex
	fb1Count := 0
	fb2Count := 0

	// fallback 1：总是失败
	fb1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		fb1Count++
		mu.Unlock()
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"fb1 down"}`))
	}))
	defer fb1.Close()

	// fallback 2：总是成功
	fb2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		fb2Count++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_fb2","type":"message","role":"assistant","content":[],"model":"fb2","stop_reason":"end_turn","usage":{"input_tokens":0,"output_tokens":0}}`))
	}))
	defer fb2.Close()

	// 主上游失败
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer primary.Close()

	traceDir := t.TempDir()
	rp := NewReverseProxy(primary.URL, traceDir)
	rp.SetFallbackConfigs([]*FallbackConfig{
		{BaseURL: fb1.URL, Model: "fb1"},
		{BaseURL: fb2.URL, Model: "fb2"},
	})

	_, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	// 请求 1：主上游 500 → 标记 unavailable，返回 500
	req1, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader([]byte(`{"model":"test","stream":false}`)))
	req1.Header.Set("Content-Type", "application/json")
	resp1, _ := http.DefaultClient.Do(req1)
	resp1.Body.Close()

	// 请求 2：此时 unavailable，走 fallback[0]（fb1）也 500 → advance
	req2, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader([]byte(`{"model":"test","stream":false}`)))
	req2.Header.Set("Content-Type", "application/json")
	resp2, _ := http.DefaultClient.Do(req2)
	resp2.Body.Close()

	// 请求 3：fallback index 已 advance，走 fallback[1]（fb2）成功
	req3, _ := http.NewRequest("POST", rp.URL()+"/v1/messages", bytes.NewReader([]byte(`{"model":"test","stream":false}`)))
	req3.Header.Set("Content-Type", "application/json")
	resp3, _ := http.DefaultClient.Do(req3)
	resp3.Body.Close()

	mu.Lock()
	if fb1Count == 0 {
		t.Error("fallback 1 should have been tried")
	}
	if fb2Count == 0 {
		t.Error("fallback 2 should have been tried after fallback 1 failed")
	}
	mu.Unlock()
}

// TestDoWithRetry_ConnectionError 测试连接错误时重试最终失败
func TestDoWithRetry_ConnectionError(t *testing.T) {
	traceDir := t.TempDir()
	rp := NewReverseProxy("http://127.0.0.1:1", traceDir)
	rp.retryInterval = 50 * time.Millisecond

	req, _ := http.NewRequest("POST", "http://127.0.0.1:1/v1/messages", bytes.NewReader([]byte(`{"model":"test"}`)))
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	_, err := rp.doWithRetry(req, 1, false)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error after retries")
	}

	// 应重试 5 次，间隔约 50ms*5 = 250ms
	if elapsed < 200*time.Millisecond {
		t.Errorf("retries too fast: %v, expected at least 200ms", elapsed)
	}
}

// TestDoWithRetry_5xxRetryThenSuccess 测试 5xx 重试后成功
func TestDoWithRetry_5xxRetryThenSuccess(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"temporarily unavailable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"ok","model":"test"}`))
	}))
	defer server.Close()

	traceDir := t.TempDir()
	rp := NewReverseProxy(server.URL, traceDir)
	rp.retryInterval = 10 * time.Millisecond

	req, _ := http.NewRequest("POST", server.URL+"/v1/messages", bytes.NewReader([]byte(`{"model":"test"}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := rp.doWithRetry(req, 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if callCount != 3 {
		t.Errorf("expected 3 calls (2 retries + success), got %d", callCount)
	}
}

// TestDoWithRetry_4xxNoRetry 测试 4xx 不重试
func TestDoWithRetry_4xxNoRetry(t *testing.T) {
	callCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer server.Close()

	traceDir := t.TempDir()
	rp := NewReverseProxy(server.URL, traceDir)

	req, _ := http.NewRequest("POST", server.URL+"/v1/messages", bytes.NewReader([]byte(`{"model":"test"}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := rp.doWithRetry(req, 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call (no retry for 4xx), got %d", callCount)
	}
}
