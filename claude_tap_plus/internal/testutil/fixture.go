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

// Fixture 表示一个完整的代理测试用例，包含请求、预期响应和预期追踪输出。
type Fixture struct {
	Name          string          `json:"name"`          // 用例名称
	Description   string          `json:"description"`   // 用例描述
	Request       FixtureRequest  `json:"request"`       // 发送给代理的 HTTP 请求
	Response      FixtureResponse `json:"response"`      // 模拟上游服务器返回的响应
	ExpectedTrace ExpectedTrace   `json:"expected_trace"` // 预期生成的追踪记录验证项
}

// FixtureRequest 定义通过代理发送的 HTTP 请求。
type FixtureRequest struct {
	Method  string            `json:"method"`  // HTTP 方法
	Path    string            `json:"path"`    // 请求路径
	Headers map[string]string `json:"headers"` // 请求头
	Body    any               `json:"body"`    // 请求体（JSON 对象）
}

// FixtureResponse 定义模拟上游服务器应返回的响应。
type FixtureResponse struct {
	Status  int               `json:"status"`              // HTTP 状态码
	Headers map[string]string `json:"headers"`             // 响应头
	Body    any               `json:"body,omitempty"`      // 响应体（JSON 对象，与 RawSSE 二选一）
	RawSSE  string            `json:"raw_sse,omitempty"`   // 原始 SSE 流文本（与 Body 二选一）
	Gzip    bool              `json:"gzip,omitempty"`      // 是否使用 gzip 压缩响应体
}

// ExpectedTrace 定义需要在追踪输出中验证的字段。
type ExpectedTrace struct {
	SessionID         string `json:"session_id"`                    // 预期会话标识
	InputTokens       int64  `json:"input_tokens,omitempty"`        // 预期输入 Token 数量
	OutputTokens      int64  `json:"output_tokens,omitempty"`       // 预期输出 Token 数量
	CacheReadTokens   int64  `json:"cache_read_tokens,omitempty"`   // 预期缓存读取 Token 数量
	CacheCreateTokens int64  `json:"cache_create_tokens,omitempty"` // 预期缓存创建 Token 数量
}

// LoadFixture 从 testdata/fixtures/ 目录读取夹具 JSON 文件。
//
// 搜索逻辑：从当前工作目录开始向上查找最多 10 层目录，
// 定位到 testdata/fixtures/{name} 文件。
// 如果读取或解析失败，调用 t.Fatalf 终止测试。
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

// FindDir 从当前目录向上搜索指定的相对路径目录。
//
// 最多向上搜索 10 层父目录，如果未找到则调用 t.Skip 跳过当前测试。
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

// findUpward 从当前目录向上搜索指定的相对路径文件。
//
// 最多向上搜索 10 层父目录，如果未找到则调用 t.Fatal 终止测试。
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

// CreateMockServer 创建一个 httptest.Server，按夹具配置返回模拟响应。
//
// 响应生成逻辑：
//   - 如果 RawSSE 非空，按 SSE 格式返回（支持流式刷新）
//   - 如果 Gzip 为 true，对响应体进行 gzip 压缩
//   - 否则返回 JSON 序列化后的 Body
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

// SendRequestThroughProxy 将夹具中定义的请求发送到代理服务器。
//
// 请求体会被 JSON 序列化，目标地址为 proxyURL + fx.Request.Path。
// 如果请求构建或发送失败，调用 t.Fatalf 终止测试。
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

// ReadTraceRecords 读取 JSONL 格式的追踪文件，返回所有解析后的记录。
//
// 逐行解析，跳过空行，忽略无法解析的行。
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

// AssertTrace 验证单条追踪记录是否符合预期。
//
// 检查项：
//   - session_id 是否匹配
//   - response.body.usage 中的 input_tokens、output_tokens、
//     cache_read_input_tokens、cache_creation_input_tokens 是否匹配
//
// 预期值为 0 的字段会被跳过验证（表示不检查该项）。
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

// AssertResponseStatus 检查 HTTP 响应状态码是否与预期一致。
func AssertResponseStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("status: got %d, want %d", resp.StatusCode, expected)
	}
}

// DrainAndClose 读取响应体的全部内容并关闭响应体。
// 如果读取失败，调用 t.Fatalf 终止测试。
func DrainAndClose(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return data
}

// TempTraceFile 在临时目录中创建一个追踪文件并返回其路径。
func TempTraceFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "trace.jsonl")
}

// toInt64 将任意数值类型转换为 int64。
// 支持 float64、int、int64；不支持的类型返回 (0, false)。
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

// FixtureName 返回夹具的名称和描述，用于测试错误信息。
func FixtureName(fx *Fixture) string {
	return fmt.Sprintf("%s (%s)", fx.Name, fx.Description)
}
