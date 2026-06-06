package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
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
	// 设置 fallback 配置（RED 阶段：此字段尚不存在）
	rp.fallbackConfig = &FallbackConfig{
		BaseURL:   fallbackServer.URL,
		Model:     "claude-sonnet-4-6",
		AuthToken: "tok-fallback",
	}

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
	rp.fallbackConfig = &FallbackConfig{
		BaseURL:   fallbackServer.URL,
		Model:     "claude-sonnet-4-6",
		AuthToken: "tok-fallback",
	}

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
	rp.fallbackConfig = &FallbackConfig{
		BaseURL:   fallbackServer.URL,
		Model:     "claude-sonnet-4-6",
		AuthToken: "tok-fallback",
	}

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
