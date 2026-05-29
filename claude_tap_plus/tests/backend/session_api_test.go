package backend_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/api"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
)

// --- Session test helpers ---

type sessionTestEnv struct {
	srv   *httptest.Server
	store *store.SQLiteStore
	db    *sql.DB
}

func setupSessionTest(t *testing.T) *sessionTestEnv {
	t.Helper()

	f, err := os.CreateTemp("", "test-session-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := f.Name()
	f.Close()

	s, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatal(err)
	}

	t.Cleanup(func() {
		s.Close()
		os.Remove(dbPath)
	})

	issueSvc := service.NewIssueService(s.Issues())
	sessionSvc := service.NewSessionService(s.Sessions())
	router := api.NewRouter(api.Handlers{
		Issue:   api.NewIssueHandler(issueSvc),
		Session: api.NewSessionHandler(sessionSvc),
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &sessionTestEnv{srv: srv, store: s, db: s.DB()}
}

func (e *sessionTestEnv) post(t *testing.T, path, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(e.srv.URL+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func (e *sessionTestEnv) get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(e.srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func sessionReadJSON(t *testing.T, resp *http.Response, v any) {
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

// --- Register tests ---

func TestSessionRegister(t *testing.T) {
	t.Run("success_writes_three_tables", func(t *testing.T) {
		env := setupSessionTest(t)

		resp := env.post(t, "/api/session/register", `{
			"session_id": "sess-001",
			"machine_id": "admin@host1",
			"os": "windows",
			"project_slug": "test-project",
			"project_cwd": "C:\\Projects\\test",
			"transcript_path": "C:\\Users\\.claude\\transcript.jsonl",
			"model": "GLM-5.1",
			"source": "startup"
		}`)

		var result map[string]string
		sessionReadJSON(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if result["status"] != "registered" {
			t.Errorf("expected status=registered, got %s", result["status"])
		}

		// Verify machines table.
		var machineCount int
		env.db.QueryRow("SELECT COUNT(*) FROM machines WHERE machine_id = 'admin@host1'").Scan(&machineCount)
		if machineCount != 1 {
			t.Errorf("expected 1 machine row, got %d", machineCount)
		}
		var username, hostname string
		env.db.QueryRow("SELECT username, hostname FROM machines WHERE machine_id = 'admin@host1'").Scan(&username, &hostname)
		if username != "admin" {
			t.Errorf("expected username=admin, got %s", username)
		}
		if hostname != "host1" {
			t.Errorf("expected hostname=host1, got %s", hostname)
		}

		// Verify projects table.
		var projectCount int
		env.db.QueryRow("SELECT COUNT(*) FROM projects WHERE project_slug = 'test-project'").Scan(&projectCount)
		if projectCount != 1 {
			t.Errorf("expected 1 project row, got %d", projectCount)
		}

		// Verify sessions table.
		var sessionStatus, sessMachineID, sessProject string
		env.db.QueryRow("SELECT status, machine_id, project_slug FROM sessions WHERE session_id = 'sess-001'").
			Scan(&sessionStatus, &sessMachineID, &sessProject)
		if sessionStatus != "active" {
			t.Errorf("expected active, got %s", sessionStatus)
		}
		if sessMachineID != "admin@host1" {
			t.Errorf("expected admin@host1, got %s", sessMachineID)
		}
		if sessProject != "test-project" {
			t.Errorf("expected test-project, got %s", sessProject)
		}
	})

	t.Run("duplicate_returns_409", func(t *testing.T) {
		env := setupSessionTest(t)

		body := `{"session_id":"sess-dup","machine_id":"admin@host1","os":"windows","project_slug":"proj","project_cwd":"/proj","transcript_path":"/t.jsonl"}`
		env.post(t, "/api/session/register", body)

		resp := env.post(t, "/api/session/register", body)

		var errResp struct {
			Code    string `json:"error"`
			Message string `json:"message"`
		}
		sessionReadJSON(t, resp, &errResp)

		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("expected 409, got %d", resp.StatusCode)
		}
		if errResp.Code != "session_exists" {
			t.Errorf("expected error=session_exists, got %s", errResp.Code)
		}
	})

	t.Run("missing_required_fields_returns_400", func(t *testing.T) {
		env := setupSessionTest(t)

		cases := []struct {
			name string
			body string
		}{
			{"missing_session_id", `{"machine_id":"m","os":"windows","project_slug":"p","project_cwd":"/p","transcript_path":"/t"}`},
			{"missing_machine_id", `{"session_id":"s","os":"windows","project_slug":"p","project_cwd":"/p","transcript_path":"/t"}`},
			{"missing_project_slug", `{"session_id":"s","machine_id":"m","os":"windows","project_cwd":"/p","transcript_path":"/t"}`},
			{"missing_transcript_path", `{"session_id":"s","machine_id":"m","os":"windows","project_slug":"p","project_cwd":"/p"}`},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				resp := env.post(t, "/api/session/register", tc.body)
				if resp.StatusCode != http.StatusBadRequest {
					t.Fatalf("expected 400, got %d", resp.StatusCode)
				}
			})
		}
	})

	t.Run("invalid_json_returns_400", func(t *testing.T) {
		env := setupSessionTest(t)
		resp := env.post(t, "/api/session/register", "not json")
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		env := setupSessionTest(t)
		resp, err := http.Get(env.srv.URL + "/api/session/register")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", resp.StatusCode)
		}
	})

	t.Run("machine_id_update_last_seen", func(t *testing.T) {
		env := setupSessionTest(t)

		// Register first session.
		env.post(t, "/api/session/register", `{
			"session_id":"sess-a","machine_id":"admin@host1","os":"windows",
			"project_slug":"proj-a","project_cwd":"/a","transcript_path":"/ta.jsonl"}`)

		// Register second session with same machine_id.
		env.post(t, "/api/session/register", `{
			"session_id":"sess-b","machine_id":"admin@host1","os":"windows",
			"project_slug":"proj-b","project_cwd":"/b","transcript_path":"/tb.jsonl"}`)

		// Machine should have only 1 row.
		var count int
		env.db.QueryRow("SELECT COUNT(*) FROM machines WHERE machine_id = 'admin@host1'").Scan(&count)
		if count != 1 {
			t.Errorf("expected 1 machine row, got %d", count)
		}

		// last_seen_at should be updated (not null).
		var lastSeen sql.NullString
		env.db.QueryRow("SELECT last_seen_at FROM machines WHERE machine_id = 'admin@host1'").Scan(&lastSeen)
		if !lastSeen.Valid {
			t.Error("expected last_seen_at to be set")
		}
	})
}

// --- Close tests ---

func TestSessionClose(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := setupSessionTest(t)
		registerSession(t, env, "sess-close-1")

		resp := env.post(t, "/api/session/close",
			`{"session_id":"sess-close-1","reason":"prompt_input_exit"}`)

		var result map[string]string
		sessionReadJSON(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if result["status"] != "closed" {
			t.Errorf("expected status=closed, got %s", result["status"])
		}

		// Verify DB.
		var status, reason string
		env.db.QueryRow("SELECT status, close_reason FROM sessions WHERE session_id = 'sess-close-1'").Scan(&status, &reason)
		if status != "closed" {
			t.Errorf("expected closed, got %s", status)
		}
		if reason != "prompt_input_exit" {
			t.Errorf("expected prompt_input_exit, got %s", reason)
		}
	})

	t.Run("not_found_returns_404", func(t *testing.T) {
		env := setupSessionTest(t)

		resp := env.post(t, "/api/session/close",
			`{"session_id":"nonexistent","reason":"exit"}`)

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
	})

	t.Run("already_closed_returns_404", func(t *testing.T) {
		env := setupSessionTest(t)
		registerSession(t, env, "sess-dbl-close")

		env.post(t, "/api/session/close",
			`{"session_id":"sess-dbl-close","reason":"first"}`)

		resp := env.post(t, "/api/session/close",
			`{"session_id":"sess-dbl-close","reason":"second"}`)

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
	})

	t.Run("missing_session_id_returns_400", func(t *testing.T) {
		env := setupSessionTest(t)
		resp := env.post(t, "/api/session/close", `{}`)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		env := setupSessionTest(t)
		resp, err := http.Get(env.srv.URL + "/api/session/close")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", resp.StatusCode)
		}
	})
}

// --- List tests ---

func TestSessionList(t *testing.T) {
	seedSessions := func(t *testing.T, env *sessionTestEnv) {
		registerSession(t, env, "sess-list-1")
		registerSessionWith(t, env, "sess-list-2", "other@host2", "linux", "proj-b", "/b")
		registerSessionWith(t, env, "sess-list-3", "admin@host1", "windows", "proj-a", "/a")
		// Close the third one.
		env.post(t, "/api/session/close", `{"session_id":"sess-list-3","reason":"done"}`)
	}

	t.Run("all_sessions", func(t *testing.T) {
		env := setupSessionTest(t)
		seedSessions(t, env)

		resp := env.get(t, "/api/sessions")

		var result struct {
			Sessions []struct {
				SessionID string `json:"session_id"`
				Status    string `json:"status"`
			} `json:"sessions"`
		}
		sessionReadJSON(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if len(result.Sessions) != 3 {
			t.Fatalf("expected 3 sessions, got %d", len(result.Sessions))
		}
	})

	t.Run("filter_by_machine_id", func(t *testing.T) {
		env := setupSessionTest(t)
		seedSessions(t, env)

		resp := env.get(t, "/api/sessions?machine_id=other@host2")

		var result struct {
			Sessions []struct {
				SessionID string `json:"session_id"`
			} `json:"sessions"`
		}
		sessionReadJSON(t, resp, &result)

		if len(result.Sessions) != 1 {
			t.Fatalf("expected 1 session, got %d", len(result.Sessions))
		}
		if result.Sessions[0].SessionID != "sess-list-2" {
			t.Errorf("expected sess-list-2, got %s", result.Sessions[0].SessionID)
		}
	})

	t.Run("filter_by_status_active", func(t *testing.T) {
		env := setupSessionTest(t)
		seedSessions(t, env)

		resp := env.get(t, "/api/sessions?status=active")

		var result struct {
			Sessions []struct {
				SessionID string `json:"session_id"`
				Status    string `json:"status"`
			} `json:"sessions"`
		}
		sessionReadJSON(t, resp, &result)

		if len(result.Sessions) != 2 {
			t.Fatalf("expected 2 active sessions, got %d", len(result.Sessions))
		}
		for _, s := range result.Sessions {
			if s.Status != "active" {
				t.Errorf("expected active, got %s", s.Status)
			}
		}
	})

	t.Run("filter_by_project_slug", func(t *testing.T) {
		env := setupSessionTest(t)
		seedSessions(t, env)

		resp := env.get(t, "/api/sessions?project_slug=proj-b")

		var result struct {
			Sessions []struct {
				SessionID string `json:"session_id"`
			} `json:"sessions"`
		}
		sessionReadJSON(t, resp, &result)

		if len(result.Sessions) != 1 {
			t.Fatalf("expected 1 session, got %d", len(result.Sessions))
		}
	})

	t.Run("empty_result", func(t *testing.T) {
		env := setupSessionTest(t)

		resp := env.get(t, "/api/sessions")

		var result struct {
			Sessions []any `json:"sessions"`
		}
		sessionReadJSON(t, resp, &result)

		if len(result.Sessions) != 0 {
			t.Fatalf("expected 0 sessions, got %d", len(result.Sessions))
		}
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		env := setupSessionTest(t)
		resp := env.post(t, "/api/sessions", "")
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", resp.StatusCode)
		}
	})
}

// --- Get tests ---

func TestSessionGet(t *testing.T) {
	t.Run("existing_session", func(t *testing.T) {
		env := setupSessionTest(t)
		registerSession(t, env, "sess-get-1")

		resp := env.get(t, "/api/session/sess-get-1")

		var result struct {
			SessionID      string `json:"session_id"`
			MachineID      string `json:"machine_id"`
			OS             string `json:"os"`
			ProjectSlug    string `json:"project_slug"`
			ProjectCwd     string `json:"project_cwd"`
			TranscriptPath string `json:"transcript_path"`
			Model          string `json:"model"`
			Source         string `json:"source"`
			Status         string `json:"status"`
		}
		sessionReadJSON(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if result.SessionID != "sess-get-1" {
			t.Errorf("expected sess-get-1, got %s", result.SessionID)
		}
		if result.Status != "active" {
			t.Errorf("expected active, got %s", result.Status)
		}
		if result.MachineID != "admin@host1" {
			t.Errorf("expected admin@host1, got %s", result.MachineID)
		}
		if result.TranscriptPath == "" {
			t.Error("expected non-empty transcript_path")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		env := setupSessionTest(t)
		resp := env.get(t, "/api/session/nonexistent")

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
	})

	t.Run("closed_session_has_close_fields", func(t *testing.T) {
		env := setupSessionTest(t)
		registerSession(t, env, "sess-closed-detail")
		env.post(t, "/api/session/close",
			`{"session_id":"sess-closed-detail","reason":"prompt_input_exit"}`)

		resp := env.get(t, "/api/session/sess-closed-detail")

		var result struct {
			Status      string  `json:"status"`
			CloseReason string  `json:"close_reason"`
			ClosedAt    *string `json:"closed_at"`
		}
		sessionReadJSON(t, resp, &result)

		if result.Status != "closed" {
			t.Errorf("expected closed, got %s", result.Status)
		}
		if result.CloseReason != "prompt_input_exit" {
			t.Errorf("expected prompt_input_exit, got %s", result.CloseReason)
		}
		if result.ClosedAt == nil {
			t.Error("expected closed_at to be set")
		}
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		env := setupSessionTest(t)
		resp := env.post(t, "/api/session/some-id", "")
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", resp.StatusCode)
		}
	})
}

// --- Timeout cleanup tests ---

func TestCleanupTimedOutSessions(t *testing.T) {
	t.Run("marks_stale_sessions_closed", func(t *testing.T) {
		env := setupSessionTest(t)

		// Insert a session with an old registered_at.
		env.db.Exec(`INSERT INTO sessions (session_id, machine_id, os, project_slug, project_cwd,
			transcript_path, status, registered_at)
			VALUES ('stale-sess', 'admin@host', 'linux', 'proj', '/p', '/t.jsonl', 'active',
			datetime('now', '-25 hours'))`)

		svc := service.NewCleanupService(env.store.Sessions())
		svc.CleanupTimedOutSessions(context.Background())

		var status, reason string
		env.db.QueryRow("SELECT status, close_reason FROM sessions WHERE session_id = 'stale-sess'").Scan(&status, &reason)
		if status != "closed" {
			t.Errorf("expected closed, got %s", status)
		}
		if reason != "timeout_cleanup" {
			t.Errorf("expected timeout_cleanup, got %s", reason)
		}
	})

	t.Run("recent_sessions_untouched", func(t *testing.T) {
		env := setupSessionTest(t)
		registerSession(t, env, "fresh-sess")

		svc := service.NewCleanupService(env.store.Sessions())
		svc.CleanupTimedOutSessions(context.Background())

		var status string
		env.db.QueryRow("SELECT status FROM sessions WHERE session_id = 'fresh-sess'").Scan(&status)
		if status != "active" {
			t.Errorf("expected active, got %s", status)
		}
	})
}

// --- Full lifecycle test ---

func TestSessionFullLifecycle(t *testing.T) {
	env := setupSessionTest(t)

	// 1. Register.
	resp := env.post(t, "/api/session/register", fmt.Sprintf(`{
		"session_id": "lifecycle-1",
		"machine_id": "admin@host1",
		"os": "windows",
		"project_slug": "proj-lifecycle",
		"project_cwd": "/proj",
		"transcript_path": "/t.jsonl",
		"model": "GLM-5.1",
		"source": "startup",
		"local_trace_path": "/traces/lifecycle-1.jsonl"
	}`))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("step 1 register: expected 200, got %d", resp.StatusCode)
	}

	// 2. Get detail.
	resp = env.get(t, "/api/session/lifecycle-1")
	var detail struct {
		Status         string `json:"status"`
		Model          string `json:"model"`
		LocalTracePath string `json:"local_trace_path"`
	}
	sessionReadJSON(t, resp, &detail)
	if detail.Status != "active" {
		t.Fatalf("step 2 get: expected active, got %s", detail.Status)
	}
	if detail.Model != "GLM-5.1" {
		t.Errorf("step 2 get: expected model=GLM-5.1, got %s", detail.Model)
	}
	if detail.LocalTracePath != "/traces/lifecycle-1.jsonl" {
		t.Errorf("step 2 get: expected local_trace_path set, got %s", detail.LocalTracePath)
	}

	// 3. List with filter.
	resp = env.get(t, "/api/sessions?status=active&machine_id=admin@host1")
	var list struct {
		Sessions []struct {
			SessionID string `json:"session_id"`
		} `json:"sessions"`
	}
	sessionReadJSON(t, resp, &list)
	if len(list.Sessions) != 1 {
		t.Fatalf("step 3 list: expected 1, got %d", len(list.Sessions))
	}

	// 4. Close.
	resp = env.post(t, "/api/session/close",
		`{"session_id":"lifecycle-1","reason":"prompt_input_exit"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("step 4 close: expected 200, got %d", resp.StatusCode)
	}

	// 5. Verify closed via get.
	resp = env.get(t, "/api/session/lifecycle-1")
	sessionReadJSON(t, resp, &detail)
	if detail.Status != "closed" {
		t.Fatalf("step 5 verify: expected closed, got %s", detail.Status)
	}

	// 6. Double close returns 404.
	resp = env.post(t, "/api/session/close",
		`{"session_id":"lifecycle-1","reason":"again"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("step 6 double close: expected 404, got %d", resp.StatusCode)
	}
}

// --- Helpers ---

func registerSession(t *testing.T, env *sessionTestEnv, sessionID string) {
	t.Helper()
	registerSessionWith(t, env, sessionID, "admin@host1", "windows", "proj-a", "/a")
}

func registerSessionWith(t *testing.T, env *sessionTestEnv, sessionID, machineID, osName, projectSlug, projectCwd string) {
	t.Helper()
	body := fmt.Sprintf(`{
		"session_id": "%s",
		"machine_id": "%s",
		"os": "%s",
		"project_slug": "%s",
		"project_cwd": "%s",
		"transcript_path": "/transcripts/%s.jsonl"
	}`, sessionID, machineID, osName, projectSlug, projectCwd, sessionID)
	resp := env.post(t, "/api/session/register", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register session %s: expected 200, got %d", sessionID, resp.StatusCode)
	}
}
