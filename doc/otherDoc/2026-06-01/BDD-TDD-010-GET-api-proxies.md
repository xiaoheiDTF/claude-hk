# BDD + TDD: GET /api/proxies 代理列表

> 接口: `GET /api/proxies`
> 日期: 2026-06-01
> 状态: ❌ 未实现

---

## 一、接口定义

| 属性 | 值 |
|------|-----|
| 端点 | `/api/proxies` |
| 方法 | GET |
| 功能 | 获取所有已注册代理的列表 |
| 查询参数 | `status` (可选), `project` (可选) |
| 响应 | `ProxiesResponse` |

---

## 二、BDD 测试用例

```gherkin
Feature: 获取代理列表
  作为监控用户
  我需要查看所有已注册的代理
  以便了解系统代理分布情况

  Background:
    Given 后端服务已启动
    And 数据库已初始化
    And 已注册以下代理:
      | proxy_id | project_slug | status  | registered_at         |
      | proxy-1  | project-a    | active  | 2026-06-01T10:00:00Z  |
      | proxy-2  | project-b    | active  | 2026-06-01T11:00:00Z  |
      | proxy-3  | project-a    | offline | 2026-06-01T09:00:00Z  |

  @positive
  Scenario: 获取所有代理列表
    When 发送 GET 请求到 /api/proxies
    Then 响应状态码应为 200
    And 响应体应包含 3 个代理
    And 每个代理应包含:
      | proxy_id | project_slug | status | registered_at |

  @positive
  Scenario: 按状态过滤代理
    When 发送 GET 请求到 /api/proxies?status=active
    Then 响应状态码应为 200
    And 响应体应包含 2 个代理
    And 所有代理的 status 应为 "active"

  @positive
  Scenario: 按项目过滤代理
    When 发送 GET 请求到 /api/proxies?project=project-a
    Then 响应状态码应为 200
    And 响应体应包含 2 个代理
    And 所有代理的 project_slug 应为 "project-a"

  @positive
  Scenario: 组合过滤代理
    When 发送 GET 请求到 /api/proxies?status=active&project=project-a
    Then 响应状态码应为 200
    And 响应体应包含 1 个代理
    And 该代理的 proxy_id 应为 "proxy-1"

  @positive
  Scenario: 无代理时返回空列表
    Given 数据库中无代理记录
    When 发送 GET 请求到 /api/proxies
    Then 响应状态码应为 200
    And 响应体应包含 {"proxies": []}

  @negative
  Scenario: 使用非 GET 方法请求
    When 发送 POST 请求到 /api/proxies
    Then 响应状态码应为 405
    And 响应体应包含 "method_not_allowed"
```

---

## 三、TDD 开发步骤

### Step 1: 定义响应类型

```go
// internal/backend/api/response.go

// ProxiesResponse 是代理列表的响应体。
type ProxiesResponse struct {
	Proxies []ProxyItem `json:"proxies"`
	Total   int         `json:"total"`
}

// ProxyItem 是单个代理的信息。
type ProxyItem struct {
	ProxyID      string    `json:"proxy_id"`
	ProjectSlug  string    `json:"project_slug"`
	Status       string    `json:"status"`
	RegisteredAt time.Time `json:"registered_at"`
	LastPingAt   time.Time `json:"last_ping_at,omitempty"`
}
```

### Step 2: 定义代理存储接口

```go
// internal/backend/store/store.go

// ProxyStore 接口定义代理 CRUD 操作。
type ProxyStore interface {
	RegisterProxy(ctx context.Context, p Proxy) error
	UnregisterProxy(ctx context.Context, proxyID string) error
	ListProxies(ctx context.Context, filter ProxyFilter) ([]Proxy, error)
	GetProxy(ctx context.Context, proxyID string) (*Proxy, error)
}

// ProxyFilter 是代理查询的过滤条件。
type ProxyFilter struct {
	Status  *string // active/offline
	Project *string // project_slug
}

// Proxy 是代理领域模型。
type Proxy struct {
	ProxyID      string    `json:"proxy_id"`
	ProjectSlug  string    `json:"project_slug"`
	Status       string    `json:"status"`
	RegisteredAt time.Time `json:"registered_at"`
	LastPingAt   time.Time `json:"last_ping_at"`
}
```

### Step 3: 实现代理存储层

```go
// internal/backend/store/proxy_store.go

package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type proxyStore struct {
	db *sql.DB
}

func newProxyStore(db *sql.DB) *proxyStore {
	return &proxyStore{db: db}
}

func (s *proxyStore) RegisterProxy(ctx context.Context, p Proxy) error {
	logger.Debug("store.proxy", "RegisterProxy: %s", p.ProxyID)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO proxies(proxy_id, project_slug, status, registered_at, last_ping_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(proxy_id) DO UPDATE SET
		   project_slug = excluded.project_slug,
		   status = excluded.status,
		   last_ping_at = excluded.last_ping_at`,
		p.ProxyID, p.ProjectSlug, p.Status, p.RegisteredAt, p.LastPingAt)
	return err
}

func (s *proxyStore) UnregisterProxy(ctx context.Context, proxyID string) error {
	logger.Debug("store.proxy", "UnregisterProxy: %s", proxyID)
	_, err := s.db.ExecContext(ctx,
		`UPDATE proxies SET status = 'offline' WHERE proxy_id = ?`, proxyID)
	return err
}

func (s *proxyStore) ListProxies(ctx context.Context, filter ProxyFilter) ([]Proxy, error) {
	logger.Debug("store.proxy", "ListProxies: status=%v project=%v", filter.Status, filter.Project)

	var conditions []string
	var args []interface{}

	if filter.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, *filter.Status)
	}
	if filter.Project != nil {
		conditions = append(conditions, "project_slug = ?")
		args = append(args, *filter.Project)
	}

	query := `SELECT proxy_id, project_slug, status, registered_at, last_ping_at FROM proxies`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY registered_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []Proxy
	for rows.Next() {
		var p Proxy
		var lastPing sql.NullTime
		if err := rows.Scan(&p.ProxyID, &p.ProjectSlug, &p.Status, &p.RegisteredAt, &lastPing); err != nil {
			return nil, err
		}
		if lastPing.Valid {
			p.LastPingAt = lastPing.Time
		}
		proxies = append(proxies, p)
	}
	return proxies, rows.Err()
}

func (s *proxyStore) GetProxy(ctx context.Context, proxyID string) (*Proxy, error) {
	logger.Debug("store.proxy", "GetProxy: %s", proxyID)

	var p Proxy
	var lastPing sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT proxy_id, project_slug, status, registered_at, last_ping_at FROM proxies WHERE proxy_id = ?`,
		proxyID).Scan(&p.ProxyID, &p.ProjectSlug, &p.Status, &p.RegisteredAt, &lastPing)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastPing.Valid {
		p.LastPingAt = lastPing.Time
	}
	return &p, nil
}
```

### Step 4: 实现代理服务层

```go
// internal/backend/service/proxy_service.go

package service

import (
	"context"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type ProxyService struct {
	store store.ProxyStore
}

func NewProxyService(s store.ProxyStore) *ProxyService {
	return &ProxyService{store: s}
}

func (svc *ProxyService) List(ctx context.Context, status, project string) ([]store.Proxy, error) {
	logger.Debug("svc.proxy", "List: status=%s project=%s", status, project)

	filter := store.ProxyFilter{}
	if status != "" {
		filter.Status = &status
	}
	if project != "" {
		filter.Project = &project
	}

	return svc.store.ListProxies(ctx, filter)
}
```

### Step 5: 实现 Handler 层

```go
// internal/backend/api/proxy_handler.go

// List 处理获取代理列表的请求。
func (h *ProxyHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	status := r.URL.Query().Get("status")
	project := r.URL.Query().Get("project")

	logger.Debug("api.proxy", "GET /api/proxies status=%s project=%s", status, project)

	proxies, err := h.svc.List(r.Context(), status, project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list proxies")
		return
	}

	items := make([]ProxyItem, len(proxies))
	for i, p := range proxies {
		items[i] = ProxyItem{
			ProxyID:      p.ProxyID,
			ProjectSlug:  p.ProjectSlug,
			Status:       p.Status,
			RegisteredAt: p.RegisteredAt,
			LastPingAt:   p.LastPingAt,
		}
	}

	writeJSON(w, http.StatusOK, ProxiesResponse{
		Proxies: items,
		Total:   len(items),
	})
}
```

### Step 6: 注册路由

```go
// internal/backend/api/router.go

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	// ... 已有路由
	mux.HandleFunc("/api/proxies", h.Proxy.List) // 新增
	return mux
}
```

---

## 四、Todo List

- [ ] Step 1: 定义 ProxiesResponse 和 ProxyItem 类型
- [ ] Step 2: 定义 ProxyStore 接口和 Proxy 模型
- [ ] Step 3: 实现 proxyStore（含 CRUD）
- [ ] Step 4: 实现 ProxyService
- [ ] Step 5: 在 ProxyHandler 中实现 List 方法
- [ ] Step 6: 在 router.go 中注册 /api/proxies 路由
- [ ] Step 7: 添加 proxies 表迁移脚本
- [ ] Step 8: 编写单元测试 proxy_list_api_test.go
- [ ] Step 9: 运行 BDD 测试验证
- [ ] Step 10: 更新 API 文档

---

*文档结束*
