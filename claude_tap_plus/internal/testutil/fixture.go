package testutil

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fixture represents a single test case with request, response, and expected trace output.
type Fixture struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Request       FixtureRequest  `json:"request"`
	Response      FixtureResponse `json:"response"`
	ExpectedTrace ExpectedTrace   `json:"expected_trace"`
}

// FixtureRequest is the HTTP request to send through the proxy.
type FixtureRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
}

// FixtureResponse is what the mock upstream server should return.
type FixtureResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body,omitempty"`
	RawSSE  string            `json:"raw_sse,omitempty"`
	Gzip    bool              `json:"gzip,omitempty"`
}

// ExpectedTrace contains fields to verify in the trace output.
type ExpectedTrace struct {
	SessionID         string `json:"session_id"`
	InputTokens       int64  `json:"input_tokens,omitempty"`
	OutputTokens      int64  `json:"output_tokens,omitempty"`
	CacheReadTokens   int64  `json:"cache_read_tokens,omitempty"`
	CacheCreateTokens int64  `json:"cache_create_tokens,omitempty"`
}

// LoadFixture reads a fixture JSON file from testdata/fixtures/.
func LoadFixture(t *testing.T, name string) *Fixture {
	t.Helper()
	path := findUpward(t, filepath.Join("testdata", "fixtures", name))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var f Fixture
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return &f
}

// FindDir searches upward for a directory. Calls t.Skip if not found.
func FindDir(t *testing.T, relPath string) string {
	t.Helper()
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, relPath)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Skipf("%s not found", relPath)
	return ""
}

// findUpward searches upward for a file. Calls t.Fatal if not found.
func findUpward(t *testing.T, relPath string) string {
	t.Helper()
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, relPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("file not found: %s", relPath)
	return ""
}

// CreateMockServer creates an httptest.Server that returns the fixture's response.
func CreateMockServer(t *testing.T, fx *Fixture) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range fx.Response.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(fx.Response.Status)

		if fx.Response.RawSSE != "" {
			flusher, _ := w.(http.Flusher)
			_, _ = io.WriteString(w, fx.Response.RawSSE)
			if flusher != nil {
				flusher.Flush()
			}
			return
		}

		bodyBytes, err := json.Marshal(fx.Response.Body)
		if err != nil {
			t.Fatalf("marshal response body: %v", err)
		}

		if fx.Response.Gzip {
			w.Header().Set("Content-Encoding", "gzip")
			var buf bytes.Buffer
			gw := gzip.NewWriter(&buf)
			_, _ = gw.Write(bodyBytes)
			gw.Close()
			_, _ = w.Write(buf.Bytes())
			return
		}

		_, _ = w.Write(bodyBytes)
	}))
}

// SendRequestThroughProxy sends the fixture's request to the proxy.
func SendRequestThroughProxy(t *testing.T, fx *Fixture, proxyURL string) *http.Response {
	t.Helper()
	bodyBytes, err := json.Marshal(fx.Request.Body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	req, err := http.NewRequest(fx.Request.Method, proxyURL+fx.Request.Path, bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	for k, v := range fx.Request.Headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	return resp
}

// ReadTraceRecords reads the trace JSONL file and returns all records.
func ReadTraceRecords(t *testing.T, path string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read trace file: %v", err)
	}
	var records []map[string]any
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var r map[string]any
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}
		records = append(records, r)
	}
	return records
}

// AssertTrace verifies that a trace record matches the expected trace.
func AssertTrace(t *testing.T, record map[string]any, expected ExpectedTrace) {
	t.Helper()
	if expected.SessionID != "" {
		got, _ := record["session_id"].(string)
		if got != expected.SessionID {
			t.Errorf("session_id: got %q, want %q", got, expected.SessionID)
		}
	}
	resp, _ := record["response"].(map[string]any)
	if resp == nil {
		t.Fatal("trace record has no response field")
	}
	body, _ := resp["body"].(map[string]any)
	if body == nil {
		return
	}
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		usage = body
	}
	assertToken := func(key string, expected int64) {
		if expected == 0 {
			return
		}
		got, ok := toInt64(usage[key])
		if !ok {
			return
		}
		if got != expected {
			t.Errorf("%s: got %d, want %d", key, got, expected)
		}
	}
	assertToken("input_tokens", expected.InputTokens)
	assertToken("output_tokens", expected.OutputTokens)
	assertToken("cache_read_input_tokens", expected.CacheReadTokens)
	assertToken("cache_creation_input_tokens", expected.CacheCreateTokens)
}

// AssertResponseStatus checks the HTTP response status code.
func AssertResponseStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("status: got %d, want %d", resp.StatusCode, expected)
	}
}

// DrainAndClose reads the response body and closes it.
func DrainAndClose(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return data
}

// TempTraceFile creates a temporary file for trace output and returns its path.
func TempTraceFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "trace.jsonl")
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	default:
		return 0, false
	}
}

// FixtureName returns a short description for error messages.
func FixtureName(fx *Fixture) string {
	return fmt.Sprintf("%s (%s)", fx.Name, fx.Description)
}
