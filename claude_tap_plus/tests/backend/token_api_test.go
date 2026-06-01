// Package backend_test 包含 GET /api/session/{id}/tokens 接口的 BDD + TDD 验收测试。
// 覆盖：获取 Token 统计、无 trace 文件零值、不存在会话 404、方法限制。
// 测试数据通过 API 注册会话 + 手动创建 JSONL trace 文件产生。
package backend_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// --- response types for token API ---

// tokenResponse 是 GET /api/session/{id}/tokens 的响应结构。
type tokenResponse struct {
	SessionID    string `json:"session_id"`
	APICalls     int    `json:"api_calls"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	CacheRead    int    `json:"cache_read"`
	CacheCreate  int    `json:"cache_create"`
	TotalTokens  int    `json:"total_tokens"`
}

// --- helpers ---

// readTokenResponse 读取并解析 Token 响应。
func readTokenResponse(t *testing.T, resp *http.Response) tokenResponse {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result tokenResponse
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("parse JSON: %v, body: %s", err, b)
	}
	return result
}

// createTraceFile 创建临时 JSONL trace 文件并返回路径。
// 每条记录包含 usage 字段的 input_tokens/output_tokens/cache_read/cache_create。
func createTraceFile(t *testing.T, records []map[string]any) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	for _, rec := range records {
		line, _ := json.Marshal(rec)
		f.Write(line)
		f.Write([]byte("\n"))
	}
	return path
}

// --- BDD Scenario tests ---

// TestSessionTokens_GetStats 验证：获取会话 Token 统计。
// BDD: @positive Scenario: 获取会话 Token 统计
//
//	Background 数据：3 条 trace 记录
//	  #1: input=1024, output=256, cache_read=0, cache_create=0
//	  #2: input=2048, output=512, cache_read=1024, cache_create=256
//	  #3: input=512,  output=128, cache_read=0,   cache_create=0
//	预期汇总：api_calls=3, input=3584, output=896, cache_read=1024, cache_create=256, total=4480
func TestSessionTokens_GetStats(t *testing.T) {
	env := setupTest(t)

	// 创建 trace 文件，包含 3 条记录
	tracePath := filepath.ToSlash(createTraceFile(t, []map[string]any{
		{"usage": map[string]any{"input_tokens": 1024, "output_tokens": 256, "cache_read": 0, "cache_create": 0}},
		{"usage": map[string]any{"input_tokens": 2048, "output_tokens": 512, "cache_read": 1024, "cache_create": 256}},
		{"usage": map[string]any{"input_tokens": 512, "output_tokens": 128, "cache_read": 0, "cache_create": 0}},
	}))

	// 注册会话，指定 local_trace_path（使用正斜杠保证 JSON 转义正确）
	body := fmt.Sprintf(
		`{"session_id":"session-abc123","machine_id":"user@host","os":"linux","project_slug":"proj","project_cwd":"/tmp","transcript_path":"/tmp/t.jsonl","local_trace_path":"%s"}`,
		tracePath)
	resp := env.post(t, "/api/session/register", body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register session: expected 200, got %d", resp.StatusCode)
	}

	resp = env.get(t, "/api/session/session-abc123/tokens")
	result := readTokenResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if result.SessionID != "session-abc123" {
		t.Errorf("expected session_id=session-abc123, got %s", result.SessionID)
	}
	if result.APICalls != 3 {
		t.Errorf("expected api_calls=3, got %d", result.APICalls)
	}
	if result.InputTokens != 3584 {
		t.Errorf("expected input_tokens=3584, got %d", result.InputTokens)
	}
	if result.OutputTokens != 896 {
		t.Errorf("expected output_tokens=896, got %d", result.OutputTokens)
	}
	if result.CacheRead != 1024 {
		t.Errorf("expected cache_read=1024, got %d", result.CacheRead)
	}
	if result.CacheCreate != 256 {
		t.Errorf("expected cache_create=256, got %d", result.CacheCreate)
	}
	if result.TotalTokens != 4480 {
		t.Errorf("expected total_tokens=4480, got %d", result.TotalTokens)
	}
}

// TestSessionTokens_NoTraceFile 验证：会话无 Token 记录返回零值。
// BDD: @positive Scenario: 会话无 Token 记录
func TestSessionTokens_NoTraceFile(t *testing.T) {
	env := setupTest(t)

	// 注册会话，不指定 local_trace_path
	env.post(t, "/api/session/register",
		`{"session_id":"session-empty","machine_id":"user@host","os":"linux","project_slug":"proj","project_cwd":"/tmp","transcript_path":"/tmp/t.jsonl"}`)

	resp := env.get(t, "/api/session/session-empty/tokens")
	result := readTokenResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if result.SessionID != "session-empty" {
		t.Errorf("expected session_id=session-empty, got %s", result.SessionID)
	}
	if result.APICalls != 0 {
		t.Errorf("expected api_calls=0, got %d", result.APICalls)
	}
	if result.InputTokens != 0 {
		t.Errorf("expected input_tokens=0, got %d", result.InputTokens)
	}
	if result.OutputTokens != 0 {
		t.Errorf("expected output_tokens=0, got %d", result.OutputTokens)
	}
	if result.CacheRead != 0 {
		t.Errorf("expected cache_read=0, got %d", result.CacheRead)
	}
	if result.CacheCreate != 0 {
		t.Errorf("expected cache_create=0, got %d", result.CacheCreate)
	}
	if result.TotalTokens != 0 {
		t.Errorf("expected total_tokens=0, got %d", result.TotalTokens)
	}
}

// TestSessionTokens_SessionNotFound 验证：获取不存在的会话返回 404。
// BDD: @negative Scenario: 获取不存在的会话
func TestSessionTokens_SessionNotFound(t *testing.T) {
	env := setupTest(t)

	resp := env.get(t, "/api/session/session-not-exist/tokens")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var errResp struct {
		Code    string `json:"error"`
		Message string `json:"message"`
	}
	readJSON(t, resp, &errResp)
	if errResp.Code != "not_found" {
		t.Errorf("expected error=not_found, got %s", errResp.Code)
	}
}

// TestSessionTokens_MethodNotAllowed 验证：使用非 GET 方法请求返回 405。
// BDD: @negative Scenario: 使用非 GET 方法请求
func TestSessionTokens_MethodNotAllowed(t *testing.T) {
	env := setupTest(t)

	// 先注册会话
	env.post(t, "/api/session/register",
		`{"session_id":"session-abc123","machine_id":"user@host","os":"linux","project_slug":"proj","project_cwd":"/tmp","transcript_path":"/tmp/t.jsonl"}`)

	resp := env.post(t, "/api/session/session-abc123/tokens", "")
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
