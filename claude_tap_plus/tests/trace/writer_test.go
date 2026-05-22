package trace_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/trace"
)

func TestNewTracePath(t *testing.T) {
	path := trace.NewTracePath("/tmp/traces")
	// Path format: /tmp/traces/{project}/{date}_{time}_{hex}.jsonl
	filename := filepath.Base(path)
	if !strings.HasSuffix(filename, ".jsonl") {
		t.Errorf("expected .jsonl suffix, got %q", filename)
	}
	// Filename format: 2006-01-02_150405_a1b2c3.jsonl
	parts := strings.Split(strings.TrimSuffix(filename, ".jsonl"), "_")
	if len(parts) != 3 {
		t.Errorf("expected 3-part filename (date_time_hex), got %d parts: %q", len(parts), parts)
	}
	if len(parts[0]) != 10 || parts[0][4] != '-' {
		t.Errorf("expected date prefix YYYY-MM-DD, got %q", parts[0])
	}
}

func TestTraceWriterWriteAndSummary(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "test_trace.jsonl")

	w, err := trace.NewTraceWriter(tracePath)
	if err != nil {
		t.Fatalf("NewTraceWriter: %v", err)
	}

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

func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
