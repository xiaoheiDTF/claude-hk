# BDD + TDD: GET /api/status 系统状态

> 接口: `GET /api/status`
> 日期: 2026-06-01
> 状态: ❌ 未实现

---

## 一、接口定义

| 属性 | 值 |
|------|-----|
| 端点 | `/api/status` |
| 方法 | GET |
| 功能 | 获取系统整体运行状态 |
| 响应 | `StatusResponse` |

---

## 二、BDD 测试用例

```gherkin
Feature: 获取系统运行状态
  作为运维人员
  我需要查看系统的整体运行状态
  以便了解系统健康状况

  Background:
    Given 后端服务已启动
    And 数据库已初始化
    And 系统中有:
      | 活跃会话数  | 3  |
      | 活跃代理数  | 2  |
      | 待处理 Issue | 5  |
      | 运行时间    | 3600 秒 |

  @positive
  Scenario: 获取系统状态
    When 发送 GET 请求到 /api/status
    Then 响应状态码应为 200
    And 响应体应包含:
      """
      {
        "status": "healthy",
        "version": "1.0.0",
        "uptime_seconds": 3600,
        "stats": {
          "active_sessions": 3,
          "active_proxies": 2,
          "pending_issues": 5,
          "total_machines": 4,
          "total_projects": 2
        },
        "timestamp": "2026-06-01T12:00:00Z"
      }
      """

  @positive
  Scenario: 系统无活跃会话时状态正常
    Given 系统中无活跃会话
    When 发送 GET 请求到 /api/status
    Then 响应状态码应为 200
    And 响应体中 stats.active_sessions 应为 0
    And 响应体中 status 应为 "healthy"

  @negative
  Scenario: 使用非 GET 方法请求
    When 发送 POST 请求到 /api/status
    Then 响应状态码应为 405
    And 响应体应包含 "method_not_allowed"
```

---

## 三、TDD 开发步骤

### Step 1: 定义响应类型

```go
// internal/backend/api/response.go

// StatusResponse 是系统状态的响应体。
type StatusResponse struct {
	Status        string     `json:"status"`
	Version       string     `json:"version"`
	UptimeSeconds int64      `json:"uptime_seconds"`
	Stats         SystemStats `json:"stats"`
	Timestamp     time.Time  `json:"timestamp"`
}

// SystemStats 是系统统计信息。
type SystemStats struct {
	ActiveSessions int64 `json:"active_sessions"`
	ActiveProxies  int64 `json:"active_proxies"`
	PendingIssues  int64 `json:"pending_issues"`
	TotalMachines  int64 `json:"total_machines"`
	TotalProjects  int64 `json:"total_projects"`
}
```

### Step 2: 实现状态统计服务

```go
// internal/backend/service/status_service.go

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

// 服务启动时间（全局变量，在 main 中设置）
var serverStartTime time.Time

func init() {
	serverStartTime = time.Now()
}

type StatusService struct {
	sessionStore store.SessionStore
	proxyStore   store.ProxyStore
	issueStore   store.IssueStore
	machineStore store.MachineStore
	projectStore store.ProjectStore
}

func NewStatusService(
	sessionStore store.SessionStore,
	proxyStore store.ProxyStore,
	issueStore store.IssueStore,
	machineStore store.MachineStore,
	projectStore store.ProjectStore,
) *StatusService {
	return &StatusService{
		sessionStore: sessionStore,
		proxyStore:   proxyStore,
		issueStore:   issueStore,
		machineStore: machineStore,
		projectStore: projectStore,
	}
}

// SystemStatus 表示系统整体状态。
type SystemStatus struct {
	Status        string
	Version       string
	UptimeSeconds int64
	Stats         SystemStats
	Timestamp     time.Time
}

func (svc *StatusService) Get(ctx context.Context) (*SystemStatus, error) {
	logger.Debug("svc.status", "Get")

	// 获取活跃会话数
	activeSessions, err := svc.countActiveSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("count active sessions: %w", err)
	}

	// 获取活跃代理数
	activeProxies, err := svc.countActiveProxies(ctx)
	if err != nil {
		return nil, fmt.Errorf("count active proxies: %w", err)
	}

	// 获取待处理 Issue 数
	pendingIssues, err := svc.countPendingIssues(ctx)
	if err != nil {
		return nil, fmt.Errorf("count pending issues: %w", err)
	}

	// 获取机器总数
	totalMachines, err := svc.countMachines(ctx)
	if err != nil {
		return nil, fmt.Errorf("count machines: %w", err)
	}

	// 获取项目总数
	totalProjects, err := svc.countProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("count projects: %w", err)
	}

	// 计算运行时间
	uptime := time.Since(serverStartTime).Seconds()

	return &SystemStatus{
		Status:        "healthy",
		Version:       "1.0.0",
		UptimeSeconds: int64(uptime),
		Stats: SystemStats{
			ActiveSessions: activeSessions,
			ActiveProxies:  activeProxies,
			PendingIssues:  pendingIssues,
			TotalMachines:  totalMachines,
			TotalProjects:  totalProjects,
		},
		Timestamp: time.Now().UTC(),
	}, nil
}

func (svc *StatusService) countActiveSessions(ctx context.Context) (int64, error) {
	sessions, err := svc.sessionStore.ListSessions(ctx, store.SessionFilter{Status: strPtr("active")})
	if err != nil {
		return 0, err
	}
	return int64(len(sessions)), nil
}

func (svc *StatusService) countActiveProxies(ctx context.Context) (int64, error) {
	proxies, err := svc.proxyStore.ListProxies(ctx, store.ProxyFilter{Status: strPtr("active")})
	if err != nil {
		return 0, err
	}
	return int64(len(proxies)), nil
}

func (svc *StatusService) countPendingIssues(ctx context.Context) (int64, error) {
	// 待处理 = idle 状态的 issue
	issues, err := svc.issueStore.ListIssues(ctx, store.IssueFilter{Status: strPtr("idle")})
	if err != nil {
		return 0, err
	}
	return int64(len(issues)), nil
}

func (svc *StatusService) countMachines(ctx context.Context) (int64, error) {
	machines, err := svc.machineStore.ListMachines(ctx, store.MachineFilter{})
	if err != nil {
		return 0, err
	}
	return int64(len(machines)), nil
}

func (svc *StatusService) countProjects(ctx context.Context) (int64, error) {
	projects, err := svc.projectStore.ListProjects(ctx, store.ProjectFilter{})
	if err != nil {
		return 0, err
	}
	return int64(len(projects)), nil
}

func strPtr(s string) *string {
	return &s
}
```

### Step 3: 实现 Handler 层

```go
// internal/backend/api/status_handler.go

package api

import (
	"net/http"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type StatusHandler struct {
	svc *service.StatusService
}

func NewStatusHandler(svc *service.StatusService) *StatusHandler {
	return &StatusHandler{svc: svc}
}

func (h *StatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	logger.Debug("api.status", "GET /api/status")

	status, err := h.svc.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get status")
		return
	}

	writeJSON(w, http.StatusOK, StatusResponse{
		Status:        status.Status,
		Version:       status.Version,
		UptimeSeconds: status.UptimeSeconds,
		Stats: SystemStats{
			ActiveSessions: status.Stats.ActiveSessions,
			ActiveProxies:  status.Stats.ActiveProxies,
			PendingIssues:  status.Stats.PendingIssues,
			TotalMachines:  status.Stats.TotalMachines,
			TotalProjects:  status.Stats.TotalProjects,
		},
		Timestamp: status.Timestamp,
	})
}
```

### Step 4: 注册路由

```go
// internal/backend/api/router.go

type Handlers struct {
	Issue   *IssueHandler
	Session *SessionHandler
	Proxy   *ProxyHandler
	Machine *MachineHandler
	Project *ProjectHandler
	Log     *LogHandler
	Config  *ConfigHandler
	Status  *StatusHandler // 新增
}

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	// ... 已有路由
	mux.HandleFunc("/api/status", h.Status.Get) // 新增
	return mux
}
```

---

## 四、Todo List

- [ ] Step 1: 定义 StatusResponse 和 SystemStats 类型
- [ ] Step 2: 实现 StatusService（含各统计计数）
- [ ] Step 3: 实现 StatusHandler
- [ ] Step 4: 在 router.go 中注册 /api/status 路由
- [ ] Step 5: 确保各 Store 接口有 List 方法
- [ ] Step 6: 编写单元测试 status_api_test.go
- [ ] Step 7: 运行 BDD 测试验证
- [ ] Step 8: 更新 API 文档

---

*文档结束*
