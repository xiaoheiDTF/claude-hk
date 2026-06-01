// Package backend_test 包含后端 Log API 的验收测试，覆盖 GET /api/logs 接口。
package backend_test

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
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/api"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
)

// --- helpers ---

// logTestEnv 是日志 API 测试环境。
type logTestEnv struct {
	srv    *httptest.Server
	logDir string
}

// setupLogTest 创建日志 API 测试环境，自动清理临时目录。
func setupLogTest(t *testing.T) *logTestEnv {
	t.Helper()

	// 创建临时日志目录
	logDir, err := os.MkdirTemp("", "test-logs-*")
	if err != nil {
		t.Fatal(err)
	}

	// 创建临时数据库
	f, err := os.CreateTemp("", "test-log-*.db")
	if err != nil {
		os.RemoveAll(logDir)
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()

	s, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		os.RemoveAll(logDir)
		os.Remove(dbPath)
		t.Fatal(err)
	}

	t.Cleanup(func() {
		s.Close()
		os.Remove(dbPath)
		os.RemoveAll(logDir)
	})

	issueSvc := service.NewIssueService(s.Issues())
	sessionSvc := service.NewSessionService(s.Sessions())
	logSvc := service.NewLogService(logDir)

	router := api.NewRouter(api.Handlers{
		Issue:   api.NewIssueHandler(issueSvc),
		Session: api.NewSessionHandler(sessionSvc, issueSvc, service.NewTokenService(s.Sessions()), service.NewTraceService(s.Sessions())),
		Log:     api.NewLogHandler(logSvc),
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &logTestEnv{srv: srv, logDir: logDir}
}

// writeLogFile 在测试日志目录中写入指定日期的日志文件。
func writeLogFile(t *testing.T, logDir, date, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(logDir, date+".log"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// get 发送 GET 请求到测试环境。
func (e *logTestEnv) get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(e.srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// readLogJSON 读取 HTTP 响应并解析为 JSON。
func readLogJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("parse JSON: %v, body: %s", err, b)
	}
}

// --- 测试用日志内容 ---
// 使用与 internal/logger/logger.go 一致的格式：
//
//	HH:MM:SS.mmm [LEVEL] module: message

const testLogContent = `20:56:28.000 [INFO ] backend.cmd: backend starting: host=127.0.0.1 port=8080
20:56:28.100 [DEBUG] backend.cmd: parsing flags
20:56:28.200 [INFO ] backend: startup cleanup done
20:56:29.000 [WARN ] backend: config missing log_dir
20:56:29.500 [ERROR] backend.db: connection failed: timeout
20:56:30.000 [INFO ] backend: listening on 127.0.0.1:8080
20:56:30.100 [DEBUG] api: routes registered: 12 endpoints
20:56:31.000 [INFO ] api.issue: POST /api/issue/check repo=test/repo
20:56:31.500 [WARN ] api.issue: claim conflict issue=#10
20:56:32.000 [ERROR] api.session: session not found: sess_abc
20:56:32.500 [INFO ] api: request completed in 45ms
`

// --- tests ---

// TestLogs_DefaultQuery 验证：不带参数查询当天日志。
func TestLogs_DefaultQuery(t *testing.T) {
	env := setupLogTest(t)

	today := time.Now().Format("2006-01-02")
	writeLogFile(t, env.logDir, today, testLogContent)

	resp := env.get(t, "/api/logs")

	var result struct {
		Logs []struct {
			Timestamp string `json:"timestamp"`
			Level     string `json:"level"`
			Source    string `json:"source"`
			Message   string `json:"message"`
		} `json:"logs"`
	}
	readLogJSON(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if len(result.Logs) != 11 {
		t.Fatalf("expected 11 log entries, got %d", len(result.Logs))
	}

	// 验证每条日志字段完整
	for i, log := range result.Logs {
		if log.Timestamp == "" {
			t.Errorf("log[%d]: timestamp is empty", i)
		}
		if log.Level == "" {
			t.Errorf("log[%d]: level is empty", i)
		}
		if log.Source == "" {
			t.Errorf("log[%d]: source is empty", i)
		}
		if log.Message == "" {
			t.Errorf("log[%d]: message is empty", i)
		}
	}

	// 验证第一条的时间戳包含日期
	if !strings.HasPrefix(result.Logs[0].Timestamp, today+" ") {
		t.Errorf("expected timestamp to start with date %s, got %s", today, result.Logs[0].Timestamp)
	}

	// 验证第一条日志内容
	if result.Logs[0].Level != "INFO" {
		t.Errorf("expected level=INFO, got %s", result.Logs[0].Level)
	}
	if result.Logs[0].Source != "backend.cmd" {
		t.Errorf("expected source=backend.cmd, got %s", result.Logs[0].Source)
	}
}

// TestLogs_FilterByLevel 验证：按级别过滤日志。
func TestLogs_FilterByLevel(t *testing.T) {
	env := setupLogTest(t)

	today := time.Now().Format("2006-01-02")
	writeLogFile(t, env.logDir, today, testLogContent)

	resp := env.get(t, "/api/logs?level=ERROR")

	var result struct {
		Logs []struct {
			Level string `json:"level"`
		} `json:"logs"`
	}
	readLogJSON(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if len(result.Logs) != 2 {
		t.Fatalf("expected 2 ERROR entries, got %d", len(result.Logs))
	}

	for _, log := range result.Logs {
		if log.Level != "ERROR" {
			t.Errorf("expected level=ERROR, got %s", log.Level)
		}
	}
}

// TestLogs_FilterByLevel_CaseInsensitive 验证：级别过滤大小写不敏感。
func TestLogs_FilterByLevel_CaseInsensitive(t *testing.T) {
	env := setupLogTest(t)

	today := time.Now().Format("2006-01-02")
	writeLogFile(t, env.logDir, today, testLogContent)

	resp := env.get(t, "/api/logs?level=error")

	var result struct {
		Logs []struct {
			Level string `json:"level"`
		} `json:"logs"`
	}
	readLogJSON(t, resp, &result)

	if len(result.Logs) != 2 {
		t.Fatalf("expected 2 error entries (case insensitive), got %d", len(result.Logs))
	}
}

// TestLogs_FilterByDate 验证：按日期查询日志。
func TestLogs_FilterByDate(t *testing.T) {
	env := setupLogTest(t)

	// 写入不同日期的日志
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	writeLogFile(t, env.logDir, yesterday, "10:00:00.000 [INFO ] test: yesterday log\n")

	today := time.Now().Format("2006-01-02")
	writeLogFile(t, env.logDir, today, testLogContent)

	resp := env.get(t, "/api/logs?date="+yesterday)

	var result struct {
		Logs []struct {
			Timestamp string `json:"timestamp"`
			Source    string `json:"source"`
		} `json:"logs"`
	}
	readLogJSON(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if len(result.Logs) != 1 {
		t.Fatalf("expected 1 entry for yesterday, got %d", len(result.Logs))
	}

	if result.Logs[0].Source != "test" {
		t.Errorf("expected source=test, got %s", result.Logs[0].Source)
	}
}

// TestLogs_Limit 验证：限制返回条数。
func TestLogs_Limit(t *testing.T) {
	env := setupLogTest(t)

	today := time.Now().Format("2006-01-02")
	writeLogFile(t, env.logDir, today, testLogContent)

	resp := env.get(t, "/api/logs?limit=5")

	var result struct {
		Logs []struct {
			Level string `json:"level"`
		} `json:"logs"`
	}
	readLogJSON(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if len(result.Logs) != 5 {
		t.Fatalf("expected 5 entries with limit=5, got %d", len(result.Logs))
	}
}

// TestLogs_NoLogsForDate 验证：指定日期无日志文件时返回空数组。
func TestLogs_NoLogsForDate(t *testing.T) {
	env := setupLogTest(t)

	resp := env.get(t, "/api/logs?date=2026-01-01")

	var result struct {
		Logs []any `json:"logs"`
	}
	readLogJSON(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if len(result.Logs) != 0 {
		t.Fatalf("expected empty logs for date with no file, got %d", len(result.Logs))
	}
}

// TestLogs_MethodNotAllowed 验证：非 GET 方法返回 405。
func TestLogs_MethodNotAllowed(t *testing.T) {
	env := setupLogTest(t)

	resp, err := http.Post(env.srv.URL+"/api/logs", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var errResp struct {
		Code    string `json:"error"`
		Message string `json:"message"`
	}
	readLogJSON(t, resp, &errResp)

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}

	if errResp.Code != "method_not_allowed" {
		t.Errorf("expected error=method_not_allowed, got %s", errResp.Code)
	}
}

// TestLogs_CombinedFilter 验证：同时使用多个过滤条件。
func TestLogs_CombinedFilter(t *testing.T) {
	env := setupLogTest(t)

	today := time.Now().Format("2006-01-02")
	writeLogFile(t, env.logDir, today, testLogContent)

	resp := env.get(t, fmt.Sprintf("/api/logs?date=%s&level=WARN&limit=1", today))

	var result struct {
		Logs []struct {
			Level string `json:"level"`
		} `json:"logs"`
	}
	readLogJSON(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// WARN 有 2 条，但 limit=1，所以只返回 1 条
	if len(result.Logs) != 1 {
		t.Fatalf("expected 1 WARN entry with limit=1, got %d", len(result.Logs))
	}

	if result.Logs[0].Level != "WARN" {
		t.Errorf("expected level=WARN, got %s", result.Logs[0].Level)
	}
}
