// Package backend_test 包含 GET /api/session/{id}/traces 接口的 BDD + TDD 验收测试。
// 覆盖：获取 Trace 文件列表（含元数据）、无 trace 文件返回空、不存在会话 404、方法限制。
// 测试数据通过 API 注册会话（指定 local_trace_path）+ 手动创建 JSONL 文件产生。
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

// --- response types for trace API ---

// traceListResponse 是 GET /api/session/{id}/traces 的响应结构。
type traceListResponse struct {
	SessionID string      `json:"session_id"`
	Traces    []traceItem `json:"traces"`
}

// traceItem 是单个 Trace 文件信息。
type traceItem struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	LineCount int    `json:"line_count"`
	Date      string `json:"date"`
	Filename  string `json:"filename"`
}

// --- helpers ---

// readTraceListResponse 读取并解析 Trace 列表响应。
func readTraceListResponse(t *testing.T, resp *http.Response) traceListResponse {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result traceListResponse
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("parse JSON: %v, body: %s", err, b)
	}
	return result
}

// createTraceFileWithLines 创建指定行数的 JSONL trace 文件并返回路径。
func createTraceFileWithLines(t *testing.T, lineCount int) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "trace_143052.jsonl")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	for i := 0; i < lineCount; i++ {
		f.WriteString(`{"usage":{"input_tokens":100,"output_tokens":50}}`)
		f.WriteString("\n")
	}
	return path
}

// --- BDD Scenario tests ---

// TestSessionTraces_GetTraces 验证：获取会话 Trace 文件列表。
// BDD: @positive Scenario: 获取会话 Trace 文件列表
func TestSessionTraces_GetTraces(t *testing.T) {
	env := setupTest(t)

	// 创建 trace 文件：2.3MB 近似值，156 行
	tracePath := createTraceFileWithLines(t, 156)
	tracePathFwd := filepath.ToSlash(tracePath)

	// 计算期望的文件大小
	stat, _ := os.Stat(tracePath)
	expectedSize := stat.Size()

	// 从路径提取 date 和 filename
	expectedDir := filepath.Base(filepath.Dir(tracePath))
	expectedFilename := filepath.Base(tracePath)

	// 注册会话，指定 local_trace_path
	body := fmt.Sprintf(
		`{"session_id":"session-abc123","machine_id":"user@host","os":"linux","project_slug":"proj","project_cwd":"/tmp","transcript_path":"/tmp/t.jsonl","local_trace_path":"%s"}`,
		tracePathFwd)
	resp := env.post(t, "/api/session/register", body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register: expected 200, got %d", resp.StatusCode)
	}

	resp = env.get(t, "/api/session/session-abc123/traces")
	result := readTraceListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if result.SessionID != "session-abc123" {
		t.Errorf("expected session_id=session-abc123, got %s", result.SessionID)
	}
	if len(result.Traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(result.Traces))
	}

	trace := result.Traces[0]
	if trace.Filename != expectedFilename {
		t.Errorf("expected filename=%s, got %s", expectedFilename, trace.Filename)
	}
	if trace.LineCount != 156 {
		t.Errorf("expected line_count=156, got %d", trace.LineCount)
	}
	if trace.SizeBytes != expectedSize {
		t.Errorf("expected size_bytes=%d, got %d", expectedSize, trace.SizeBytes)
	}
	if trace.Date != expectedDir {
		t.Errorf("expected date=%s, got %s", expectedDir, trace.Date)
	}
	if trace.Path == "" {
		t.Error("expected non-empty path")
	}
}

// TestSessionTraces_NoTraceFile 验证：会话无 Trace 文件返回空数组。
// BDD: @positive Scenario: 会话无 Trace 文件
func TestSessionTraces_NoTraceFile(t *testing.T) {
	env := setupTest(t)

	// 注册会话，不指定 local_trace_path
	resp := env.post(t, "/api/session/register",
		`{"session_id":"session-no-trace","machine_id":"user@host","os":"linux","project_slug":"proj","project_cwd":"/tmp","transcript_path":"/tmp/t.jsonl"}`)
	resp.Body.Close()

	resp = env.get(t, "/api/session/session-no-trace/traces")
	result := readTraceListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Traces) != 0 {
		t.Fatalf("expected empty traces array, got %d items", len(result.Traces))
	}
}

// TestSessionTraces_SessionNotFound 验证：获取不存在的会话返回 404。
// BDD: @negative Scenario: 获取不存在的会话
func TestSessionTraces_SessionNotFound(t *testing.T) {
	env := setupTest(t)

	resp := env.get(t, "/api/session/session-not-exist/traces")
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

// TestSessionTraces_MethodNotAllowed 验证：使用非 GET 方法请求返回 405。
// BDD: @negative Scenario: 使用非 GET 方法请求
func TestSessionTraces_MethodNotAllowed(t *testing.T) {
	env := setupTest(t)

	// 注册会话
	resp := env.post(t, "/api/session/register",
		`{"session_id":"session-abc123","machine_id":"user@host","os":"linux","project_slug":"proj","project_cwd":"/tmp","transcript_path":"/tmp/t.jsonl"}`)
	resp.Body.Close()

	resp = env.post(t, "/api/session/session-abc123/traces", "")
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
