# claude_tap_plus 重构方案：TDD / BDD / DDD / DSL / ATDD

> **目标范围**：`claude_tap_plus/` Go 侧（proxy + backend）
> **输出形式**：文档方案，每个理论独立成章，可分阶段实施
> **生成日期**：2026-06-08

---

## 现状诊断

| 维度 | 现状 | 痛点 |
|------|------|------|
| **架构** | domain → store → service → api 四层 | 层次存在但职责偏移：service 层几乎全是 store 的透传 wrapper |
| **领域模型** | `domain/` 有 struct 和 enum，无行为 | 贫血模型：`IssueClaim`/`Session` 是纯数据，状态流转逻辑散落在 SQL 和 handler 里 |
| **测试** | 26 个文件，全部是 API 集成测试 | domain/service/store 三层零单元测试；e2e 测试因缺少 fixture 失败 |
| **耦合** | `StatusService` 依赖 5 个 store 接口 | God Service 倾向；service 不定义接口，无法 mock |
| **错误处理** | `store.ErrSessionExists` 等零散 sentinel error | 无领域级错误类型，store 错误直接泄漏到 API 层 |
| **Proxy** | `reverse.go` 878 行单文件 | 混合了路由、转发、SSE、trace、fallback 五种职责 |

以下五章按 **独立可实施** 原则组织，每章包含：理论要点、当前差距、重构目标、具体步骤、验证标准。

---

## 第一章 TDD — 测试驱动开发

### 1.1 理论要点

```
Red → Green → Refactor 循环：
1. 先写一个失败的测试（Red）
2. 写最少代码让测试通过（Green）
3. 在测试保护下重构（Refactor）
```

TDD 的核心价值不是"写测试"，而是 **通过测试驱动设计**：如果你觉得某个逻辑难测，说明设计有问题。

### 1.2 当前差距

| 位置 | 现状 | 风险 |
|------|------|------|
| `domain/` | 4 个文件，全是数据 struct | 零行为逻辑，无测试价值 — 但重构后会有 |
| `service/` | 11 个文件，全是 store 透传 | 改动业务逻辑无任何回归保护 |
| `store/` | 接口定义 + SQLite 实现 | SQL 错误路径、并发竞争完全未验证 |
| `proxy/` | `reverse.go` 878 行 | SSE 拆分、fallback 切换等核心路径无单测 |
| `tests/e2e/` | 6 个测试全部 FAIL | 缺少 fixture 文件，无法运行 |

### 1.3 重构目标

1. **domain 层**：每个实体行为有独立单元测试
2. **service 层**：用 mock store 验证业务编排逻辑
3. **store 层**：SQLite 实现有针对 SQL 边界条件的测试
4. **proxy 层**：核心路径（转发、SSE、fallback）有 table-driven 测试
5. **修复 e2e**：补齐 fixture，让 6 个失败测试通过

### 1.4 具体步骤

#### 阶段 1：补齐基础设施（1-2 天）

```
internal/testutil/
├── fixture.go       ← 已有，需扩展支持内联 fixture
├── mock_store.go    ← 新增：自动生成的 store mock
└── testdb.go        ← 新增：内存 SQLite 测试 DB 工具
```

**步骤 1.1** — 创建 `internal/testutil/testdb.go`

```go
// OpenTestDB 创建一个内存 SQLite 数据库，执行完整 migration，返回 *store.SQLiteStore。
// 自动清理（t.Cleanup）。
func OpenTestDB(t *testing.T) store.Store {
    t.Helper()
    // 用 ":memory:" 打开 SQLite，执行 migrations，返回 store.Store 接口
}
```

**步骤 1.2** — 为每个 Store 接口生成 mock

```go
// internal/testutil/mock_issue_store.go
type MockIssueStore struct {
    CheckIssuesFunc         func(ctx context.Context, repo string, numbers []int) ([]store.IssueCheckResult, error)
    ClaimIssueFunc          func(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*store.ClaimResult, error)
    // ...
}
```

> 不使用 mockgen 等工具，手写 mock 以保持零外部依赖（项目当前只用 Go 标准库 + modernc.org/sqlite）。

**步骤 1.3** — 修复 e2e fixture

```bash
# 补齐 tests/e2e/testdata/fixtures/ 下的 JSON 文件
# 从现有 proxy 测试用例中提取标准 fixture
```

**验证**：`go test ./...` 全部 PASS（当前有 2 个包 FAIL）

#### 阶段 2：Store 层单元测试（2-3 天）

优先测试 SQL 边界条件，每个 Store 接口方法至少覆盖：

| 方法 | 正常路径 | 边界条件 | 错误路径 |
|------|---------|---------|---------|
| `ClaimIssue` | 首次领取成功 | 重复领取、已被他人领取 | 无 |
| `UpdateIssueStatus` | 正常流转 | 非持有者更新、无效状态值 | 无 |
| `ReleaseSessionIssues` | 释放多个 | 无 claim 时的空列表 | 无 |
| `RegisterSession` | 首次注册 | 重复 session_id → `ErrSessionExists` | 无 |
| `CloseSession` | 正常关闭 | 已关闭 → `ErrSessionNotFound` | 无 |

目录结构：

```
internal/backend/store/
├── issue_store_test.go     ← 新增
├── session_store_test.go   ← 新增
├── machine_store_test.go   ← 新增
├── config_store_test.go    ← 新增
└── sqlite_test.go          ← 新增（DB 初始化、migration 验证）
```

测试风格：**table-driven tests**

```go
func TestClaimIssue(t *testing.T) {
    db := testutil.OpenTestDB(t)
    issues := db.Issues()

    tests := []struct {
        name      string
        setup     func()  // 预置数据
        repo      string
        number    int
        sessionID string
        want      *store.ClaimResult
        wantErr   bool
    }{
        {name: "首次领取成功", ...},
        {name: "已被其他 session 领取", ...},
        {name: "同一 session 重复领取", ...},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) { ... })
    }
}
```

**验证**：`go test ./internal/backend/store/... -cover` 覆盖率 > 80%

#### 阶段 3：Service 层单元测试（2-3 天）

用 mock store 隔离 service 逻辑。测试重点不再是 SQL，而是 **业务编排**：

```
internal/backend/service/
├── issue_service_test.go     ← 新增
├── session_service_test.go   ← 新增
├── status_service_test.go    ← 新增
├── cleanup_service_test.go   ← 新增
└── idle_watchdog_test.go     ← 新增
```

关键测试场景：

- `IssueService.UpdateStatus` — 验证日志记录、错误传播
- `StatusService.Get` — mock 5 个 store，验证聚合逻辑
- `CleanupService` — 验证超时判定和批量清理
- `IdleWatchdog` — 验证定时触发和 context 取消

**验证**：`go test ./internal/backend/service/... -cover` 覆盖率 > 70%

#### 阶段 4：Proxy 核心路径测试（2-3 天）

将 `reverse.go` 的核心方法提取为可测试的纯函数：

```go
// internal/proxy/transform_test.go
func TestRewriteModel(t *testing.T) { ... }
func TestInjectThinking(t *testing.T) { ... }
func TestFallbackDecision(t *testing.T) { ... }
```

已有测试文件（可扩展）：
- `internal/proxy/fallback_test.go` ✓
- `internal/proxy/model_rewrite_test.go` ✓
- `internal/proxy/thinking_inject_test.go` ✓
- `internal/proxy/reasoning_cache_test.go` ✓

**验证**：`go test ./internal/proxy/... -cover` 覆盖率 > 60%

### 1.5 实施原则

1. **每个重构动作前先写测试** — 测试是安全网，不是装饰
2. **先覆盖高风险路径**：并发 claim、状态流转、超时清理
3. **测试即文档**：测试名用中文描述场景，作为行为规格
4. **保持零外部依赖**：不用 testify/mockgen，手写 mock 保持项目简洁

---

## 第二章 BDD — 行为驱动开发

### 2.1 理论要点

```
BDD 用 Given-When-Then 描述系统行为：
Given（前置条件）→ When（操作）→ Then（预期结果）

BDD 关注的是"系统应该做什么"，而不是"代码怎么写"。
场景（Scenario）是团队共享的验收标准。
```

### 2.2 当前差距

当前测试（`tests/backend/`）是 API 级别的 HTTP 集成测试，关注"请求-响应"匹配，但 **不描述行为场景**：

```go
// 现有测试 — 关注 HTTP 契约，不关注业务语义
func TestClaimAPI(t *testing.T) {
    resp := httpPost("/api/issue/claim", body)
    assert(resp.StatusCode == 200)
}
```

缺失：
- 无 Given-When-Then 场景描述
- 测试名是技术性的（`TestClaimAPI`）而非业务性的
- 无场景覆盖：正常领取 → 已被领取 → 释放 → 重新领取 的完整生命周期

### 2.3 重构目标

1. 用 **场景函数** 包装测试，测试名即行为描述
2. 覆盖 3 条核心业务流的完整 BDD 场景
3. 场景可同时用于单元测试和集成测试

### 2.4 核心业务流与场景

#### 流程 1：Issue 生命周期

| # | 场景 | Given | When | Then |
|---|------|-------|------|------|
| 1.1 | 首次领取空闲 Issue | Issue 状态为 idle | session-A 发起 claim | 返回成功，状态变为 claimed |
| 1.2 | 领取已被占用的 Issue | Issue 已被 session-A 领取 | session-B 发起 claim | 返回失败，包含当前持有者信息 |
| 1.3 | 同 session 重复领取 | Issue 已被 session-A 领取 | session-A 再次 claim | 返回成功（幂等） |
| 1.4 | 状态正向流转 | Issue 为 claimed 状态 | 持有者更新为 fixing | 返回前一状态 claimed |
| 1.5 | 非持有者无法流转 | Issue 被 session-A 持有 | session-B 尝试更新 | 返回失败，状态不变 |
| 1.6 | 终态不可释放 | Issue 为 merged 状态 | 持有者尝试 release | 返回 false，状态保持 |
| 1.7 | Session 结束批量释放 | session-A 持有 3 个 Issue | session 关闭 | 3 个 Issue 回到 idle |
| 1.8 | Issue 列表过滤 | 存在多个 Issue | 按状态过滤 | 只返回匹配的记录 |

#### 流程 2：Session 生命周期

| # | 场景 | Given | When | Then |
|---|------|-------|------|------|
| 2.1 | 注册新会话 | 无同名 session | 注册 session-A | 成功，状态为 active |
| 2.2 | 重复注册 | session-A 已存在 | 再次注册同一 session | 返回 `ErrSessionExists` |
| 2.3 | 关闭活跃会话 | session-A 为 active | 关闭并指定原因 | 状态变为 closed，记录关闭时间 |
| 2.4 | 关闭已关闭会话 | session-A 已 closed | 再次关闭 | 返回 `ErrSessionNotFound` |
| 2.5 | 超时清理 | 有超过阈值的 active session | 执行 cleanup | 标记为 closed，close_reason 为 timeout |

#### 流程 3：Proxy 拦截

| # | 场景 | Given | When | Then |
|---|------|-------|------|------|
| 3.1 | 正常转发非流式请求 | 上游可用 | 发送 messages API 请求 | 转发成功，记录 trace |
| 3.2 | SSE 流式响应 | 上游返回 SSE | 代理转发 | 逐事件转发，记录完整 trace |
| 3.3 | 上游不可用切换兜底 | 上游不可用 + 有兜底配置 | 发送请求 | 切换到兜底上游 |
| 3.4 | 上游恢复切回 | 兜底模式中上游恢复 | 发送请求 | 切回原上游 |
| 3.5 | 非白名单路径 | 请求路径不在白名单 | 发送请求 | 拒绝转发 |

### 2.5 实施方案

#### BDD 场景测试模式

不引入外部 BDD 框架（如 godog），用 **Go 子测试 + 中文场景名** 实现轻量 BDD：

```go
// tests/backend/issue_lifecycle_test.go
func TestIssueLifecycle(t *testing.T) {
    db := testutil.OpenTestDB(t)

    t.Run("场景1.1: 首次领取空闲Issue", func(t *testing.T) {
        // Given: Issue 状态为 idle（数据库初始状态）
        // When
        result, err := db.Issues().ClaimIssue(ctx, "owner/repo", 42, "session-A", "Bug fix")
        // Then
        require.NoError(t, err)
        assert.True(t, result.Success)
        assert.Equal(t, "claimed", result.Status)
    })

    t.Run("场景1.2: 领取已被占用的Issue", func(t *testing.T) {
        // Given: Issue 已被 session-A 领取
        db.Issues().ClaimIssue(ctx, "owner/repo", 42, "session-A", "title")
        // When: session-B 尝试领取
        result, err := db.Issues().ClaimIssue(ctx, "owner/repo", 42, "session-B", "title")
        // Then: 返回冲突信息
        require.NoError(t, err)
        assert.False(t, result.Success)
        assert.Equal(t, "session-A", *result.ClaimedBy)
    })

    // ... 继续覆盖 1.3 ~ 1.8
}
```

#### 文件组织

```
tests/
├── backend/
│   ├── issue_lifecycle_test.go      ← 新增：Issue 全生命周期场景
│   ├── session_lifecycle_test.go    ← 新增：Session 全生命周期场景
│   ├── concurrency_test.go          ← 已有，扩展更多并发场景
│   └── ...existing API tests
├── proxy/
│   ├── intercept_lifecycle_test.go  ← 新增：Proxy 拦截场景
│   └── ...existing tests
└── integration/
    └── full_flow_test.go            ← 新增：端到端 Issue → Fix → PR → Review 全流程
```

### 2.6 验证标准

- 每个场景的测试名完整描述 Given-When-Then
- `go test -run "场景" -v` 输出可作为行为文档阅读
- 三条核心流程覆盖率达到 100% 场景通过
- 并发场景（多 session 竞争 claim）有独立测试覆盖

---

## 第三章 DDD — 领域驱动设计

### 3.1 理论要点

```
DDD 核心概念：
- 限界上下文（Bounded Context）：划分领域边界
- 聚合根（Aggregate Root）：事务一致性边界
- 值对象（Value Object）：不可变的领域概念
- 领域事件（Domain Event）：跨聚合通信
- 通用语言（Ubiquitous Language）：代码与业务统一术语
```

DDD 的核心价值不是"把 struct 放到 domain/ 目录"，而是 **让代码表达业务语义**。

### 3.2 当前差距

#### 差距 1：贫血模型

```go
// 当前：domain.IssueClaim 是纯数据容器
type IssueClaim struct {
    Status    IssueStatus `json:"status"`
    SessionID string      `json:"session_id"`
    // ... 无方法
}

// 状态流转逻辑散落在 store SQL 里：
// UPDATE issue_claims SET status = ? WHERE ...
// 谁能流转到什么状态？答案是藏在 SQL 条件里的隐式规则
```

#### 差距 2：store 和 domain 重复定义

```
domain/issue.go    → IssueClaim struct
store/issue_store.go → IssueCheckResult, ClaimResult, UpdateStatusResult, IssueListItem
store/store.go      → Session struct（与 domain.Session 重复）
```

同一个 "Issue" 概念被拆成了多个 DTO，散落在不同包。

#### 差距 3：StatusService 跨聚合查询

```go
// status_service.go — 依赖 5 个 store，跨聚合聚合统计
type StatusService struct {
    sessionStore store.SessionStore
    proxyStore   store.ProxyStore
    issueStore   store.IssueStore
    machineStore store.MachineStore
    projectStore store.ProjectStore
}
```

#### 差距 4：无领域错误类型

```go
// 当前只有两个 sentinel error
var (
    ErrSessionExists  = errors.New("session already exists")
    ErrSessionNotFound = errors.New("session not found or already closed")
)
// Issue 相关的错误全靠 bool 返回值判断
```

### 3.3 重构目标

1. **富领域模型**：实体拥有行为方法，封装状态流转规则
2. **统一领域类型**：消除 domain/ 和 store/ 的重复定义
3. **领域错误**：用类型化错误替代 sentinel error + bool
4. **限界上下文**：明确 Issue、Session、Proxy 三个聚合根

### 3.4 限界上下文划分

```
┌──────────────────────────────────────────────────┐
│                claude_tap_plus                    │
│                                                   │
│  ┌─────────────┐ ┌──────────────┐ ┌───────────┐ │
│  │ Issue 领域   │ │ Session 领域  │ │ Proxy 领域│ │
│  │             │ │              │ │           │ │
│  │ Aggregate:  │ │ Aggregate:   │ │ Aggregate:│ │
│  │ IssueClaim  │ │ Session      │ │ ProxyInst │ │
│  │             │ │              │ │           │ │
│  │ ValueObj:   │ │ ValueObj:    │ │ ValueObj: │ │
│  │ IssueStatus │ │ SessionStatus│ │ TraceID   │ │
│  │ RepoRef     │ │ MachineID    │ │ TurnNo    │ │
│  │             │ │              │ │           │ │
│  │ Events:     │ │ Events:      │ │           │ │
│  │ IssueClaimed│ │ SessionReg-  │ │           │ │
│  │ IssueReleased│ │   istered   │ │           │ │
│  │ IssueMerged │ │ SessionClos- │ │           │ │
│  │             │ │   ed         │ │           │ │
│  └─────────────┘ └──────────────┘ └───────────┘ │
│                                                   │
│  ┌─────────────────────────────────────────────┐ │
│  │ Shared Kernel（共享内核）                      │ │
│  │ • MachineID, ProjectSlug, TimeRange         │ │
│  │ • 通用错误类型                                │ │
│  └─────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────┘
```

### 3.5 具体步骤

#### 步骤 1：富领域模型 — IssueClaim

**当前** (`domain/issue.go`)：纯数据 struct

**目标**：IssueClaim 封装状态流转规则

```go
// domain/issue.go — 重构后

// CanTransitionTo 检查当前状态是否允许转换到目标状态。
func (s IssueStatus) CanTransitionTo(target IssueStatus) bool {
    transitions := map[IssueStatus][]IssueStatus{
        IssueIdle:       {IssueClaimed},
        IssueClaimed:    {IssueFixing, IssueIdle},           // 可开始修复或释放
        IssueFixing:     {IssueReadyForPR, IssueIdle},        // 可完成或放弃
        IssueReadyForPR: {IssuePRCreated, IssueIdle},
        IssuePRCreated:  {IssueTesting, IssueIdle},
        IssueTesting:    {IssueReviewing, IssueIdle},
        IssueReviewing:  {IssueMerged, IssueRejected, IssueIdle},
        IssueRejected:   {IssueFixing, IssueIdle},            // 打回可重新修复
        IssueMerged:     {},                                   // 终态
    }
    allowed, ok := transitions[s]
    if !ok {
        return false
    }
    for _, t := range allowed {
        if t == target {
            return true
        }
    }
    return false
}

// IsTerminal 判断是否为终态。
func (s IssueStatus) IsTerminal() bool {
    return s == IssueMerged
}

// IsOwnedBy 检查指定 session 是否为当前持有者。
func (c *IssueClaim) IsOwnedBy(sessionID string) bool {
    return c.SessionID == sessionID && c.Status != IssueIdle
}

// Claim 尝试由指定 session 领取此 Issue。
// 成功时返回 nil；失败时返回领域错误。
func (c *IssueClaim) Claim(sessionID string) error {
    if c.Status == IssueClaimed && c.SessionID == sessionID {
        return nil // 幂等：同一 session 重复领取
    }
    if c.Status != IssueIdle {
        return ErrIssueAlreadyClaimed{Owner: c.SessionID, Status: string(c.Status)}
    }
    c.Status = IssueClaimed
    c.SessionID = sessionID
    now := time.Now()
    c.ClaimedAt = &now
    return nil
}

// Release 尝试释放此 Issue。
func (c *IssueClaim) Release(sessionID string) error {
    if !c.IsOwnedBy(sessionID) {
        return ErrNotIssueOwner
    }
    if c.Status.IsTerminal() {
        return ErrIssueTerminal
    }
    c.Status = IssueIdle
    c.SessionID = ""
    c.ClaimedAt = nil
    return nil
}

// TransitionStatus 执行状态流转。
func (c *IssueClaim) TransitionStatus(sessionID string, target IssueStatus) error {
    if !c.IsOwnedBy(sessionID) {
        return ErrNotIssueOwner
    }
    if !c.Status.CanTransitionTo(target) {
        return ErrInvalidTransition{From: string(c.Status), To: string(target)}
    }
    c.Status = target
    return nil
}
```

#### 步骤 2：领域错误类型

**新建** `domain/errors.go`：

```go
package domain

import "fmt"

// ErrIssueAlreadyClaimed 表示 Issue 已被其他 session 领取。
type ErrIssueAlreadyClaimed struct {
    Owner  string
    Status string
}

func (e ErrIssueAlreadyClaimed) Error() string {
    return fmt.Sprintf("issue already claimed by %s (status: %s)", e.Owner, e.Status)
}

// ErrNotIssueOwner 表示操作者不是 Issue 的持有者。
type ErrNotIssueOwner struct{}

func (e ErrNotIssueOwner) Error() string {
    return "not the issue owner"
}

// ErrIssueTerminal 表示 Issue 已处于终态，不可变更。
type ErrIssueTerminal struct{}

func (e ErrIssueTerminal) Error() string {
    return "issue is in terminal state"
}

// ErrInvalidTransition 表示状态流转不合法。
type ErrInvalidTransition struct {
    From string
    To   string
}

func (e ErrInvalidTransition) Error() string {
    return fmt.Sprintf("invalid status transition: %s -> %s", e.From, e.To)
}
```

#### 步骤 3：统一类型定义

**消除 `store/store.go` 与 `domain/` 的重复**：

```
重构前：
  domain/session.go → Session struct (带 JSON tag)
  store/store.go    → Session struct (无 JSON tag，字段完全相同)

重构后：
  domain/session.go → Session struct (唯一定义)
  store/store.go    → 引用 domain.Session
```

将 `store/` 中的 DTO 类型（`ClaimResult`, `UpdateStatusResult`, `IssueListItem` 等）迁移到 `domain/`，让 store 层返回领域类型。

#### 步骤 4：重构 StatusService — 引入 ReadModel

**当前**：StatusService 直接依赖 5 个 store 做全表扫描计数。

**目标**：引入 `SystemStatsReadModel` 接口，封装统计查询。

```go
// domain/stats.go
type SystemStats struct {
    ActiveSessions int64
    ActiveProxies  int64
    PendingIssues  int64
    TotalMachines  int64
    TotalProjects  int64
}

// StatsReadModel 封装系统统计的读取模型。
type StatsReadModel interface {
    GetSystemStats(ctx context.Context) (*SystemStats, error)
}
```

```go
// store/stats_store.go — 新增，用 COUNT 查询替代全表扫描
type StatsStore interface {
    CountActiveSessions(ctx context.Context) (int64, error)
    CountActiveProxies(ctx context.Context) (int64, error)
    CountPendingIssues(ctx context.Context) (int64, error)
    CountMachines(ctx context.Context) (int64, error)
    CountProjects(ctx context.Context) (int64, error)
}
```

```go
// service/status_service.go — 重构后
type StatusService struct {
    stats     StatsReadModel  // 从 5 个 store → 1 个接口
    startTime time.Time
}
```

**收益**：StatusService 依赖从 5 个 → 1 个；SQL 从全表扫描 → COUNT 优化。

### 3.6 目录结构（重构后）

```
internal/backend/
├── domain/
│   ├── issue.go         ← 富模型：IssueClaim + 状态机 + 行为方法
│   ├── session.go       ← 富模型：Session + 行为方法
│   ├── machine.go       ← 值对象：MachineID
│   ├── project.go       ← 值对象：ProjectSlug
│   ├── token.go         ← 值对象：TokenStats
│   ├── errors.go        ← 新增：领域错误类型
│   └── stats.go         ← 新增：统计读取模型
├── store/
│   ├── store.go         ← 接口引用 domain 类型
│   ├── sqlite.go
│   ├── issue_store.go   ← 返回 domain.IssueClaim
│   ├── session_store.go ← 返回 domain.Session
│   └── ...
├── service/
│   ├── issue_service.go ← 调用 domain 方法做业务校验，再调 store 持久化
│   ├── status_service.go ← 依赖 StatsReadModel 而非 5 个 store
│   └── ...
└── api/
    └── ...              ← 不变，继续依赖 service 接口
```

### 3.7 验证标准

- `domain.IssueClaim` 封装了全部状态流转规则，store 层不再包含业务逻辑
- 状态转换有明确的 `CanTransitionTo()` 规则表，且 **有测试覆盖每种转换**
- `StatusService` 依赖数从 5 → 1
- `store/` 和 `domain/` 无重复类型定义
- 领域错误类型覆盖全部业务异常场景

---

## 第四章 DSL — 领域特定语言

### 4.1 理论要点

```
DSL 是针对特定问题域设计的表达方式，分两类：
- 外部 DSL：独立语法（如 SQL、正则表达式）
- 内部 DSL：嵌入宿主语言，用 Fluent API / Builder 模式实现

Go 语言偏好内部 DSL：链式调用、Option 模式、函数式选项。
```

在 Go 项目中，DSL 的核心价值是 **让配置和编排代码读起来像自然语言**。

### 4.2 当前差距

#### 差距 1：Proxy 配置是裸参数

```go
// main.go — 大量散落的配置参数
func NewReverseProxy(target, traceDir string) *ReverseProxy
```

缺少结构化的配置 DSL。

#### 差距 2：状态流转是隐式的字符串比较

```go
// issue_store.go — 状态判断散落在 SQL 和 Go 代码里
if status != "idle" { ... }
// 没有统一的流转规则表达
```

#### 差距 3：Backend 路由注册是手动的字符串拼接

```go
// router.go
r.HandleFunc("POST", "/api/issue/check", h.Issue.Check)
r.HandleFunc("POST", "/api/issue/claim", h.Issue.Claim)
// 20+ 路由全部手动注册
```

### 4.3 重构目标

1. **Proxy Builder DSL**：链式构建代理配置
2. **状态机 DSL**：用 Go 代码声明式定义状态转换规则
3. **路由注册 DSL**：用声明式注册替代手动拼接

### 4.4 具体步骤

#### 步骤 1：Proxy Builder DSL

**当前**：

```go
p := proxy.NewReverseProxy(target, traceDir)
p.SetModel(model)          // 需要记得调用
p.SetFallback(config)      // 需要记得调用
p.SetKimiMode(kimiMode)    // 需要记得调用
```

**目标** — Option 模式 + 链式调用：

```go
// internal/proxy/options.go — 新增

// ProxyOption 是代理配置的函数式选项。
type ProxyOption func(*ReverseProxy)

// WithModel 设置模型改写。
func WithModel(model string) ProxyOption {
    return func(p *ReverseProxy) { p.model = model }
}

// WithFallback 设置兜底配置。
func WithFallback(cfg FallbackConfig) ProxyOption {
    return func(p *ReverseProxy) { p.fallbackConfig = &cfg }
}

// WithKimiMode 启用 Kimi 上游模式。
func WithKimiMode() ProxyOption {
    return func(p *ReverseProxy) { p.kimiMode = true }
}

// WithSessionInit 设置会话初始化回调。
func WithSessionInit(fn func(string, string)) ProxyOption {
    return func(p *ReverseProxy) { p.OnSessionInit = fn }
}
```

**使用方式**：

```go
p, err := proxy.NewReverseProxy(
    target, traceDir,
    proxy.WithModel("claude-sonnet-4-6"),
    proxy.WithFallback(fallbackCfg),
    proxy.WithSessionInit(onInit),
)
```

#### 步骤 2：状态机 DSL

在 DDD 章节（第三章）定义了 `CanTransitionTo()`，这里进一步用 DSL 风格声明状态机：

```go
// domain/issue.go — 状态机 DSL

// IssueTransitions 定义 Issue 状态的合法流转路径。
// 格式：当前状态 → 允许的目标状态列表
var IssueTransitions = TransitionTable{
    IssueIdle:       {IssueClaimed},
    IssueClaimed:    {IssueFixing, IssueIdle},
    IssueFixing:     {IssueReadyForPR, IssueIdle},
    IssueReadyForPR: {IssuePRCreated, IssueIdle},
    IssuePRCreated:  {IssueTesting, IssueIdle},
    IssueTesting:    {IssueReviewing, IssueIdle},
    IssueReviewing:  {IssueMerged, IssueRejected, IssueIdle},
    IssueRejected:   {IssueFixing, IssueIdle},
    IssueMerged:     {},
}

// TransitionTable 是状态流转规则的声明式定义。
type TransitionTable map[IssueStatus][]IssueStatus

// Allowed 检查从 source 到 target 的流转是否合法。
func (t TransitionTable) Allowed(source, target IssueStatus) bool {
    targets, ok := t[source]
    if !ok { return false }
    for _, t := range targets {
        if t == target { return true }
    }
    return false
}

// Validate 验证整个流转表的一致性（测试时调用）。
func (t TransitionTable) Validate() error {
    for source, targets := range t {
        for _, target := range targets {
            if _, ok := t[target]; !ok && !target.IsTerminal() {
                return fmt.Errorf("target %s of %s is not defined as a source", target, source)
            }
        }
    }
    return nil
}
```

**使用**：

```go
// CanTransitionTo 委托给声明式的流转表
func (s IssueStatus) CanTransitionTo(target IssueStatus) bool {
    return IssueTransitions.Allowed(s, target)
}
```

**收益**：状态流转规则从散落在各处的 if-else → 一张声明式的 `TransitionTable`，一眼可读、可测试、可验证。

#### 步骤 3：路由注册 DSL

**当前** (`api/router.go`)：

```go
r.HandleFunc("POST", "/api/issue/check", h.Issue.Check)
r.HandleFunc("POST", "/api/issue/claim", h.Issue.Claim)
r.HandleFunc("POST", "/api/issue/release", h.Issue.Release)
// ... 20+ 行手动注册
```

**目标** — 声明式路由表：

```go
// internal/backend/api/routes.go — 新增

// RouteDef 定义单条路由。
type RouteDef struct {
    Method  string
    Path    string
    Handler http.HandlerFunc
}

// Routes 返回所有 API 路由定义。
// 路由表即文档：一眼看清全部 API。
func (h Handlers) Routes() []RouteDef {
    return []RouteDef{
        // Health
        {"GET", "/health", h.Health.Check},

        // Issue
        {"POST", "/api/issue/check", h.Issue.Check},
        {"POST", "/api/issue/claim", h.Issue.Claim},
        {"POST", "/api/issue/release", h.Issue.Release},
        {"POST", "/api/issue/release-session", h.Issue.ReleaseSession},
        {"POST", "/api/issue/status", h.Issue.UpdateStatus},
        {"GET", "/api/issues", h.Issue.List},
        {"GET", "/api/session/{id}/issues", h.Issue.ListBySession},

        // Session
        {"POST", "/api/session/register", h.Session.Register},
        {"POST", "/api/session/close", h.Session.Close},
        {"GET", "/api/sessions", h.Session.List},

        // ... 其余路由
    }
}
```

```go
// router.go — 简化为循环注册
func NewRouter(h Handlers) http.Handler {
    r := httprouter.New()
    for _, route := range h.Routes() {
        r.HandlerFunc(route.Method, route.Path, route.Handler)
    }
    return r
}
```

### 4.5 验证标准

- Proxy 创建使用 Option 模式，所有可选配置通过 `WithXxx()` 函数传入
- 状态流转规则集中在一张 `TransitionTable`，有 `Validate()` 自检测试
- 路由注册从 20+ 行手动 `HandleFunc` → 一张声明式路由表
- 每个 DSL 组件有独立的可读性测试

---

## 第五章 ATDD — 验收测试驱动开发

### 5.1 理论要点

```
ATDD 关注"完成标准"（Definition of Done）：
在开发前定义验收条件，只有通过全部验收条件才算完成。

ATDD 与 BDD 的区别：
- BDD 关注行为描述（Given-When-Then）
- ATDD 关注验收条件（可交付的标准）
- ATDD 通常在 API/集成层面执行，而非单元层面
```

### 5.2 当前差距

当前 `tests/backend/` 有丰富的 API 集成测试（26 个文件），但：

1. **缺少验收标准文档**：测试通过 ≠ 满足验收条件
2. **缺少端到端验收场景**：没有 Issue 从创建到合并的完整链路测试
3. **缺少非功能性验收**：无并发安全、性能、降级策略的验收测试
4. **缺少 shell ↔ backend 联调验收**：`.claude/` 与 `claude_tap_plus/` 的集成点无验收

### 5.3 重构目标

1. 为每条 API 端点定义明确的验收条件
2. 端到端验收测试覆盖 Issue 全链路
3. 并发安全验收（多 session 竞争 claim）
4. 降级验收（backend 不可用时 shell 侧优雅降级）

### 5.4 验收条件定义

#### API 端点验收表

| 端点 | 方法 | 验收条件 | 已有测试 |
|------|------|---------|---------|
| `/health` | GET | 返回 200 + JSON `{"status":"healthy"}` | ✓ |
| `/api/issue/check` | POST | 批量返回每个 issue 的状态和持有者 | ✓ |
| `/api/issue/claim` | POST | 原子领取，冲突时返回当前持有者 | ✓ |
| `/api/issue/release` | POST | 释放非终态 issue，终态拒绝 | ✓ |
| `/api/issue/release-session` | POST | 释放指定 session 的全部非终态 issue | ✓ |
| `/api/issue/status` | POST | 合法流转返回前状态，非法返回错误 | ✓ |
| `/api/session/register` | POST | 首次成功，重复返回 409 | ✓ |
| `/api/session/close` | POST | 标记关闭，已关闭返回 404 | ✓ |
| `/api/proxy/trace-init` | POST | 写入 proxy.json | ✓ |

#### 端到端验收场景

**场景 E2E-1：Issue 完整链路**

```
验收条件：
1. POST /api/issue/claim     → 成功领取 issue #42
2. POST /api/issue/status    → claimed → fixing（返回 previous_status=claimed）
3. POST /api/issue/status    → fixing → ready-for-pr
4. POST /api/issue/status    → ready-for-pr → pr-created
5. POST /api/issue/status    → pr-created → testing
6. POST /api/issue/status    → testing → reviewing
7. POST /api/issue/status    → reviewing → merged（终态）
8. POST /api/issue/release   → 返回 false（终态不可释放）
9. GET  /api/issues?status=merged → 列表中包含 issue #42
```

**场景 E2E-2：Session 生命周期 + 自动释放**

```
验收条件：
1. POST /api/session/register → session-A 注册成功
2. POST /api/issue/claim      → session-A 领取 issue #1, #2, #3
3. POST /api/session/close    → session-A 关闭
4. POST /api/issue/release-session → issue #1, #2, #3 全部释放
5. POST /api/issue/claim      → session-B 可以领取 issue #1
```

**场景 E2E-3：并发竞争**

```
验收条件：
1. 10 个 goroutine 同时 claim 同一个 issue
2. 恰好 1 个成功，9 个失败
3. 成功者的 session_id 与 issue 持有者一致
```

**场景 E2E-4：Backend 降级**

```
验收条件：
1. 启动 shell 侧（模拟 .claude/hooks/29-session-end/base.sh）
2. 调用 POST /api/issue/release-session → backend 未启动
3. shell 侧不崩溃，继续执行
4. 启动 backend → 重复调用 → 正常工作
```

### 5.5 实施方案

#### 端到端测试文件

```
tests/acceptance/
├── issue_e2e_test.go        ← 新增：E2E-1 Issue 完整链路
├── session_e2e_test.go      ← 新增：E2E-2 Session 生命周期
├── concurrency_e2e_test.go  ← 新增：E2E-3 并发竞争
├── degradation_e2e_test.go  ← 新增：E2E-4 Backend 降级
└── helper.go                ← 新增：启动真实 backend server 的测试辅助
```

#### 验收测试辅助

```go
// tests/acceptance/helper.go
type AcceptanceTestEnv struct {
    Server *backend.Server
    Store  store.Store
    BaseURL string
}

// SetupAcceptanceTest 启动真实的 backend server（随机端口）用于端到端验收。
func SetupAcceptanceTest(t *testing.T) *AcceptanceTestEnv {
    t.Helper()
    // 1. 创建临时 SQLite DB
    // 2. 启动 backend server 在随机端口
    // 3. t.Cleanup 清理
    // 4. 返回 BaseURL 供 HTTP 测试使用
}

// APIClient 是验收测试的 HTTP 客户端。
func (env *AcceptanceTestEnv) APIClient() *APIClient {
    return &APIClient{BaseURL: env.BaseURL}
}
```

#### 验收测试示例

```go
// tests/acceptance/issue_e2e_test.go
func TestIssueCompleteLifecycle(t *testing.T) {
    env := SetupAcceptanceTest(t)
    api := env.APIClient()

    // 验收条件 1：成功领取
    claim := api.ClaimIssue(t, "owner/repo", 42, "session-A", "Bug fix")
    require.True(t, claim.Success, "验收条件1: 首次领取应成功")

    // 验收条件 2-7：正向状态流转
    transitions := []struct{ from, to string }{
        {"claimed", "fixing"},
        {"fixing", "ready-for-pr"},
        {"ready-for-pr", "pr-created"},
        {"pr-created", "testing"},
        {"testing", "reviewing"},
        {"reviewing", "merged"},
    }
    for i, tr := range transitions {
        result := api.UpdateStatus(t, "owner/repo", 42, "session-A", tr.to)
        assert.Equal(t, tr.from, result.PreviousStatus,
            "验收条件%d: %s → %s", i+2, tr.from, tr.to)
    }

    // 验收条件 8：终态不可释放
    released := api.ReleaseIssue(t, "owner/repo", 42, "session-A")
    assert.False(t, released, "验收条件8: 终态 issue 不可释放")

    // 验收条件 9：列表中包含已合并的 issue
    issues := api.ListIssues(t, "owner/repo", "merged")
    assert.Contains(t, issues, 42, "验收条件9: 列表包含已合并 issue")
}
```

### 5.6 验证标准

- 每条 API 端点有对应的验收条件表（本文档即验收规格）
- 4 个端到端场景全部通过
- 并发测试用 `-race` flag 运行无 data race
- 降级测试验证 shell 侧在 backend 不可用时不崩溃

---

## 实施路线图

### 分阶段执行顺序

```
Phase 0：基础设施        ──────────────────────── 1-2 天
  ├─ 补齐 e2e fixture（让全部测试 PASS）
  ├─ 创建 testutil/testdb.go
  └─ 创建 testutil/mock_*.go

Phase 1：TDD + BDD        ──────────────────────── 5-8 天
  ├─ Store 层单元测试（Phase 1a）
  ├─ Service 层单元测试（Phase 1b）
  ├─ BDD 场景测试（Phase 1c）
  └─ Proxy 核心路径测试（Phase 1d）
  注：先有测试保护网，再做后续重构

Phase 2：DDD              ──────────────────────── 5-7 天
  ├─ 富领域模型 IssueClaim（Phase 2a）
  ├─ 富领域模型 Session（Phase 2b）
  ├─ 领域错误类型（Phase 2c）
  ├─ 统一 domain/store 类型（Phase 2d）
  └─ 重构 StatusService（Phase 2e）
  注：每步重构后运行 Phase 1 的全部测试验证无回归

Phase 3：DSL              ──────────────────────── 3-4 天
  ├─ Proxy Option 模式（Phase 3a）
  ├─ 状态机 TransitionTable（Phase 3b）
  └─ 路由注册声明式（Phase 3c）

Phase 4：ATDD             ──────────────────────── 3-5 天
  ├─ 端到端验收测试（Phase 4a）
  ├─ 并发安全验收（Phase 4b）
  └─ 降级验收（Phase 4c）
```

### 总工时估算

| 阶段 | 工时 | 前置依赖 |
|------|------|---------|
| Phase 0 基础设施 | 1-2 天 | 无 |
| Phase 1 TDD + BDD | 5-8 天 | Phase 0 |
| Phase 2 DDD | 5-7 天 | Phase 1 |
| Phase 3 DSL | 3-4 天 | Phase 2（状态机 DSL 依赖 DDD 富模型） |
| Phase 4 ATDD | 3-5 天 | Phase 2（验收测试依赖重构后的接口） |
| **合计** | **17-26 天** | — |

### 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| 重构引入回归 bug | 高 | Phase 1 的测试保护网确保每个重构步骤可验证 |
| DDD 重构涉及大量文件改动 | 中 | 按 Issue → Session → Stats 顺序逐个聚合推进 |
| mock 手写维护成本 | 低 | mock 只在 service 层使用，store 层用内存 DB 直接测试 |
| 五个理论冲突 | 低 | TDD/BDD 是测试策略（Phase 1），DDD 是设计策略（Phase 2），DSL 是表达策略（Phase 3），ATDD 是验收策略（Phase 4），互不冲突 |

---

## 附录：五理论关系图

```
                    ┌──────────────────────────┐
                    │    ATDD 验收驱动          │  ← 定义"什么时候算做完"
                    │  （Phase 4: 验收测试）     │
                    └────────────┬─────────────┘
                                 │
                    ┌────────────▼─────────────┐
                    │     BDD 行为驱动          │  ← 定义"系统应该做什么"
                    │  （Phase 1c: 场景测试）    │
                    └────────────┬─────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                  │
   ┌──────────▼──────┐  ┌───────▼────────┐  ┌──────▼─────────┐
   │  TDD 测试驱动    │  │  DDD 领域驱动   │  │  DSL 领域语言   │
   │ (Phase 1: 测试) │  │ (Phase 2: 设计) │  │ (Phase 3: 表达) │
   │                 │  │                 │  │                │
   │ 先测后写        │  │ 富模型+限界上下文│  │ 声明式规则      │
   └─────────────────┘  └─────────────────┘  └────────────────┘
```

五者在时间轴上的关系：
- **TDD** 是编码纪律（先写测试）— 贯穿全过程
- **BDD** 是规格表达（用场景描述行为）— 贯穿全过程
- **DDD** 是设计哲学（让代码表达业务）— Phase 2 集中实施
- **DSL** 是表达手段（让代码读起来像自然语言）— Phase 3 集中实施
- **ATDD** 是交付标准（定义完成条件）— Phase 4 最终验收
