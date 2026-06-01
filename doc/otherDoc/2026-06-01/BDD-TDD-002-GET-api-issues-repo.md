# BDD + TDD: GET /api/issues?repo=xxx 按仓库列出 Issue

> 接口: `GET /api/issues?repo={repo_full_name}`
> 日期: 2026-06-01
> 状态: ❌ 未实现

---

## 一、接口定义

| 属性 | 值 |
|------|-----|
| 端点 | `/api/issues` |
| 方法 | GET |
| 查询参数 | `repo` (必填) — 仓库全名，如 `xiaoheiDTF/claude-hk` |
| 功能 | 按仓库列出该仓库下所有 Issue |
| 响应 | `IssuesListResponse` |

---

## 二、BDD 测试用例

```gherkin
Feature: 按仓库列出 Issue
  作为监控用户
  我需要查看指定仓库下所有 Issue 的列表
  以便了解该仓库的 Issue 处理情况

  Background:
    Given 后端服务已启动
    And 数据库已初始化
    And issue_claims 表存在以下记录:
      | repo_full_name       | issue_number | issue_title      | status  | session_id   |
      | xiaoheiDTF/claude-hk | 42           | Fix auth bug     | idle    |              |
      | xiaoheiDTF/claude-hk | 45           | Add test cov     | claimed | sess-abc123  |
      | xiaoheiDTF/claude-hk | 38           | Refactor API     | fixing  | sess-def456  |
      | xiaoheiDTF/claude-hk | 51           | Update docs      | merged  | sess-abc123  |
      | other/repo           | 10           | Update README    | idle    |              |

  @positive
  Scenario: 按仓库列出所有 Issue
    When 发送 GET 请求到 /api/issues?repo=xiaoheiDTF/claude-hk
    Then 响应状态码应为 200
    And 响应体应包含 4 个 Issue
    And 所有 Issue 的 repo_full_name 应为 "xiaoheiDTF/claude-hk"
    And Issue 应按 updated_at 倒序排列

  @positive
  Scenario: 按仓库和状态组合过滤
    When 发送 GET 请求到 /api/issues?repo=xiaoheiDTF/claude-hk&status=idle
    Then 响应状态码应为 200
    And 响应体应包含 1 个 Issue
    And 该 Issue 的 number 应为 42

  @positive
  Scenario: 仓库无 Issue 时返回空数组
    When 发送 GET 请求到 /api/issues?repo=not-exist/repo
    Then 响应状态码应为 200
    And 响应体应包含 {"issues": [], "total": 0}

  @negative
  Scenario: 缺少 repo 参数
    When 发送 GET 请求到 /api/issues
    Then 响应状态码应为 400
    And 响应体应包含错误码 "invalid_request"
    And 响应体应包含 "repo parameter is required"

  @negative
  Scenario: 使用非 GET 方法请求
    When 发送 POST 请求到 /api/issues?repo=xiaoheiDTF/claude-hk
    Then 响应状态码应为 405
    And 响应体应包含 "method_not_allowed"
```

---

## 三、TDD 开发步骤

### Step 1: 复用 IssueFilter 和 IssueListItem（已在 /api/issues 中定义）

### Step 2: 修改 Handler 支持 repo 必填校验

```go
// internal/backend/api/issue_handler.go

// ListIssues 处理获取 Issue 列表的请求（支持 repo 必填模式）。
func (h *IssueHandler) ListIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	repo := r.URL.Query().Get("repo")
	
	// 当请求 /api/issues 且无 repo 参数时，返回所有 Issue
	// 当请求 /api/issues?repo=xxx 时，按仓库过滤
	var filter store.IssueFilter
	if repo != "" {
		filter.RepoFullName = &repo
	}
	
	// 其他过滤参数...
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}
	if v := r.URL.Query().Get("session_id"); v != "" {
		filter.SessionID = &v
	}

	logger.Debug("api.issue", "GET /api/issues repo=%s", repo)

	items, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list issues")
		return
	}

	if items == nil {
		items = []store.IssueListItem{}
	}

	writeJSON(w, http.StatusOK, IssuesListResponse{
		Issues: items,
		Total:  total,
	})
}
```

### Step 3: 实现单元测试

```go
// tests/backend/issue_list_repo_api_test.go

package backend

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListIssuesByRepo(t *testing.T) {
	// 复用 TestListIssues 的初始化逻辑
	
	t.Run("list by repo", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/issues?repo=xiaoheiDTF/claude-hk", nil)
		rec := httptest.NewRecorder()
		handler.ListIssues(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		// 断言返回 4 个 Issue...
	})

	t.Run("repo not found returns empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/issues?repo=not-exist/repo", nil)
		rec := httptest.NewRecorder()
		handler.ListIssues(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		// 断言返回空数组...
	})
}
```

---

## 四、Todo List

- [ ] Step 1: 复用 /api/issues 的 IssueFilter、IssueListItem 类型
- [ ] Step 2: 修改 ListIssues Handler 支持 repo 参数过滤
- [ ] Step 3: 编写按仓库过滤的单元测试
- [ ] Step 4: 运行 BDD 测试验证功能
- [ ] Step 5: 更新 API 文档

---

*文档结束*
