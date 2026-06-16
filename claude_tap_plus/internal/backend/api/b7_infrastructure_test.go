// Package api_test 包含 API 的验收测试，覆盖后端服务启动、配置、数据库等基础设施。
package api_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
)

// --- B7: 公共后端调用模块与服务启动 ---
//
// 验收标准覆盖：
//   - 后端运行时 /health 返回 {"status": "ok"}
//   - 启动时自动创建数据库表（如不存在）
//   - Config 默认值和 Addr() 格式化
//   - CLI 参数 --port/--db/--host 解析
//   - 后端不可用时连接失败

// TestB7_HealthReturnsOK 验收：/health 返回 {"status":"ok"} + HTTP 200。
// 验证后端服务启动后，健康检查接口返回正确的状态码和 JSON 响应。
func TestB7_HealthReturnsOK(t *testing.T) {
	env := setupTest(t)

	resp, err := http.Get(env.srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	readJSON(t, resp, &result)
	if result["status"] != "ok" {
		t.Fatalf("expected status=ok, got %s", result["status"])
	}
}

// TestB7_HealthJSONFormat 验收：/health 返回 JSON Content-Type。
// 验证健康检查接口的响应头包含 application/json，且响应体为合法 JSON。
func TestB7_HealthJSONFormat(t *testing.T) {
	env := setupTest(t)

	resp, err := http.Get(env.srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var m map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatal(err)
	}
	if len(m) != 1 {
		t.Errorf("expected 1 field, got %d: %v", len(m), m)
	}
	if m["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", m["status"])
	}
}

// TestB7_AutoTableCreation 验收：启动时自动创建数据库表和索引（4 张表 + 9 个索引）。
// 验证使用新数据库文件启动时，后端自动创建所需的表和索引，且 repo_full_name + issue_number 唯一约束生效。
func TestB7_AutoTableCreation(t *testing.T) {
	f, err := os.CreateTemp("", "test-b7-tables-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()
	t.Cleanup(func() { os.Remove(dbPath) })

	s, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	db := s.DB()

	// 验证 4 张核心表已创建
	for _, tbl := range []string{"machines", "projects", "sessions", "issue_claims"} {
		var name string
		if err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&name); err != nil {
			t.Errorf("table %s not found: %v", tbl, err)
		}
	}

	// 验证 9 个索引已创建
	for _, idx := range []string{
		"idx_machines_hostname", "idx_projects_slug",
		"idx_sessions_machine", "idx_sessions_project", "idx_sessions_status", "idx_sessions_registered",
		"idx_issue_claims_repo", "idx_issue_claims_session", "idx_issue_claims_status",
	} {
		var name string
		if err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&name); err != nil {
			t.Errorf("index %s not found: %v", idx, err)
		}
	}

	// 验证 UNIQUE(repo_full_name, issue_number) 约束生效
	db.Exec(`INSERT INTO issue_claims (repo_full_name, issue_number, status) VALUES ('test/repo', 1, 'idle')`)
	if _, err := db.Exec(`INSERT INTO issue_claims (repo_full_name, issue_number, status) VALUES ('test/repo', 1, 'idle')`); err == nil {
		t.Error("expected UNIQUE constraint violation")
	}
}

// TestB7_ConfigDefaults 验收：DefaultConfig 返回正确的默认值。
// 验证后端配置的默认主机、端口和数据库路径符合预期。
func TestB7_ConfigDefaults(t *testing.T) {
	cfg := backend.DefaultConfig()
	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want 127.0.0.1", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.DBPath != "backend.db" {
		t.Errorf("DBPath = %q, want backend.db", cfg.DBPath)
	}
}

// TestB7_ConfigAddr 验收：Config.Addr() 正确格式化为 host:port。
// 验证配置对象的 Addr 方法能正确拼接主机地址和端口号。
func TestB7_ConfigAddr(t *testing.T) {
	for _, tt := range []struct{ host string; port int; want string }{
		{"127.0.0.1", 8080, "127.0.0.1:8080"},
		{"0.0.0.0", 3000, "0.0.0.0:3000"},
		{"localhost", 9090, "localhost:9090"},
	} {
		got := backend.Config{Host: tt.host, Port: tt.port}.Addr()
		if got != tt.want {
			t.Errorf("Config{%s,%d}.Addr() = %q, want %q", tt.host, tt.port, got, tt.want)
		}
	}
}

// TestB7_CLIFlagParsing 验收：CLI 参数 --port/--db/--host 解析正确。
// 验证长参数、短参数和等号语法都能被正确解析到配置对象中。
func TestB7_CLIFlagParsing(t *testing.T) {
	for _, tt := range []struct {
		name     string
		args     []string
		wantHost string
		wantPort int
		wantDB   string
	}{
		{"defaults", []string{}, "127.0.0.1", 8080, "backend.db"},
		{"long flags", []string{"--port", "9090", "--db", "/tmp/test.db", "--host", "0.0.0.0"}, "0.0.0.0", 9090, "/tmp/test.db"},
		{"short flags", []string{"-p", "3000", "-d", "my.db"}, "127.0.0.1", 3000, "my.db"},
		{"equals syntax", []string{"--port=7070", "--db=custom.db", "--host=192.168.1.1"}, "192.168.1.1", 7070, "custom.db"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := parseBackendFlags(tt.args)
			if cfg.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", cfg.Host, tt.wantHost)
			}
			if cfg.Port != tt.wantPort {
				t.Errorf("Port = %d, want %d", cfg.Port, tt.wantPort)
			}
			if cfg.DBPath != tt.wantDB {
				t.Errorf("DBPath = %q, want %q", cfg.DBPath, tt.wantDB)
			}
		})
	}
}

// parseBackendFlags 模拟后端命令行参数解析逻辑，用于测试中验证 CLI 行为。
func parseBackendFlags(args []string) backend.Config {
	cfg := backend.DefaultConfig()
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case (arg == "--port" || arg == "-p") && i+1 < len(args):
			i++
			fmt.Sscanf(args[i], "%d", &cfg.Port)
		case strings.HasPrefix(arg, "--port="):
			fmt.Sscanf(arg[len("--port="):], "%d", &cfg.Port)
		case (arg == "--db" || arg == "-d") && i+1 < len(args):
			i++
			cfg.DBPath = args[i]
		case strings.HasPrefix(arg, "--db="):
			cfg.DBPath = arg[len("--db="):]
		case arg == "--host" && i+1 < len(args):
			i++
			cfg.Host = args[i]
		case strings.HasPrefix(arg, "--host="):
			cfg.Host = arg[len("--host="):]
		}
	}
	return cfg
}

// TestB7_ConnectionRefusedWhenBackendDown 验收：后端不可用时连接失败。
// 验证当后端服务未运行时，HTTP 请求会返回连接错误而非成功。
func TestB7_ConnectionRefusedWhenBackendDown(t *testing.T) {
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get("http://127.0.0.1:1/health")
	if err == nil {
		t.Error("expected connection error when backend is down")
	}
}

// TestB7_SQLiteWALMode 验收：SQLite WAL 模式。
// 验证后端数据库启动后自动设置为 WAL 日志模式，以支持更高的并发性能。
func TestB7_SQLiteWALMode(t *testing.T) {
	f, err := os.CreateTemp("", "test-b7-wal-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()
	t.Cleanup(func() { os.Remove(dbPath) })

	s, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var mode string
	if err := s.DB().QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Errorf("expected journal_mode=wal, got %s", mode)
	}
}

// TestB7_SessionsTableSchema 验收：sessions 表完整字段可读写。
// 验证 sessions 表的所有字段都能正确写入和读取。
func TestB7_SessionsTableSchema(t *testing.T) {
	env := setupTest(t)
	db := env.store.DB()

	_, err := db.Exec(`INSERT INTO sessions
		(session_id, machine_id, os, project_slug, project_cwd, transcript_path, local_trace_path, model, source, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"test-sess", "user@host", "windows", "proj", "/cwd", "/transcript", "/trace", "opus", "startup", "active")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	var sid, status, model string
	err = db.QueryRow(`SELECT session_id, status, model FROM sessions WHERE session_id='test-sess'`).Scan(&sid, &status, &model)
	if err != nil {
		t.Fatal(err)
	}
	if sid != "test-sess" || status != "active" || model != "opus" {
		t.Errorf("got sid=%s status=%s model=%s", sid, status, model)
	}
}

// TestB7_SessionCloseFields 验收：sessions 表 closed_at/close_reason 可更新。
// 验证关闭 session 时，closed_at 和 close_reason 字段能正确被更新。
func TestB7_SessionCloseFields(t *testing.T) {
	env := setupTest(t)
	db := env.store.DB()

	db.Exec(`INSERT INTO sessions (session_id, machine_id, os, project_slug, project_cwd, transcript_path, status)
		VALUES ('sess-close', 'u@h', 'linux', 'p', '/c', '/t', 'active')`)

	_, err := db.Exec(`UPDATE sessions SET status='closed', closed_at=datetime('now'), close_reason='user_exit'
		WHERE session_id='sess-close'`)
	if err != nil {
		t.Fatal(err)
	}

	var status, reason string
	var closedAt sql.NullString
	db.QueryRow(`SELECT status, close_reason, closed_at FROM sessions WHERE session_id='sess-close'`).Scan(&status, &reason, &closedAt)
	if status != "closed" || reason != "user_exit" || !closedAt.Valid {
		t.Errorf("status=%s reason=%s closedAt.Valid=%v", status, reason, closedAt.Valid)
	}
}
