# BDD + TDD: GET /api/session/{id}/issues 会话关联 Issue

> 接口: `GET /api/session/{id}/issues`
> 日期: 2026-06-01
> 状态: ❌ 未实现

---

## 一、接口定义

| 属性 | 值 |
|------|-----|
| 端点 | `/api/session/{id}/issues` |
| 方法 | GET |
| 功能 | 获取指定会话领取的所有 Issue |
| 路径参数 | `id` — session_id |
| 响应 | `SessionIssuesResponse` |

---

## 二、BDD 测试用例

```gherkin
Feature: 获取会话关联的 Issue
  作为监控用户
  我需要查看某个会话领取的所有 Issue
  以便了解该会话的工作内容

  Background:
    Given 后端服务已启动
    And 数据库已初始化
    And 存在会话 "session-abc123"
    And issue_claims 表存在以下记录:
      | repo_full_name       | issue_number | issue_title      | status  | session_id    |
      | xiaoheiDTF/claude-hk | 42           | Fix auth bug     | claimed | session-abc123|
      | xiaoheiDTF/claude-hk | 45           | Add test cov     | fixing  | session-abc123|
      | other/repo           | 10           | Update README    | claimed | session-def456|

  @positive
  Scenario: 获取会话领取的所有 Issue
    When 发送 GET 请求到 /api/session/session-abc123/issues
    Then 响应状态码应为 200
    And 响应体应包含 2 个 Issue
    And 所有 Issue 的 session_id 应为 "session-abc123"
    And Issue #42 的 status 应为 "claimed"
    And Issue #45 的 status 应为 "fixing"

  @positive
  Scenario: 会话无领取 Issue
    Given 存在会话 "session-empty"
    And 该会话未领取任何 Issue
    When 发送 GET 请求到 /api/session/session-empty/issues
    Then 响应状态码应为 200
    And 响应体应包含 {"issues": []}

  @negative
  Scenario: 获取不存在的会话
    When 发送 GET 请求到 /api/session/session-not-exist/issues
    Then 响应状态码应为 404
    And 响应体应包含错误码 "not_found"

  @negative
  Scenario: 使用非 GET 方法请求
    When 发送 POST 请求到 /api/session/session-abc123/issues
    Then 响应状态码应为 405
    And 响应体应包含 "method_not_allowed"
```

---

## 三、TDD 开发步骤

### Step 1: 定义响应类型

```go
// internal/backend/api/response.go

// SessionIssuesResponse 是会话关联 Issue 的响应体。
type SessionIssuesResponse struct {
	Issues []IssueListItem `json:"issues"`
}
```

### Step 2: 扩展 IssueStore 接口

```go
// internal/backend/store/store.go

type IssueStore interface {
	// ... 已有方法
	ListIssuesBySession(ctx context.Context, sessionID string) ([]IssueListItem, error) // 新增
}
```

### Step 3: 实现 SQLite 存储层

```go
// internal/backend/store/issue_store.go

// ListIssuesBySession 获取指定 session 领取的所有 Issue。
func (s *sqliteIssueStore) ListIssuesBySession(ctx context.Context, sessionID string) ([]IssueListItem, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, repo_full_name, issue_number, issue_title, status,
		        session_id, claimed_at, updated_at
		   FROM issue_claims
		  WHERE session_id = ?
		  ORDER BY updated_at DESC`,
		sessionID)
	if err != nil {
		return nil, fmt.Errorf("list issues by session: %w", err)
	}
	defer rows.Close()

	var items []IssueListItem
	for rows.Next() {
		var item IssueListItem
		var sessionID sql.NullString
		var claimedAt sql.NullString
		if err := rows.Scan(
			&item.ID, &item.RepoFullName, &item.IssueNumber, &item.IssueTitle,
			&item.Status, &sessionID, &claimedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan issue: %w", err)
		}
		if sessionID.Valid {
			item.SessionID = &sessionID.String
		}
		if claimedAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", claimedAt.String)
			item.ClaimedAt = &t
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
```

### Step 4: 实现 Service 层

```go
// internal/backend/service/issue_service.go

// ListBySession 获取指定 session 的 Issue 列表。
func (svc *IssueService) ListBySession(ctx context.Context, sessionID string) ([]store.IssueListItem, error) {
	logger.Debug("svc.issue", "ListBySession: session=%s", sessionID)
	return svc.store.ListIssuesBySession(ctx, sessionID)
}
```

### Step 5: 实现 Handler 层

```go
// internal/backend/api/session_handler.go

// GetIssues 处理获取会话关联 Issue 的请求。
func (h *SessionHandler) GetIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	// 从 URL 路径中提取 session_id
	// 路径格式: /api/session/{id}/issues
	path := r.URL.Path
	prefix := "/api/session/"
	suffix := "/issues"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid path")
		return
	}
	sessionID := path[len(prefix) : len(path)-len(suffix)]
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session_id is required")
		return
	}

	logger.Debug("api.session", "GET /api/session/%s/issues", sessionID)

	// 先检查会话是否存在
	sess, err := h.svc.Get(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get session")
		return
	}
	if sess == nil {
		writeError(w, http.StatusNotFound, "not_found", "session not found")
		return
	}

	// 获取该会话的 Issue 列表
	// 注意：这里需要 IssueService，可以通过依赖注入获取
	// 为简化，假设 SessionHandler 也持有 IssueService
	items, err := h.issueSvc.ListBySession(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list issues")
		return
	}

	if items == nil {
		items = []store.IssueListItem{}
	}

	writeJSON(w, http.StatusOK, SessionIssuesResponse{Issues: items})
}
```

### Step 6: 修改 SessionHandler 结构体

```go
// internal/backend/api/session_handler.go

type SessionHandler struct {
	svc     *service.SessionService
	issueSvc *service.IssueService // 新增
}

func NewSessionHandler(svc *service.SessionService, issueSvc *service.IssueService) *SessionHandler {
	return &SessionHandler{svc: svc, issueSvc: issueSvc}
}
```

### Step 7: 注册路由

```go
// internal/backend/api/router.go

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	// ... 已有路由
	
	// Session 关联路由
	mux.HandleFunc("/api/session/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/issues") {
			h.Session.GetIssues(w, r)
		} else {
			h.Session.Get(w, r)
		}
	})
	
	return mux
}
```

---

## 四、Todo List

- [ ] Step 1: 定义 SessionIssuesResponse 响应类型
- [ ] Step 2: 扩展 IssueStore 接口添加 ListIssuesBySession
- [ ] Step 3: 实现 sqliteIssueStore.ListIssuesBySession
- [ ] Step 4: 实现 IssueService.ListBySession
- [ ] Step 5: 修改 SessionHandler 注入 IssueService
- [ ] Step 6: 实现 SessionHandler.GetIssues
- [ ] Step 7: 在 router.go 中注册 /api/session/{id}/issues 路由
- [ ] Step 8: 编写单元测试 session_issues_api_test.go
- [ ] Step 9: 运行 BDD 测试验证
- [ ] Step 10: 更新 API 文档

---

*文档结束*
