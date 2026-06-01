# BDD + TDD: GET /api/machines 机器列表

> 接口: `GET /api/machines`
> 日期: 2026-06-01
> 状态: ❌ 未实现

---

## 一、接口定义

| 属性 | 值 |
|------|-----|
| 端点 | `/api/machines` |
| 方法 | GET |
| 功能 | 列出所有已注册的机器信息 |
| 查询参数 | `os` (可选), `hostname` (可选) |
| 响应 | `MachinesListResponse` |

---

## 二、BDD 测试用例

```gherkin
Feature: 列出所有机器
  作为监控用户
  我需要查看所有使用 claude-tap-plus 的机器
  以便了解系统部署情况

  Background:
    Given 后端服务已启动
    And 数据库已初始化
    And machines 表存在以下记录:
      | machine_id    | os      | hostname | username | first_seen_at        | last_seen_at         |
      | user@host-1   | windows | host-1   | user     | 2026-05-20T10:00:00Z | 2026-06-01T14:32:00Z |
      | dev@host-2    | linux   | host-2   | dev      | 2026-05-25T09:00:00Z | 2026-06-01T14:28:00Z |
      | admin@host-3  | macos   | host-3   | admin    | 2026-05-28T08:00:00Z | 2026-05-30T12:00:00Z |

  @positive
  Scenario: 获取所有机器列表
    When 发送 GET 请求到 /api/machines
    Then 响应状态码应为 200
    And 响应体应包含 3 个机器
    And 每个机器应包含字段:
      | id | machine_id | os | hostname | username | first_seen_at | last_seen_at |

  @positive
  Scenario: 按操作系统过滤
    When 发送 GET 请求到 /api/machines?os=windows
    Then 响应状态码应为 200
    And 响应体应包含 1 个机器
    And 该机器的 machine_id 应为 "user@host-1"

  @positive
  Scenario: 按主机名过滤
    When 发送 GET 请求到 /api/machines?hostname=host-2
    Then 响应状态码应为 200
    And 响应体应包含 1 个机器
    And 该机器的 machine_id 应为 "dev@host-2"

  @positive
  Scenario: 无机器时返回空数组
    Given machines 表为空
    When 发送 GET 请求到 /api/machines
    Then 响应状态码应为 200
    And 响应体应包含 {"machines": []}

  @negative
  Scenario: 使用非 GET 方法请求
    When 发送 POST 请求到 /api/machines
    Then 响应状态码应为 405
    And 响应体应包含 "method_not_allowed"
```

---

## 三、TDD 开发步骤

### Step 1: 定义响应类型

```go
// internal/backend/api/response.go

// MachinesListResponse 是机器列表的响应体。
type MachinesListResponse struct {
	Machines []MachineListItem `json:"machines"`
}

// MachineListItem 是机器列表中的单个条目。
type MachineListItem struct {
	ID          int64     `json:"id"`
	MachineID   string    `json:"machine_id"`
	OS          string    `json:"os"`
	Hostname    string    `json:"hostname"`
	Username    string    `json:"username"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}
```

### Step 2: 定义 MachineFilter

```go
// internal/backend/store/store.go

// MachineFilter 是机器列表的过滤条件。
type MachineFilter struct {
	OS       *string
	Hostname *string
}
```

### Step 3: 定义 MachineStore 接口

```go
// internal/backend/store/store.go

// MachineStore 定义 Machine 数据存储的接口。
type MachineStore interface {
	ListMachines(ctx context.Context, filter MachineFilter) ([]Machine, error)
	GetMachine(ctx context.Context, machineID string) (*Machine, error)
}
```

### Step 4: 实现 SQLite 存储层

```go
// internal/backend/store/machine_store.go

type sqliteMachineStore struct {
	db *sql.DB
}

func (s *sqliteMachineStore) ListMachines(ctx context.Context, filter MachineFilter) ([]Machine, error) {
	query := `SELECT id, machine_id, os, hostname, username, first_seen_at, last_seen_at FROM machines WHERE 1=1`
	args := []any{}

	if filter.OS != nil {
		query += " AND os = ?"
		args = append(args, *filter.OS)
	}
	if filter.Hostname != nil {
		query += " AND hostname = ?"
		args = append(args, *filter.Hostname)
	}
	query += " ORDER BY last_seen_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list machines: %w", err)
	}
	defer rows.Close()

	var machines []Machine
	for rows.Next() {
		var m Machine
		if err := rows.Scan(&m.ID, &m.MachineID, &m.OS, &m.Hostname, &m.Username, &m.FirstSeenAt, &m.LastSeenAt); err != nil {
			return nil, fmt.Errorf("scan machine: %w", err)
		}
		machines = append(machines, m)
	}
	return machines, rows.Err()
}

func (s *sqliteMachineStore) GetMachine(ctx context.Context, machineID string) (*Machine, error) {
	var m Machine
	err := s.db.QueryRowContext(ctx,
		`SELECT id, machine_id, os, hostname, username, first_seen_at, last_seen_at FROM machines WHERE machine_id = ?`,
		machineID,
	).Scan(&m.ID, &m.MachineID, &m.OS, &m.Hostname, &m.Username, &m.FirstSeenAt, &m.LastSeenAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get machine: %w", err)
	}
	return &m, nil
}
```

### Step 5: 扩展 Store 聚合接口

```go
// internal/backend/store/store.go

type Store interface {
	Issues() IssueStore
	Sessions() SessionStore
	Machines() MachineStore // 新增
	Close() error
}
```

### Step 6: 实现 Service 层

```go
// internal/backend/service/machine_service.go

package service

import (
	"context"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type MachineService struct {
	store store.MachineStore
}

func NewMachineService(s store.MachineStore) *MachineService {
	return &MachineService{store: s}
}

func (svc *MachineService) List(ctx context.Context, filter store.MachineFilter) ([]store.Machine, error) {
	logger.Debug("svc.machine", "List: filter applied")
	return svc.store.ListMachines(ctx, filter)
}

func (svc *MachineService) Get(ctx context.Context, machineID string) (*store.Machine, error) {
	logger.Debug("svc.machine", "Get: id=%s", machineID)
	return svc.store.GetMachine(ctx, machineID)
}
```

### Step 7: 实现 Handler 层

```go
// internal/backend/api/machine_handler.go

package api

import (
	"net/http"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type MachineHandler struct {
	svc *service.MachineService
}

func NewMachineHandler(svc *service.MachineService) *MachineHandler {
	return &MachineHandler{svc: svc}
}

func (h *MachineHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	var filter store.MachineFilter
	if v := r.URL.Query().Get("os"); v != "" {
		filter.OS = &v
	}
	if v := r.URL.Query().Get("hostname"); v != "" {
		filter.Hostname = &v
	}

	logger.Debug("api.machine", "GET /api/machines")

	machines, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list machines")
		return
	}

	if machines == nil {
		machines = []store.Machine{}
	}

	items := make([]MachineListItem, len(machines))
	for i, m := range machines {
		items[i] = MachineListItem{
			ID:          m.ID,
			MachineID:   m.MachineID,
			OS:          m.OS,
			Hostname:    m.Hostname,
			Username:    m.Username,
			FirstSeenAt: m.FirstSeenAt,
			LastSeenAt:  m.LastSeenAt,
		}
	}

	writeJSON(w, http.StatusOK, MachinesListResponse{Machines: items})
}
```

### Step 8: 注册路由

```go
// internal/backend/api/router.go

type Handlers struct {
	Issue   *IssueHandler
	Session *SessionHandler
	Proxy   *ProxyHandler
	Machine *MachineHandler // 新增
}

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", Health)
	
	// ... 已有路由
	
	mux.HandleFunc("/api/machines", h.Machine.List) // 新增
	
	return mux
}
```

---

## 四、Todo List

- [ ] Step 1: 定义 MachinesListResponse 和 MachineListItem 响应类型
- [ ] Step 2: 定义 MachineFilter 过滤条件
- [ ] Step 3: 定义 MachineStore 接口
- [ ] Step 4: 实现 sqliteMachineStore（ListMachines + GetMachine）
- [ ] Step 5: 扩展 Store 聚合接口添加 Machines()
- [ ] Step 6: 实现 MachineService
- [ ] Step 7: 实现 MachineHandler
- [ ] Step 8: 在 router.go 中注册 /api/machines 路由
- [ ] Step 9: 编写单元测试 machine_api_test.go
- [ ] Step 10: 运行 BDD 测试验证
- [ ] Step 11: 更新 API 文档

---

*文档结束*
