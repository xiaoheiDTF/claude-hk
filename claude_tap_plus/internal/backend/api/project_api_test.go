// Package api_test 包含 GET /api/projects 接口的 BDD + TDD 验收测试。
// 覆盖：获取全部、空列表、方法限制、按 last_seen_at 倒序验证。
// 所有测试数据均通过 POST /api/session/register 产生（该接口 upsert projects 表）。
package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// --- response types for project API ---

// projectListResponse 是 GET /api/projects 的响应结构。
type projectListResponse struct {
	Projects []projectListItem `json:"projects"`
}

// projectListItem 是项目列表中的单个条目。
type projectListItem struct {
	ID          int64  `json:"id"`
	ProjectSlug string `json:"project_slug"`
	ProjectCwd  string `json:"project_cwd"`
	FirstSeenAt string `json:"first_seen_at"`
	LastSeenAt  string `json:"last_seen_at"`
}

// --- helpers ---

// readProjectListResponse 读取并解析项目列表响应。
func readProjectListResponse(t *testing.T, resp *http.Response) projectListResponse {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result projectListResponse
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("parse JSON: %v, body: %s", err, b)
	}
	return result
}

// apiRegisterSessionWithCwd 通过 POST /api/session/register 注册会话，指定 project_cwd。
func apiRegisterSessionWithCwd(t *testing.T, env *testEnv, sessionID, machineID, os, projectSlug, projectCwd string) {
	t.Helper()
	body := fmt.Sprintf(
		`{"session_id":"%s","machine_id":"%s","os":"%s","project_slug":"%s","project_cwd":"%s","transcript_path":"/tmp/t.jsonl"}`,
		sessionID, machineID, os, projectSlug, projectCwd)
	resp := env.post(t, "/api/session/register", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("apiRegisterSessionWithCwd failed: session=%s status=%d body=%s", sessionID, resp.StatusCode, b)
	}
}

// seedProjectData 通过 session register 预置 BDD Background 中的 3 条项目数据。
// projects 表由 session register 的 upsert 逻辑填充：
//   - project-x  D:\code\project-x
//   - project-y  D:\code\project-y
//   - project-z  /Users/dev/project-z
func seedProjectData(t *testing.T, env *testEnv) {
	t.Helper()
	apiRegisterSessionWithCwd(t, env, "sess-px1", "user@host-1", "windows", "project-x", `D:\\code\\project-x`)
	apiRegisterSessionWithCwd(t, env, "sess-py1", "dev@host-2", "linux", "project-y", `D:\\code\\project-y`)
	apiRegisterSessionWithCwd(t, env, "sess-pz1", "admin@host-3", "macos", "project-z", "/Users/dev/project-z")
}

// --- BDD Scenario tests ---

// TestListProjects_GetAll 验证：获取所有项目列表。
// BDD: @positive Scenario: 获取所有项目列表
func TestListProjects_GetAll(t *testing.T) {
	env := setupTest(t)
	seedProjectData(t, env)

	resp := env.get(t, "/api/projects")
	result := readProjectListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Projects) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(result.Projects))
	}

	// 验证每个项目包含必要字段
	for _, p := range result.Projects {
		if p.ID == 0 {
			t.Error("expected non-zero id")
		}
		if p.ProjectSlug == "" {
			t.Error("expected non-empty project_slug")
		}
		if p.ProjectCwd == "" {
			t.Error("expected non-empty project_cwd")
		}
		if p.FirstSeenAt == "" {
			t.Error("expected non-empty first_seen_at")
		}
		if p.LastSeenAt == "" {
			t.Error("expected non-empty last_seen_at")
		}
	}
}

// TestListProjects_OrderedByLastSeenAt 验证：项目按 last_seen_at 倒序排列。
// BDD: @positive Scenario: 获取所有项目列表 — 项目应按 last_seen_at 倒序排列
func TestListProjects_OrderedByLastSeenAt(t *testing.T) {
	env := setupTest(t)

	// 按顺序注册 3 个项目
	apiRegisterSessionWithCwd(t, env, "sess-o1", "user@host", "linux", "project-x", "/a")
	apiRegisterSessionWithCwd(t, env, "sess-o2", "user@host", "linux", "project-y", "/b")
	apiRegisterSessionWithCwd(t, env, "sess-o3", "user@host", "linux", "project-z", "/c")

	// project-x 再次被访问，其 last_seen_at 应为最新
	apiRegisterSessionWithCwd(t, env, "sess-o1-later", "user@host", "linux", "project-x", "/a")

	resp := env.get(t, "/api/projects")
	result := readProjectListResponse(t, resp)

	if len(result.Projects) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(result.Projects))
	}
	// project-x 最后被访问过，应排在第一位
	if result.Projects[0].ProjectSlug != "project-x" {
		t.Errorf("expected first project to be project-x, got %s", result.Projects[0].ProjectSlug)
	}
}

// TestListProjects_EmptyResult 验证：无项目时返回空数组。
// BDD: @positive Scenario: 无项目时返回空数组
func TestListProjects_EmptyResult(t *testing.T) {
	env := setupTest(t)

	resp := env.get(t, "/api/projects")
	result := readProjectListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Projects) != 0 {
		t.Fatalf("expected empty projects array, got %d items", len(result.Projects))
	}
}

// TestListProjects_MethodNotAllowed 验证：使用非 GET 方法请求返回 405。
// BDD: @negative Scenario: 使用非 GET 方法请求
func TestListProjects_MethodNotAllowed(t *testing.T) {
	env := setupTest(t)

	resp := env.post(t, "/api/projects", "")
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
