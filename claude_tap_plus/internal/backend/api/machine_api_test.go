// Package api_test 包含 GET /api/machines 接口的 BDD + TDD 验收测试。
// 覆盖：获取全部、按 OS/hostname 过滤、空列表、方法限制。
// 所有测试数据均通过 POST /api/session/register 产生（该接口 upsert machines 表）。
package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// --- response types for machine API ---

// machineListResponse 是 GET /api/machines 的响应结构。
type machineListResponse struct {
	Machines []machineListItem `json:"machines"`
}

// machineListItem 是机器列表中的单个条目。
type machineListItem struct {
	ID          int64  `json:"id"`
	MachineID   string `json:"machine_id"`
	OS          string `json:"os"`
	Hostname    string `json:"hostname"`
	Username    string `json:"username"`
	FirstSeenAt string `json:"first_seen_at"`
	LastSeenAt  string `json:"last_seen_at"`
}

// --- helpers ---

// readMachineListResponse 读取并解析机器列表响应。
func readMachineListResponse(t *testing.T, resp *http.Response) machineListResponse {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result machineListResponse
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("parse JSON: %v, body: %s", err, b)
	}
	return result
}

// apiRegisterSession 通过 POST /api/session/register 注册会话，
// 该接口会 upsert machines 表，用于产生机器测试数据。
func apiRegisterSession(t *testing.T, env *testEnv, sessionID, machineID, os, projectSlug string) {
	t.Helper()
	body := fmt.Sprintf(
		`{"session_id":"%s","machine_id":"%s","os":"%s","project_slug":"%s","project_cwd":"/tmp","transcript_path":"/tmp/t.jsonl"}`,
		sessionID, machineID, os, projectSlug)
	resp := env.post(t, "/api/session/register", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("apiRegisterSession failed: session=%s machine=%s status=%d", sessionID, machineID, resp.StatusCode)
	}
}

// seedMachineData 通过 session register 预置 BDD Background 中的 3 条机器数据。
// machines 表由 session register 的 upsert 逻辑填充：
//   - user@host-1  windows  host-1
//   - dev@host-2   linux    host-2
//   - admin@host-3 macos    host-3
func seedMachineData(t *testing.T, env *testEnv) {
	t.Helper()
	apiRegisterSession(t, env, "sess-m1", "user@host-1", "windows", "proj/a")
	apiRegisterSession(t, env, "sess-m2", "dev@host-2", "linux", "proj/b")
	apiRegisterSession(t, env, "sess-m3", "admin@host-3", "macos", "proj/c")
}

// --- BDD Scenario tests ---

// TestListMachines_GetAll 验证：获取所有机器列表。
// BDD: @positive Scenario: 获取所有机器列表
func TestListMachines_GetAll(t *testing.T) {
	env := setupTest(t)
	seedMachineData(t, env)

	resp := env.get(t, "/api/machines")
	result := readMachineListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Machines) != 3 {
		t.Fatalf("expected 3 machines, got %d", len(result.Machines))
	}

	// 验证每个机器包含必要字段
	for _, m := range result.Machines {
		if m.ID == 0 {
			t.Error("expected non-zero id")
		}
		if m.MachineID == "" {
			t.Error("expected non-empty machine_id")
		}
		if m.OS == "" {
			t.Error("expected non-empty os")
		}
		if m.Hostname == "" {
			t.Error("expected non-empty hostname")
		}
		if m.Username == "" {
			t.Error("expected non-empty username")
		}
		if m.FirstSeenAt == "" {
			t.Error("expected non-empty first_seen_at")
		}
	}
}

// TestListMachines_FilterByOS 验证：按操作系统过滤。
// BDD: @positive Scenario: 按操作系统过滤
func TestListMachines_FilterByOS(t *testing.T) {
	env := setupTest(t)
	seedMachineData(t, env)

	resp := env.get(t, "/api/machines?os=windows")
	result := readMachineListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Machines) != 1 {
		t.Fatalf("expected 1 machine, got %d", len(result.Machines))
	}
	if result.Machines[0].MachineID != "user@host-1" {
		t.Errorf("expected machine_id=user@host-1, got %s", result.Machines[0].MachineID)
	}
}

// TestListMachines_FilterByHostname 验证：按主机名过滤。
// BDD: @positive Scenario: 按主机名过滤
func TestListMachines_FilterByHostname(t *testing.T) {
	env := setupTest(t)
	seedMachineData(t, env)

	resp := env.get(t, "/api/machines?hostname=host-2")
	result := readMachineListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Machines) != 1 {
		t.Fatalf("expected 1 machine, got %d", len(result.Machines))
	}
	if result.Machines[0].MachineID != "dev@host-2" {
		t.Errorf("expected machine_id=dev@host-2, got %s", result.Machines[0].MachineID)
	}
}

// TestListMachines_EmptyResult 验证：无机器时返回空数组。
// BDD: @positive Scenario: 无机器时返回空数组
func TestListMachines_EmptyResult(t *testing.T) {
	env := setupTest(t)

	resp := env.get(t, "/api/machines")
	result := readMachineListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Machines) != 0 {
		t.Fatalf("expected empty machines array, got %d items", len(result.Machines))
	}
}

// TestListMachines_MethodNotAllowed 验证：使用非 GET 方法请求返回 405。
// BDD: @negative Scenario: 使用非 GET 方法请求
func TestListMachines_MethodNotAllowed(t *testing.T) {
	env := setupTest(t)

	resp := env.post(t, "/api/machines", "")
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

// --- TDD additional edge-case tests ---

// TestListMachines_FilterByNonexistentOS 验证：过滤不存在的 OS 返回空。
func TestListMachines_FilterByNonexistentOS(t *testing.T) {
	env := setupTest(t)
	seedMachineData(t, env)

	resp := env.get(t, "/api/machines?os=freebsd")
	result := readMachineListResponse(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(result.Machines) != 0 {
		t.Fatalf("expected 0 machines, got %d", len(result.Machines))
	}
}

// TestListMachines_MachineFieldsFromSessionRegister 验证：
// session register 的 machine_id 格式为 username@hostname，
// 注册后 machines 表正确解析出 username 和 hostname。
func TestListMachines_MachineFieldsFromSessionRegister(t *testing.T) {
	env := setupTest(t)
	apiRegisterSession(t, env, "sess-test", "alice@myhost", "linux", "proj/test")

	resp := env.get(t, "/api/machines")
	result := readMachineListResponse(t, resp)

	if len(result.Machines) != 1 {
		t.Fatalf("expected 1 machine, got %d", len(result.Machines))
	}
	m := result.Machines[0]
	if m.MachineID != "alice@myhost" {
		t.Errorf("expected machine_id=alice@myhost, got %s", m.MachineID)
	}
	if m.OS != "linux" {
		t.Errorf("expected os=linux, got %s", m.OS)
	}
	if m.Hostname != "myhost" {
		t.Errorf("expected hostname=myhost, got %s", m.Hostname)
	}
	if m.Username != "alice" {
		t.Errorf("expected username=alice, got %s", m.Username)
	}
	if m.LastSeenAt == "" {
		t.Error("expected non-empty last_seen_at")
	}
}
