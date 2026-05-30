// Package e2e_test 包含代理层的端到端测试，覆盖请求转发、流式响应、追踪记录等完整链路。
package e2e_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/proxy"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/testutil"
)

// setupProxyWithTrace 创建带追踪初始化的代理，返回代理 URL、追踪文件路径。
// 通过调用内部 /_internal/trace-init 端点初始化 session 追踪。
func setupProxyWithTrace(t *testing.T, mockServerURL string) (string, string) {
	t.Helper()

	traceDir := t.TempDir()
	rp := proxy.NewReverseProxy(mockServerURL, traceDir)
	port, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	t.Cleanup(func() { rp.Stop() })

	proxyURL := "http://127.0.0.1:" + fmt.Sprint(port)

	// 通过内部端点初始化追踪
	sessionID := "test-session-001"
	machineID := "testuser@testhost"
	projectSlug := "test-project"
	initBody := fmt.Sprintf(
		`{"session_id":"%s","machine_id":"%s","project_slug":"%s"}`,
		sessionID, machineID, projectSlug,
	)
	resp, err := http.Post(proxyURL+"/_internal/trace-init", "application/json", strings.NewReader(initBody))
	if err != nil {
		t.Fatalf("trace-init: %v", err)
	}
	defer resp.Body.Close()

	var initResult struct {
		Status    string `json:"status"`
		TracePath string `json:"trace_path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&initResult); err != nil {
		t.Fatalf("parse trace-init response: %v", err)
	}
	if initResult.Status != "ok" {
		t.Fatalf("trace-init failed: %s", initResult.Status)
	}

	tracePath := initResult.TracePath

	return proxyURL, tracePath
}

// TestNonStreamingRequest 验证：非流式请求正常转发，响应正确，且追踪记录包含脱敏后的请求头。
func TestNonStreamingRequest(t *testing.T) {
	fx := testutil.LoadFixture(t, "nonstreaming_anthropic.json")
	mockServer := testutil.CreateMockServer(t, fx)
	defer mockServer.Close()

	proxyURL, tracePath := setupProxyWithTrace(t, mockServer.URL)

	resp := testutil.SendRequestThroughProxy(t, fx, proxyURL)
	testutil.AssertResponseStatus(t, resp, 200)

	body := testutil.DrainAndClose(t, resp)
	var respBody map[string]any
	if err := json.Unmarshal(body, &respBody); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if respBody["type"] != "message" {
		t.Errorf("response type: got %v, want message", respBody["type"])
	}
	if respBody["stop_reason"] != "tool_use" {
		t.Errorf("stop_reason: got %v, want tool_use", respBody["stop_reason"])
	}

	// 验证追踪记录
	records := testutil.ReadTraceRecords(t, tracePath)
	if len(records) != 1 {
		t.Fatalf("expected 1 trace record, got %d", len(records))
	}
	testutil.AssertTrace(t, records[0], fx.ExpectedTrace)

	// 验证请求头中的敏感信息已被脱敏
	reqHeaders, _ := records[0]["request"].(map[string]any)["headers"].(map[string]any)
	if auth, ok := reqHeaders["Authorization"].(string); ok {
		if auth == "Bearer sk-ant-test-key-12345678" {
			t.Error("Authorization header not redacted in trace")
		}
	}
	if apiKey, ok := reqHeaders["X-Api-Key"].(string); ok {
		if apiKey == "sk-ant-test-key-12345678" {
			t.Error("X-Api-Key not redacted in trace")
		}
	}
}

// TestStreamingSSE 验证：SSE 流式请求正常转发，响应包含所有事件，且追踪记录正确。
func TestStreamingSSE(t *testing.T) {
	fx := testutil.LoadFixture(t, "streaming_anthropic.json")
	mockServer := testutil.CreateMockServer(t, fx)
	defer mockServer.Close()

	proxyURL, tracePath := setupProxyWithTrace(t, mockServer.URL)

	resp := testutil.SendRequestThroughProxy(t, fx, proxyURL)
	testutil.AssertResponseStatus(t, resp, 200)

	body := testutil.DrainAndClose(t, resp)
	bodyStr := string(body)

	// 验证响应中包含所有 SSE 事件
	if !strings.Contains(bodyStr, "message_start") {
		t.Error("response missing message_start event")
	}
	if !strings.Contains(bodyStr, "content_block_delta") {
		t.Error("response missing content_block_delta event")
	}
	if !strings.Contains(bodyStr, "message_stop") {
		t.Error("response missing message_stop event")
	}

	// 验证追踪记录
	records := testutil.ReadTraceRecords(t, tracePath)
	if len(records) != 1 {
		t.Fatalf("expected 1 trace record, got %d", len(records))
	}
	testutil.AssertTrace(t, records[0], fx.ExpectedTrace)

	respMap, _ := records[0]["response"].(map[string]any)
	if sseEvents, ok := respMap["sse_events"].([]any); ok {
		if len(sseEvents) == 0 {
			t.Error("trace has no sse_events for streaming response")
		}
	}
}

// TestMultiTurn 验证：多轮对话请求会在追踪文件中生成多条记录，且 turn 编号递增。
func TestMultiTurn(t *testing.T) {
	fx := testutil.LoadFixture(t, "nonstreaming_anthropic.json")
	mockServer := testutil.CreateMockServer(t, fx)
	defer mockServer.Close()

	proxyURL, tracePath := setupProxyWithTrace(t, mockServer.URL)

	// 发送 3 轮请求
	for i := 0; i < 3; i++ {
		resp := testutil.SendRequestThroughProxy(t, fx, proxyURL)
		testutil.AssertResponseStatus(t, resp, 200)
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}

	records := testutil.ReadTraceRecords(t, tracePath)
	if len(records) != 3 {
		t.Fatalf("expected 3 trace records, got %d", len(records))
	}

	// 验证每轮记录的 turn 编号和 session_id
	for i, rec := range records {
		turn, _ := rec["turn"].(float64)
		if int(turn) != i+1 {
			t.Errorf("record %d: turn = %v, want %d", i, turn, i+1)
		}
		sid, _ := rec["session_id"].(string)
		if sid != "test-session-001" {
			t.Errorf("record %d: session_id = %q, want test-session-001", i, sid)
		}
	}
}

// TestSessionIDExtraction 验证：追踪记录中正确提取了 session_id。
func TestSessionIDExtraction(t *testing.T) {
	fx := testutil.LoadFixture(t, "nonstreaming_anthropic.json")
	mockServer := testutil.CreateMockServer(t, fx)
	defer mockServer.Close()

	proxyURL, tracePath := setupProxyWithTrace(t, mockServer.URL)

	resp := testutil.SendRequestThroughProxy(t, fx, proxyURL)
	io.ReadAll(resp.Body)
	resp.Body.Close()

	records := testutil.ReadTraceRecords(t, tracePath)
	if len(records) == 0 {
		t.Fatal("no trace records")
	}

	sid, _ := records[0]["session_id"].(string)
	if sid != "test-session-001" {
		t.Errorf("session_id: got %q, want %q", sid, "test-session-001")
	}
}

// TestTokenStats 验证：代理正确统计 API 调用次数和输入/输出 token 数量。
func TestTokenStats(t *testing.T) {
	fx := testutil.LoadFixture(t, "nonstreaming_anthropic.json")
	mockServer := testutil.CreateMockServer(t, fx)
	defer mockServer.Close()

	traceDir := t.TempDir()
	rp := proxy.NewReverseProxy(mockServer.URL, traceDir)
	port, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	proxyURL := "http://127.0.0.1:" + fmt.Sprint(port)
	resp := testutil.SendRequestThroughProxy(t, fx, proxyURL)
	io.ReadAll(resp.Body)
	resp.Body.Close()

	summary := rp.Summary()

	// 验证 API 调用次数
	apiCalls, _ := summary["api_calls"].(int)
	if apiCalls != 1 {
		t.Errorf("api_calls: got %d, want 1", apiCalls)
	}

	// 验证输入 token 数
	inputTokens, _ := summary["input_tokens"].(int64)
	if inputTokens != 1234 {
		t.Errorf("input_tokens: got %d, want 1234", inputTokens)
	}

	// 验证输出 token 数
	outputTokens, _ := summary["output_tokens"].(int64)
	if outputTokens != 56 {
		t.Errorf("output_tokens: got %d, want 56", outputTokens)
	}
}

// TestHeaderRedaction 验证：追踪记录中的敏感请求头（Authorization、X-Api-Key）已被脱敏。
func TestHeaderRedaction(t *testing.T) {
	fx := testutil.LoadFixture(t, "nonstreaming_anthropic.json")
	mockServer := testutil.CreateMockServer(t, fx)
	defer mockServer.Close()

	proxyURL, tracePath := setupProxyWithTrace(t, mockServer.URL)

	resp := testutil.SendRequestThroughProxy(t, fx, proxyURL)
	io.ReadAll(resp.Body)
	resp.Body.Close()

	records := testutil.ReadTraceRecords(t, tracePath)
	if len(records) == 0 {
		t.Fatal("no trace records")
	}

	reqMap, _ := records[0]["request"].(map[string]any)
	headers, _ := reqMap["headers"].(map[string]any)

	// 验证 X-Api-Key 被脱敏
	if apiKey, ok := headers["X-Api-Key"].(string); ok {
		if apiKey == "sk-ant-test-key-12345678" {
			t.Error("X-Api-Key not redacted in trace")
		}
	}
	// 验证 Authorization 被脱敏
	if auth, ok := headers["Authorization"].(string); ok {
		if auth == "Bearer sk-ant-test-key-12345678" {
			t.Error("Authorization not redacted in trace")
		}
	}
}

// TestBlockedPath 验证：被屏蔽的路径（如 /admin/config）返回 404，不会转发到上游。
func TestBlockedPath(t *testing.T) {
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("should not reach"))
	})
	mockServer := httptest.NewServer(mockHandler)
	defer mockServer.Close()

	traceDir := t.TempDir()
	rp := proxy.NewReverseProxy(mockServer.URL, traceDir)
	port, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	resp, err := http.Get("http://127.0.0.1:" + fmt.Sprint(port) + "/admin/config")
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("blocked path status: got %d, want 404", resp.StatusCode)
	}
}

// TestGzipResponse 验证：代理能正确处理 gzip 压缩的响应并解压。
func TestGzipResponse(t *testing.T) {
	fx := testutil.LoadFixture(t, "gzip_response.json")
	mockServer := testutil.CreateMockServer(t, fx)
	defer mockServer.Close()

	proxyURL, tracePath := setupProxyWithTrace(t, mockServer.URL)

	resp := testutil.SendRequestThroughProxy(t, fx, proxyURL)
	testutil.AssertResponseStatus(t, resp, 200)

	body := testutil.DrainAndClose(t, resp)
	var respBody map[string]any
	if err := json.Unmarshal(body, &respBody); err != nil {
		t.Fatalf("parse response (should be decompressed): %v", err)
	}
	if respBody["type"] != "message" {
		t.Errorf("response type: got %v, want message", respBody["type"])
	}

	records := testutil.ReadTraceRecords(t, tracePath)
	if len(records) == 0 {
		t.Fatal("no trace records")
	}
	testutil.AssertTrace(t, records[0], fx.ExpectedTrace)
}

// TestTraceInitEndpoint 验证：/_internal/trace-init 端点能正确初始化追踪文件并返回路径。
func TestTraceInitEndpoint(t *testing.T) {
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"id":"msg_test"}`))
	})
	mockServer := httptest.NewServer(mockHandler)
	defer mockServer.Close()

	traceDir := t.TempDir()
	rp := proxy.NewReverseProxy(mockServer.URL, traceDir)
	port, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	proxyURL := "http://127.0.0.1:" + fmt.Sprint(port)

	// 调用 trace-init
	body := `{"session_id":"sess-abc","machine_id":"user@host","project_slug":"my-project"}`
	resp, err := http.Post(proxyURL+"/_internal/trace-init", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("trace-init: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Status    string `json:"status"`
		TracePath string `json:"trace_path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if result.Status != "ok" {
		t.Errorf("status: got %q, want ok", result.Status)
	}

	// 验证追踪文件路径以 session_id 结尾
	if !strings.HasSuffix(filepath.ToSlash(result.TracePath), "sess-abc.jsonl") {
		t.Errorf("trace_path suffix: got %q, want ending with sess-abc.jsonl", result.TracePath)
	}

	// 验证追踪文件已被创建
	if _, err := os.Stat(result.TracePath); os.IsNotExist(err) {
		t.Errorf("trace file not created: %s", result.TracePath)
	}

	// 验证目录结构：{machine_id}/{project_slug}/{date}/{time}/{session_id}.jsonl
	rel, _ := filepath.Rel(traceDir, result.TracePath)
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) != 5 {
		t.Errorf("expected 5 path components (machine/project/date/time/file), got %d: %v", len(parts), parts)
	}
	if parts[0] != "user@host" {
		t.Errorf("machine_id dir: got %q, want user@host", parts[0])
	}
	if parts[1] != "my-project" {
		t.Errorf("project_slug dir: got %q, want my-project", parts[1])
	}
}

// TestTraceInitMissingSessionID 验证：trace-init 请求缺少 session_id 时返回 400。
func TestTraceInitMissingSessionID(t *testing.T) {
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	mockServer := httptest.NewServer(mockHandler)
	defer mockServer.Close()

	traceDir := t.TempDir()
	rp := proxy.NewReverseProxy(mockServer.URL, traceDir)
	port, err := rp.Start("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer rp.Stop()

	proxyURL := "http://127.0.0.1:" + fmt.Sprint(port)

	resp, err := http.Post(proxyURL+"/_internal/trace-init", "application/json", strings.NewReader(`{"session_id":""}`))
	if err != nil {
		t.Fatalf("trace-init: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}
