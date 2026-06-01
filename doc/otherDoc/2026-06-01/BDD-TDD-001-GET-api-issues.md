# BDD + TDD: GET /api/issues 列出所有 Issue

> 接口: `GET /api/issues`
> 日期: 2026-06-01
> 状态: ❌ 未实现

---

## 一、接口定义

| 属性 | 值 |
|------|-----|
| 端点 | `/api/issues` |
| 方法 | GET |
| 功能 | 列出数据库中所有 Issue 记录，支持过滤和分页 |
| 查询参数 | `repo` (可选), `status` (可选), `session_id` (可选), `page` (可选), `page_size` (可选) |
| 响应 | `IssuesListResponse` |

---

## 二、BDD 测试用例

```gherkin
Feature: 列出所有 Issue
  作为监控用户
  我需要查看系统中所有 Issue 的列表
  以便了解 Issue 的整体分布和状态

  Background:
    Given 后端服务已启动
    And 数据库已初始化
    And issue_claims 表存在以下记录:
      | repo_full_name       | issue_number | issue_title      | status  | session_id   |
      | xiaoheiDTF/claude-hk | 42           | Fix auth bug     | idle    |              |
      | xiaoheiDTF/claude-hk | 45           | Add test cov     | claimed | sess-abc123  |
      | xiaoheiDTF/claude-hk | 38           | Refactor API     | fixing  | sess-def456  |
      | other/repo           | 10           | Update README    | idle    |              |
      | other/repo           | 11           | Fix CI           | merged  | sess-abc123  |

  @positive
  Scenario: 获取所有 Issue 列表
    When 发送 GET 请求到 /api/issues
    Then 响应状态码应为 200
    And 响应体应包含 5 个 Issue
    And 每个 Issue 应包含字段:
      | id | repo_full_name | issue_number | issue_title | status | session_id | claimed_at | updated_at |

  @positive
  Scenario: 按仓库过滤 Issue
    When 发送 GET 请求到 /api/issues?repo=xiaoheiDTF/claude-hk
    Then 响应状态码应为 200
    And 响应体应包含 3 个 Issue
    And 所有 Issue 的 repo_full_name 应为 "xiaoheiDTF/claude-hk"

  @positive
  Scenario: 按状态过滤 Issue
    When 发送 GET 请求到 /api/issues?status=idle
    Then 响应状态码应为 200
    And 响应体应包含 2 个 Issue
    And 所有 Issue 的 status 应为 "idle"

  @positive
  Scenario: 按 session_id 过滤 Issue
    Given 存在 session "sess-abc123"
    When 发送 GET 请求到 /api/issues?session_id=sess-abc123
    Then 响应状态码应为 200
    And 响应体应包含 2 个 Issue
    And 所有 Issue 的 session_id 应为 "sess-abc123"

  @positive
  Scenario: 组合过滤条件
    When 发送 GET 请求到 /api/issues?repo=xiaoheiDTF/claude-hk&status=idle
    Then 响应状态码应为 200
    And 响应体应包含 1 个 Issue
    And 该 Issue 的 repo_full_name 应为 "xiaoheiDTF/claude-hk"
    And 该 Issue 的 status 应为 "idle"

  @positive
  Scenario: 分页查询
    When 发送 GET 请求到 /api/issues?page=1&page_size=2
    Then 响应状态码应为 200
    And 响应体应包含 2 个 Issue
    And 响应体应包含分页信息:
      """
      {
        "total": 5,
        "page": 1,
        "page_size": 2,
        "total_pages": 3
      }
      """

  @positive
  Scenario: 无 Issue 时返回空数组
    Given issue_claims 表为空
    When 发送 GET 请求到 /api/issues
    Then 响应状态码应为 200
    And 响应体应包含 {"issues": [], "total": 0}

  @negative
  Scenario: 使用非 GET 方法请求
    When 发送 POST 请求到 /api/issues
    Then 响应状态码应为 405
    And 响应体应包含 "method_not_allowed"
```

---

## 三、TDD 开发步骤

### Step 1: 定义响应类型

```go
// internal/backend/api/response.go

// IssuesListResponse 是 Issue 列表的响应体。
type IssuesListResponse struct {
	Issues     []IssueListItem `json:"issues"`      // Issue 列表
	Total      int             `json:"total"`       // 总数量
	Page       int             `json:"page"`        // 当前页码
	PageSize   int             `json:"page_size"`   // 每页大小
	TotalPages int             `json:"total_pages"` // 总页数
}

// IssueListItem 是 Issue 列表中的单个条目。
type IssueListItem struct {
	ID           int64      `json:"id"`
	RepoFullName string     `json:"repo_full_name"`
	IssueNumber  int        `json:"issue_number"`
	IssueTitle   string     `json:"issue_title"`
	Status       string     `json:"status"`
	SessionID    *string    `json:"session_id"`
	ClaimedAt    *time.Time `json:"claimed_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
```

### Step 2: 定义过滤条件类型

```go
// internal/backend/store/store.go

// IssueFilter 是 Issue 列表的过滤条件。
type IssueFilter struct {
	RepoFullName *string // 按仓库过滤
	Status       *string // 按状态过滤
	SessionID    *string // 按 session 过滤
	Page         int     // 页码，默认 1
	PageSize     int     // 每页大小，默认 20
}
```

### Step 3: 扩展 IssueStore 接口

```go
// internal/backend/store/store.go

// IssueStore 定义 Issue 数据存储的接口。
type IssueStore interface {
	CheckIssues(ctx context.Context, repo string, numbers []int) ([]IssueCheckResult, error)
	ClaimIssue(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*ClaimResult, error)
	UpdateIssueStatus(ctx context.Context, repo string, number int, sessionID string, newStatus string) (*UpdateStatusResult, error)
	ReleaseIssue(ctx context.Context, repo string, number int, sessionID string) (bool, error)
	ReleaseSessionIssues(ctx context.Context, sessionID string) ([]int, error)
	ListIssues(ctx context.Context, filter IssueFilter) ([]IssueListItem, int, error) // 新增
}
```

### Step 4: 实现 SQLite 存储层

```go
// internal/backend/store/issue_store.go

// ListIssues 获取 Issue 列表，支持过滤和分页。
func (s *sqliteIssueStore) ListIssues(ctx context.Context, filter IssueFilter) ([]IssueListItem, int, error) {
	// 构建 WHERE 条件
	where := "WHERE 1=1"
	args := []any{}

	if filter.RepoFullName != nil {
		where += " AND repo_full_name = ?"
		args = append(args, *filter.RepoFullName)
	}
	if filter.Status != nil {
		where += " AND status = ?"
		args = append(args, *filter.Status)
	}
	if filter.SessionID != nil {
		where += " AND session_id = ?"
		args = append(args, *filter.SessionID)
	}

	// 查询总数
	var total int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM issue_claims "+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count issues: %w", err)
	}

	// 分页参数
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 查询数据
	query := fmt.Sprintf(
		`SELECT id, repo_full_name, issue_number, issue_title, status,
		        session_id, claimed_at, updated_at
		   FROM issue_claims %s
		  ORDER BY updated_at DESC
		  LIMIT ? OFFSET ?`, where)
	args = append(args, pageSize, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list issues: %w", err)
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
			return nil, 0, fmt.Errorf("scan issue: %w", err)
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

	return items, total, rows.Err()
}
```

### Step 5: 实现 Service 层

```go
// internal/backend/service/issue_service.go

// List 获取 Issue 列表。
func (svc *IssueService) List(ctx context.Context, filter store.IssueFilter) ([]store.IssueListItem, int, error) {
	logger.Debug("svc.issue", "List: filter applied")
	return svc.store.ListIssues(ctx, filter)
}
```

### Step 6: 实现 Handler 层

```go
// internal/backend/api/issue_handler.go

// ListIssues 处理获取 Issue 列表的请求。
func (h *IssueHandler) ListIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	// 解析查询参数
	var filter store.IssueFilter
	if v := r.URL.Query().Get("repo"); v != "" {
		filter.RepoFullName = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}
	if v := r.URL.Query().Get("session_id"); v != "" {
		filter.SessionID = &v
	}
	// 分页参数解析...

	logger.Debug("api.issue", "GET /api/issues filter=%+v", filter)

	items, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list issues")
		return
	}

	if items == nil {
		items = []store.IssueListItem{}
	}

	// 转换为响应格式
	respItems := make([]IssueListItem, len(items))
	for i, item := range items {
		respItems[i] = IssueListItem{
			ID:           item.ID,
			RepoFullName: item.RepoFullName,
			IssueNumber:  item.IssueNumber,
			IssueTitle:   item.IssueTitle,
			Status:       item.Status,
			SessionID:    item.SessionID,
			ClaimedAt:    item.ClaimedAt,
			UpdatedAt:    item.UpdatedAt,
		}
	}

	writeJSON(w, http.StatusOK, IssuesListResponse{
		Issues: respItems,
		Total:  total,
	})
}
```

### Step 7: 注册路由

```go
// internal/backend/api/router.go

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", Health)

	// Issue 相关路由。
	mux.HandleFunc("/api/issue/check", h.Issue.CheckIssues)
	mux.HandleFunc("/api/issue/claim", h.Issue.ClaimIssue)
	mux.HandleFunc("/api/issue/release", h.Issue.ReleaseIssue)
	mux.HandleFunc("/api/issue/release-session", h.Issue.ReleaseSession)
	mux.HandleFunc("/api/issue/status", h.Issue.UpdateStatus)
	mux.HandleFunc("/api/issues", h.Issue.ListIssues) // 新增

	// ... 其他路由
}
```

---

## 四、单元测试

```go
// tests/backend/issue_list_api_test.go

package backend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/api"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
)

func TestListIssues(t *testing.T) {
	// 初始化内存数据库
	db, _ := store.NewSQLiteStore(":memory:")
	defer db.Close()

	issueSvc := service.NewIssueService(db.Issues())
	handler := api.NewIssueHandler(issueSvc)

	t.Run("get all issues", func(t *testing.T) {
		// 准备数据
		ctx := context.Background()
		db.Issues().ClaimIssue(ctx, "repo/test", 1, "sess-1", "Issue 1")
		db.Issues().ClaimIssue(ctx, "repo/test", 2, "sess-2", "Issue 2")

		// 发送请求
		req := httptest.NewRequest(http.MethodGet, "/api/issues", nil)
		rec := httptest.NewRecorder()
		handler.ListIssues(rec, req)

		// 断言
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		// 解析响应并断言...
	})

	t.Run("filter by repo", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/issues?repo=repo/test", nil)
		rec := httptest.NewRecorder()
		handler.ListIssues(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/issues", nil)
		rec := httptest.NewRecorder()
		handler.ListIssues(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rec.Code)
		}
	})
}
```

---

## 五、Todo List

- [ ] Step 1: 定义 IssuesListResponse 和 IssueListItem 响应类型
- [ ] Step 2: 定义 IssueFilter 过滤条件类型
- [ ] Step 3: 扩展 IssueStore 接口，添加 ListIssues 方法
- [ ] Step 4: 实现 sqliteIssueStore.ListIssues SQLite 查询
- [ ] Step 5: 实现 IssueService.List 业务逻辑层
- [ ] Step 6: 实现 IssueHandler.ListIssues HTTP 处理器
- [ ] Step 7: 在 router.go 中注册 /api/issues 路由
- [ ] Step 8: 编写单元测试 issue_list_api_test.go
- [ ] Step 9: 运行 BDD 测试验证功能
- [ ] Step 10: 更新 API 文档

---

*文档结束*
