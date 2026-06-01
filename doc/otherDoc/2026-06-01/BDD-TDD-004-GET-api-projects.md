# BDD + TDD: GET /api/projects 项目列表

> 接口: `GET /api/projects`
> 日期: 2026-06-01
> 状态: ❌ 未实现

---

## 一、接口定义

| 属性 | 值 |
|------|-----|
| 端点 | `/api/projects` |
| 方法 | GET |
| 功能 | 列出所有已注册的项目信息 |
| 查询参数 | 无 |
| 响应 | `ProjectsListResponse` |

---

## 二、BDD 测试用例

```gherkin
Feature: 列出所有项目
  作为监控用户
  我需要查看所有 Claude Code 工作过的项目
  以便了解项目分布情况

  Background:
    Given 后端服务已启动
    And 数据库已初始化
    And projects 表存在以下记录:
      | project_slug | project_cwd          | first_seen_at        | last_seen_at         |
      | project-x    | D:\code\project-x    | 2026-05-20T10:00:00Z | 2026-06-01T14:32:00Z |
      | project-y    | D:\code\project-y    | 2026-05-25T09:00:00Z | 2026-06-01T14:28:00Z |
      | project-z    | /Users/dev/project-z | 2026-05-28T08:00:00Z | 2026-05-30T12:00:00Z |

  @positive
  Scenario: 获取所有项目列表
    When 发送 GET 请求到 /api/projects
    Then 响应状态码应为 200
    And 响应体应包含 3 个项目
    And 每个项目应包含字段:
      | id | project_slug | project_cwd | first_seen_at | last_seen_at |
    And 项目应按 last_seen_at 倒序排列

  @positive
  Scenario: 无项目时返回空数组
    Given projects 表为空
    When 发送 GET 请求到 /api/projects
    Then 响应状态码应为 200
    And 响应体应包含 {"projects": []}

  @negative
  Scenario: 使用非 GET 方法请求
    When 发送 POST 请求到 /api/projects
    Then 响应状态码应为 405
    And 响应体应包含 "method_not_allowed"
```

---

## 三、TDD 开发步骤

### Step 1: 定义响应类型

```go
// internal/backend/api/response.go

// ProjectsListResponse 是项目列表的响应体。
type ProjectsListResponse struct {
	Projects []ProjectListItem `json:"projects"`
}

// ProjectListItem 是项目列表中的单个条目。
type ProjectListItem struct {
	ID          int64     `json:"id"`
	ProjectSlug string    `json:"project_slug"`
	ProjectCwd  string    `json:"project_cwd"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}
```

### Step 2: 定义 ProjectStore 接口

```go
// internal/backend/store/store.go

// ProjectStore 定义 Project 数据存储的接口。
type ProjectStore interface {
	ListProjects(ctx context.Context) ([]Project, error)
	GetProject(ctx context.Context, projectSlug string) (*Project, error)
}
```

### Step 3: 实现 SQLite 存储层

```go
// internal/backend/store/project_store.go

type sqliteProjectStore struct {
	db *sql.DB
}

func (s *sqliteProjectStore) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_slug, project_cwd, first_seen_at, last_seen_at 
		   FROM projects ORDER BY last_seen_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.ProjectSlug, &p.ProjectCwd, &p.FirstSeenAt, &p.LastSeenAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *sqliteProjectStore) GetProject(ctx context.Context, projectSlug string) (*Project, error) {
	var p Project
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_slug, project_cwd, first_seen_at, last_seen_at FROM projects WHERE project_slug = ?`,
		projectSlug,
	).Scan(&p.ID, &p.ProjectSlug, &p.ProjectCwd, &p.FirstSeenAt, &p.LastSeenAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &p, nil
}
```

### Step 4: 扩展 Store 聚合接口

```go
// internal/backend/store/store.go

type Store interface {
	Issues() IssueStore
	Sessions() SessionStore
	Machines() MachineStore
	Projects() ProjectStore // 新增
	Close() error
}
```

### Step 5: 实现 Service 层

```go
// internal/backend/service/project_service.go

package service

import (
	"context"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type ProjectService struct {
	store store.ProjectStore
}

func NewProjectService(s store.ProjectStore) *ProjectService {
	return &ProjectService{store: s}
}

func (svc *ProjectService) List(ctx context.Context) ([]store.Project, error) {
	logger.Debug("svc.project", "List: all projects")
	return svc.store.ListProjects(ctx)
}

func (svc *ProjectService) Get(ctx context.Context, projectSlug string) (*store.Project, error) {
	logger.Debug("svc.project", "Get: slug=%s", projectSlug)
	return svc.store.GetProject(ctx, projectSlug)
}
```

### Step 6: 实现 Handler 层

```go
// internal/backend/api/project_handler.go

package api

import (
	"net/http"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type ProjectHandler struct {
	svc *service.ProjectService
}

func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	logger.Debug("api.project", "GET /api/projects")

	projects, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list projects")
		return
	}

	if projects == nil {
		projects = []store.Project{}
	}

	items := make([]ProjectListItem, len(projects))
	for i, p := range projects {
		items[i] = ProjectListItem{
			ID:          p.ID,
			ProjectSlug: p.ProjectSlug,
			ProjectCwd:  p.ProjectCwd,
			FirstSeenAt: p.FirstSeenAt,
			LastSeenAt:  p.LastSeenAt,
		}
	}

	writeJSON(w, http.StatusOK, ProjectsListResponse{Projects: items})
}
```

### Step 7: 注册路由

```go
// internal/backend/api/router.go

type Handlers struct {
	Issue   *IssueHandler
	Session *SessionHandler
	Proxy   *ProxyHandler
	Machine *MachineHandler
	Project *ProjectHandler // 新增
}

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	// ... 已有路由
	mux.HandleFunc("/api/projects", h.Project.List) // 新增
	return mux
}
```

---

## 四、Todo List

- [ ] Step 1: 定义 ProjectsListResponse 和 ProjectListItem 响应类型
- [ ] Step 2: 定义 ProjectStore 接口
- [ ] Step 3: 实现 sqliteProjectStore（ListProjects + GetProject）
- [ ] Step 4: 扩展 Store 聚合接口添加 Projects()
- [ ] Step 5: 实现 ProjectService
- [ ] Step 6: 实现 ProjectHandler
- [ ] Step 7: 在 router.go 中注册 /api/projects 路由
- [ ] Step 8: 编写单元测试 project_api_test.go
- [ ] Step 9: 运行 BDD 测试验证
- [ ] Step 10: 更新 API 文档

---

*文档结束*
