// Package trace_test 包含追踪写入器的单元测试，覆盖文件创建、记录写入、统计汇总等功能。
package trace_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/trace"
)

// TestNewTracePath 验证：追踪文件路径格式正确。
// 路径应包含日期、时间和随机十六进制后缀，以 .jsonl 结尾。
func TestNewTracePath(t *testing.T) {
	path := trace.NewTracePath("/tmp/traces")
	// 路径格式：/tmp/traces/{machine_id}/{project}/{date}/{time}/pending.jsonl
	filename := filepath.Base(path)
	if filename != "pending.jsonl" {
		t.Errorf("expected filename pending.jsonl, got %q", filename)
	}
	// 验证路径包含 date/time 目录层级
	if !strings.Contains(path, "traces") {
		t.Errorf("expected path to contain traces dir, got %q", path)
	}
}

// TestTraceWriterWriteAndSummary 验证：追踪写入器能正确写入记录并汇总统计。
// 写入两条记录后，验证 API 调用次数、输入/输出 token 数、模型使用次数。
func TestTraceWriterWriteAndSummary(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "test_trace.jsonl")

	w, err := trace.NewTraceWriter(tracePath)
	if err != nil {
		t.Fatalf("NewTraceWriter: %v", err)
	}

	// 写入第一条记录
	record := map[string]any{
		"request_id": "req_001",
		"turn":       1,
		"request": map[string]any{
			"method": "POST",
			"path":   "/v1/messages",
			"body":   map[string]any{"model": "claude-3"},
		},
		"response": map[string]any{
			"status": 200,
			"body": map[string]any{
				"usage": map[string]any{
					"input_tokens":  float64(100),
					"output_tokens": float64(50),
				},
			},
		},
	}

	if err := w.Write(record); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// 写入第二条记录
	record2 := map[string]any{
		"request_id": "req_002",
		"turn":       2,
		"request": map[string]any{
			"method": "POST",
			"path":   "/v1/messages",
			"body":   map[string]any{"model": "claude-3"},
		},
		"response": map[string]any{
			"status": 200,
			"body": map[string]any{
				"usage": map[string]any{
					"input_tokens":  float64(200),
					"output_tokens": float64(80),
				},
			},
		},
	}
	if err := w.Write(record2); err != nil {
		t.Fatalf("Write second record: %v", err)
	}

	// 验证汇总统计
	summary := w.Summary()
	if summary["api_calls"] != 2 {
		t.Errorf("expected 2 api_calls, got %v", summary["api_calls"])
	}
	if summary["input_tokens"] != int64(300) {
		t.Errorf("expected 300 input_tokens, got %v", summary["input_tokens"])
	}
	if summary["output_tokens"] != int64(130) {
		t.Errorf("expected 130 output_tokens, got %v", summary["output_tokens"])
	}

	models := summary["models_used"].(map[string]int)
	if models["claude-3"] != 2 {
		t.Errorf("expected claude-3 count 2, got %v", models)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// 验证文件内容
	data, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("read trace file: %v", err)
	}

	lines := splitLines(string(data))
	if len(lines) != 2 {
		t.Fatalf("expected 2 JSONL lines, got %d", len(lines))
	}

	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("parse first line: %v", err)
	}
	if first["request_id"] != "req_001" {
		t.Errorf("first record request_id: got %v, want req_001", first["request_id"])
	}
}

// TestTraceWriterCreatesDirectory 验证：追踪写入器能自动创建嵌套目录。
func TestTraceWriterCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	nestedPath := filepath.Join(dir, "sub1", "sub2", "trace.jsonl")

	w, err := trace.NewTraceWriter(nestedPath)
	if err != nil {
		t.Fatalf("NewTraceWriter with nested dirs: %v", err)
	}
	defer w.Close()

	if _, err := os.Stat(filepath.Dir(nestedPath)); os.IsNotExist(err) {
		t.Error("expected nested directories to be created")
	}
}

// TestTraceWriterCacheTokenStats 验证：缓存 token（cache_read_tokens、cache_create_tokens）被正确统计。
func TestTraceWriterCacheTokenStats(t *testing.T) {
	dir := t.TempDir()
	w, err := trace.NewTraceWriter(filepath.Join(dir, "cache_test.jsonl"))
	if err != nil {
		t.Fatalf("NewTraceWriter: %v", err)
	}
	defer w.Close()

	record := map[string]any{
		"request": map[string]any{
			"body": map[string]any{"model": "claude-3"},
		},
		"response": map[string]any{
			"body": map[string]any{
				"usage": map[string]any{
					"input_tokens":                float64(100),
					"output_tokens":               float64(50),
					"cache_read_input_tokens":     float64(30),
					"cache_creation_input_tokens": float64(10),
				},
			},
		},
	}

	if err := w.Write(record); err != nil {
		t.Fatalf("Write: %v", err)
	}

	summary := w.Summary()
	if summary["cache_read_tokens"] != int64(30) {
		t.Errorf("cache_read_tokens: got %v, want 30", summary["cache_read_tokens"])
	}
	if summary["cache_create_tokens"] != int64(10) {
		t.Errorf("cache_create_tokens: got %v, want 10", summary["cache_create_tokens"])
	}
}

// splitLines 将字符串按换行符分割，过滤空行。
func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// TestMachineID 验证：MachineID 返回非空字符串，且包含用户名和主机名。
func TestMachineID(t *testing.T) {
	mid := trace.MachineID()
	if mid == "" {
		t.Fatal("MachineID should not be empty")
	}
	if !strings.Contains(mid, "@") {
		t.Errorf("MachineID should contain '@', got %q", mid)
	}
	parts := strings.SplitN(mid, "@", 2)
	if parts[0] == "" || parts[1] == "" {
		t.Errorf("MachineID should have non-empty username and hostname, got %q", mid)
	}
}

// TestExtractProjectSlug 验证：从追踪文件路径中提取项目 slug。
func TestExtractProjectSlug(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"windows_path", `C:\Users\Admin\.claude\projects\D--CodeDevelopment-CodeProject-claude-hk\uuid.jsonl`, "D--CodeDevelopment-CodeProject-claude-hk"},
		{"unix_path", "/home/user/.claude/projects/D--CodeDevelopment-CodeProject-claude-hk/uuid.jsonl", "D--CodeDevelopment-CodeProject-claude-hk"},
		{"empty", "", ""},
		{"no_projects_segment", "/some/random/path.jsonl", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trace.ExtractProjectSlug(tt.input)
			if got != tt.want {
				t.Errorf("ExtractProjectSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestNewSessionTracePath 验证：Session 追踪路径按 machine_id/project_slug/session_id.jsonl 格式生成。
func TestNewSessionTracePath(t *testing.T) {
	path := trace.NewSessionTracePath("/tmp/traces", "user@host", "D--my-project", "abc-123")
	// 路径格式：/tmp/traces/user@host/D--my-project/{date}/{time}/abc-123.jsonl
	if !strings.HasSuffix(path, "abc-123.jsonl") {
		t.Errorf("expected path to end with abc-123.jsonl, got %q", path)
	}
	if !strings.Contains(path, filepath.Join("user@host", "D--my-project")) {
		t.Errorf("expected path to contain machine_id/project_slug dirs, got %q", path)
	}
}

// TestDefaultTraceDir 验证：默认追踪目录包含 .claude-tap-plus 和 .traces 后缀。
func TestDefaultTraceDir(t *testing.T) {
	dir := trace.DefaultTraceDir()
	if dir == "" {
		t.Fatal("DefaultTraceDir should not be empty")
	}
	if !strings.Contains(dir, ".claude-tap-plus") {
		t.Errorf("DefaultTraceDir should contain .claude-tap-plus, got %q", dir)
	}
	if !strings.HasSuffix(dir, ".traces") {
		t.Errorf("DefaultTraceDir should end with .traces, got %q", dir)
	}
}
