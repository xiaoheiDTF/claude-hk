# Phase 2：DDD — 领域驱动设计重构

> **前置依赖**：Phase 1 完成（测试保护网就位）
> **预估工时**：5-7 天
> **目标**：将贫血模型重构为富领域模型，消除类型重复，引入领域错误，拆解 God Service

---

## 一、任务清单

### 阶段 2a：富领域模型 — IssueClaim（2 天）

#### 任务 2a.1：新增状态机 TransitionTable

| 序号 | 操作 | 文件 | 说明 |
|------|------|------|------|
| 2a.1.1 | 新增 `TransitionTable` 类型 | `internal/backend/domain/issue.go` | `map[IssueStatus][]IssueStatus` |
| 2a.1.2 | 新增 `IssueTransitions` 变量（声明式流转表） | 同上 | 包含全部 9 种状态的合法流转 |
| 2a.1.3 | 新增 `Allowed(source, target)` 方法 | 同上 | TransitionTable 上的查询方法 |
| 2a.1.4 | 新增 `Validate()` 方法 | 同上 | 流转表自检（测试时调用） |
| 2a.1.5 | 新增 `IsTerminal()` 方法 | 同上 | `IssueMerged` 为唯一终态 |
| 2a.1.6 | 新增 `CanTransitionTo(target)` 方法 | 同上 | 委托给 `IssueTransitions.Allowed()` |

**状态流转表定义**：

```
idle        → [claimed]
claimed     → [fixing, idle]
fixing      → [ready-for-pr, idle]
ready-for-pr → [pr-created, idle]
pr-created  → [testing, idle]
testing     → [reviewing, idle]
reviewing   → [merged, rejected, idle]
rejected    → [fixing, idle]
merged      → []（终态）
```

> 注意：每个非终态都允许回到 `idle`（代表放弃/释放）。`merged` 是唯一终态，`rejected` 不是终态（可重新修复）。

**修改方式**：在 `domain/issue.go` 的 `IssueStatus` 类型上新增方法，**不修改已有 `IssueClaim` struct 的字段**。

---

#### 任务 2a.2：新增 IssueClaim 行为方法

| 序号 | 操作 | 文件 | 说明 |
|------|------|------|------|
| 2a.2.1 | 新增 `IsOwnedBy(sessionID) bool` | `internal/backend/domain/issue.go` | 判断持有者 |
| 2a.2.2 | 新增 `Claim(sessionID) error` | 同上 | 封装领取逻辑 |
| 2a.2.3 | 新增 `Release(sessionID) error` | 同上 | 封装释放逻辑 |
| 2a.2.4 | 新增 `TransitionStatus(sessionID, target) error` | 同上 | 封装状态流转 |

**方法实现要点**：

- `Claim()`：幂等（同 session 重复领取返回 nil），冲突时返回 `ErrIssueAlreadyClaimed`
- `Release()`：只允许持有者释放，终态不可释放
- `TransitionStatus()`：只允许持有者流转，必须符合 `TransitionTable` 规则
- 所有方法 **修改 struct 自身状态**，不涉及 DB 操作（纯内存逻辑）

---

#### 任务 2a.3：新增领域错误类型

| 序号 | 操作 | 文件 | 说明 |
|------|------|------|------|
| 2a.3.1 | 新增 `ErrIssueAlreadyClaimed` struct | `internal/backend/domain/errors.go` | 含 `Owner`、`Status` 字段 |
| 2a.3.2 | 新增 `ErrNotIssueOwner` struct | 同上 | 无字段 |
| 2a.3.3 | 新增 `ErrIssueTerminal` struct | 同上 | 无字段 |
| 2a.3.4 | 新增 `ErrInvalidTransition` struct | 同上 | 含 `From`、`To` 字段 |

**文件 `domain/errors.go`** — 全部新增，无修改已有代码。

---

#### 任务 2a.4：领域模型单元测试

| 序号 | 测试函数 | 场景 | 文件 |
|------|---------|------|------|
| 2a.4.1 | `TestTransitionTable` | 全部 9 种状态的合法/非法流转 | `internal/backend/domain/issue_test.go` |
| 2a.4.2 | `TestTransitionTableValidate` | `Validate()` 自检通过 | 同上 |
| 2a.4.3 | `TestIssueStatusIsTerminal` | merged=true, 其余=false | 同上 |
| 2a.4.4 | `TestIssueClaimClaim` | 首次/重复/冲突 | 同上 |
| 2a.4.5 | `TestIssueClaimRelease` | 正常/终态/非持有者 | 同上 |
| 2a.4.6 | `TestIssueClaimTransitionStatus` | 正向流转/非法流转/非持有者 | 同上 |
| 2a.4.7 | `TestDomainErrors` | 错误类型断言（`errors.As`） | 同上 |

```go
// 示例：状态机全量测试
func TestTransitionTable(t *testing.T) {
    // 遍历每一种 (source, target) 组合
    for source, targets := range domain.IssueTransitions {
        t.Run(string(source), func(t *testing.T) {
            // 验证合法目标
            for _, target := range targets {
                assert.True(t, source.CanTransitionTo(target),
                    "%s should transition to %s", source, target)
            }
            // 验证非法目标（不在列表中的状态）
            allStatuses := []domain.IssueStatus{
                domain.IssueIdle, domain.IssueClaimed, domain.IssueFixing,
                domain.IssueReadyForPR, domain.IssuePRCreated,
                domain.IssueTesting, domain.IssueReviewing,
                domain.IssueMerged, domain.IssueRejected,
            }
            for _, s := range allStatuses {
                found := false
                for _, target := range targets {
                    if target == s { found = true; break }
                }
                if !found {
                    assert.False(t, source.CanTransitionTo(s),
                        "%s should NOT transition to %s", source, s)
                }
            }
        })
    }
}
```

---

### 阶段 2b：富领域模型 — Session（1 天）

#### 任务 2b.1：Session 行为方法

| 序号 | 操作 | 文件 | 说明 |
|------|------|------|------|
| 2b.1.1 | 新增 `IsActive() bool` | `internal/backend/domain/session.go` | 判断是否 active |
| 2b.1.2 | 新增 `Close(reason string) error` | 同上 | 关闭 session，设置 ClosedAt |
| 2b.1.3 | 新增 `IsTimedOut(threshold time.Duration) bool` | 同上 | 判断是否超时 |

**领域错误**（在 `domain/errors.go` 中追加）：

| 序号 | 错误类型 | 说明 |
|------|---------|------|
| 2b.1.4 | `ErrSessionAlreadyClosed` | 关闭已关闭的 session |

#### 任务 2b.2：Session 领域测试

| 序号 | 测试函数 | 场景 | 文件 |
|------|---------|------|------|
| 2b.2.1 | `TestSessionClose` | 正常关闭/已关闭 | `internal/backend/domain/session_test.go` |
| 2b.2.2 | `TestSessionIsTimedOut` | 未超时/已超时 | 同上 |

---

### 阶段 2c：统一类型定义（1-2 天）

#### 任务 2c.1：消除 store/store.go 与 domain/ 的重复

**当前重复**：

| 类型 | domain/ 定义 | store/ 定义 | 差异 |
|------|-------------|------------|------|
| Session | `domain.Session` (有 JSON tag) | `store.Session` (无 JSON tag) | 字段完全相同 |
| IssueStatus | `domain.IssueStatus` | store 中用 `string` | 类型不同 |
| MachineID | `domain.Machine.MachineID` | store 中用 `string` | — |

**操作步骤**：

| 序号 | 操作 | 影响文件 | 风险 |
|------|------|---------|------|
| 2c.1.1 | 将 `store.Session` 的字段对齐到 `domain.Session`（添加 JSON tag） | `internal/backend/store/store.go` | 低 |
| 2c.1.2 | 让 `store.SessionStore` 接口返回 `domain.Session` | `internal/backend/store/store.go` | **中**：所有 session_store 实现和调用方需同步修改 |
| 2c.1.3 | 将 `store.ClaimResult` / `UpdateStatusResult` / `IssueListItem` / `IssueCheckResult` 迁移到 `domain/` | `domain/issue_dto.go` 新增 | 中 |
| 2c.1.4 | 更新 `store.IssueStore` 接口引用 `domain.Xxx` 类型 | `internal/backend/store/store.go` | 中 |
| 2c.1.5 | 更新 `store/issue_store.go` 实现的返回类型 | `internal/backend/store/issue_store.go` | 中 |
| 2c.1.6 | 更新 `service/` 层的 import 和类型引用 | `internal/backend/service/*.go` | 低 |
| 2c.1.7 | 更新 `api/` 层的 import 和类型引用 | `internal/backend/api/*.go` | 低 |

**关键**：每完成一步就运行 Phase 1 的全部测试，确保无回归。

---

### 阶段 2d：重构 StatusService（1 天）

#### 任务 2d.1：引入 StatsReadModel

| 序号 | 操作 | 文件 | 说明 |
|------|------|------|------|
| 2d.1.1 | 新增 `StatsReadModel` 接口 | `internal/backend/domain/stats.go` | `GetSystemStats(ctx) (*SystemStats, error)` |
| 2d.1.2 | 新增 `SystemStats` struct | 同上 | 5 个字段 |
| 2d.1.3 | 新增 `SQLiteStatsStore` 实现 | `internal/backend/store/stats_store.go` | 用 COUNT SQL 替代全表扫描 |
| 2d.1.4 | 重构 `StatusService` 依赖 | `internal/backend/service/status_service.go` | 5 个 store → 1 个 StatsReadModel |
| 2d.1.5 | 更新 `server.go` 的依赖注入 | `internal/backend/server.go` | 构造 `SQLiteStatsStore` 传给 StatusService |
| 2d.1.6 | 更新 `StatusService` 测试 | `internal/backend/service/status_service_test.go` | mock 1 个接口而非 5 个 |

**重构前后对比**：

```go
// 重构前
type StatusService struct {
    sessionStore store.SessionStore   // 依赖 1
    proxyStore   store.ProxyStore     // 依赖 2
    issueStore   store.IssueStore     // 依赖 3
    machineStore store.MachineStore   // 依赖 4
    projectStore store.ProjectStore   // 依赖 5
    startTime    time.Time
}

// 重构后
type StatusService struct {
    stats     domain.StatsReadModel   // 依赖 1
    startTime time.Time
}
```

---

### 阶段 2e：Service 层调用 Domain 方法（1 天）

#### 任务 2e.1：IssueService 调用 domain 行为

**重构前**（当前 IssueService 全部透传到 store）：

```go
func (svc *IssueService) Claim(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*store.ClaimResult, error) {
    return svc.store.ClaimIssue(ctx, repo, number, sessionID, issueTitle)
}
```

**重构后**（service 先用 domain 校验，再持久化）：

```go
func (svc *IssueService) Claim(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*domain.ClaimResult, error) {
    // 1. 从 store 加载当前状态
    claim, err := svc.store.GetIssue(ctx, repo, number)
    if err != nil {
        return nil, err
    }
    if claim == nil {
        claim = domain.NewIssueClaim(repo, number, issueTitle)
    }
    // 2. 调用 domain 方法校验并修改状态
    if err := claim.Claim(sessionID); err != nil {
        return &domain.ClaimResult{Success: false}, err
    }
    // 3. 持久化
    return svc.store.SaveIssue(ctx, claim)
}
```

> **注意**：这一步涉及 store 接口变更（需要 `GetIssue` / `SaveIssue`），是本 Phase 中 **风险最高** 的改动。需要权衡两种方案：

**方案 A（推荐：渐进式）**：domain 方法只做校验，store 仍用现有接口（claim/release/update 各自独立）。Service 层先调 domain 校验，再调 store 持久化。不修改 store 接口。

**方案 B（彻底式）**：store 暴露 `Get` / `Save` 泛用接口，service 完全编排。改动量大，风险高。

| 序号 | 操作 | 文件 | 说明 |
|------|------|------|------|
| 2e.1.1 | 确定方案（A 或 B） | — | 建议方案 A |
| 2e.1.2 | IssueService.Claim 加前置 domain 校验 | `internal/backend/service/issue_service.go` | 调用 `claim.Claim(sessionID)` 校验 |
| 2e.1.3 | IssueService.UpdateStatus 加前置 domain 校验 | 同上 | 调用 `claim.TransitionStatus()` 校验 |
| 2e.1.4 | IssueService.Release 加前置 domain 校验 | 同上 | 调用 `claim.Release()` 校验 |
| 2e.1.5 | 运行全部 Phase 1 测试 | — | 无回归 |
| 2e.1.6 | 补充 domain 校验相关的 service 测试 | `internal/backend/service/issue_service_test.go` | 新增 domain 错误传播测试 |

---

## 二、修改原则

### 原则 P2-1：每步可验证

- 每完成一个任务（2a.1、2a.2、...），立即运行 `go test ./...` 验证无回归
- 如果测试失败，**立即回滚**到上一个通过状态，不可"先做完再统一修"
- 理由：DDD 重构涉及大量文件改动，积累越多越难定位问题

### 原则 P2-2：新增优先于修改

- 领域行为方法全部是 **新增**（`func (c *IssueClaim) Claim(...) error`）
- 不修改已有 struct 的字段定义
- 不删除已有的函数/方法（只在确认无调用方后才能删除）
- `domain/errors.go` 全新文件

### 原则 P2-3：类型迁移分步走

统一类型定义（阶段 2c）必须按以下顺序执行：

```
1. domain/ 中新增 DTO 类型（ClaimResult 等）
2. store/ 接口改为引用 domain 类型
3. store/ 实现改为返回 domain 类型
4. service/ 更新 import
5. api/ 更新 import
6. 删除 store/ 中的旧类型定义
7. 每一步后运行测试
```

不可跳步。不可一次修改多个步骤再测试。

### 原则 P2-4：领域错误穿透

- domain 层返回的错误类型必须在 service 层 **原样传播**，不吞掉、不包装成新类型
- API 层通过 `errors.As()` 匹配领域错误，返回对应的 HTTP 状态码
- 不在 service 层把 `ErrIssueAlreadyClaimed` 转成 `fmt.Errorf("claim: %w", err)` — 可以 wrap 但必须保留原始类型

```go
// ✓ 正确：API 层直接匹配领域错误
var errDomain domain.ErrIssueAlreadyClaimed
if errors.As(err, &errDomain) {
    w.WriteHeader(409) // Conflict
    json.NewEncoder(w).Encode(response{Error: err.Error()})
}

// ✗ 错误：service 层吞掉领域错误
return nil, fmt.Errorf("claim failed") // 丢失了 Owner/Status 信息
```

### 原则 P2-5：Service 层不做领域校验

- 领域校验逻辑（谁能 claim、哪些状态可以流转）全部在 `domain/` 中
- Service 层只做：加载 domain 对象 → 调用 domain 方法 → 持久化结果
- Service 层不包含任何 `if status == "claimed"` 之类的业务判断

---

## 三、约束条件

### 约束 C2-1：不可破坏已有 API 契约

- HTTP 端点的 URL、请求/响应 JSON 格式 **不可改变**
- API 层（`internal/backend/api/`）的 handler 函数签名不可改变
- 如果 domain 类型迁移影响了 API 响应格式，必须在 api 层做适配（adapter）
- `go test ./tests/backend/...` 必须全部 PASS（这些是 API 集成测试）

### 约束 C2-2：不可引入新依赖

- 不引入新的 Go 第三方库
- `domain/` 包只能依赖 Go 标准库
- `domain/` 不能 import `store/` 或 `service/`（依赖方向：domain ← store ← service ← api）

### 约束 C2-3：domain 包不可有 I/O

- `domain/` 中的方法不能做 DB 读写、HTTP 调用、文件操作
- domain 方法签名只接受/返回纯内存类型
- 例子：`IssueClaim.Claim()` 修改 struct 字段，返回 error；DB 保存由 service 调用 store 完成

### 约束 C2-4：StatusService 重构必须向后兼容

- `GET /api/status` 的响应 JSON 格式不变
- `SystemStats` 的字段名和类型与现有 `StatusService.Get()` 返回的结构一致
- 如果字段名变化，在 API 层做 JSON alias

### 约束 C2-5：迁移期间可共存

- 新旧类型定义可以临时共存（`store.ClaimResult` 和 `domain.ClaimResult` 同时存在）
- 共存期不超过 1 天，完成迁移后立即删除旧定义
- 共存期所有测试必须通过

### 约束 C2-6：不修改 Proxy 侧

- 本 Phase 只重构 `internal/backend/` 下的代码
- `internal/proxy/`、`internal/sse/`、`internal/trace/`、`internal/usage/` 不动
- `cmd/claude-tap/main.go` 只在 `server.go` 依赖注入变化时才改

---

## 四、产出物

| 产出物 | 路径 | 操作 |
|--------|------|------|
| 状态机 TransitionTable | `internal/backend/domain/issue.go` | 新增方法+变量 |
| IssueClaim 行为方法 | `internal/backend/domain/issue.go` | 新增方法 |
| Session 行为方法 | `internal/backend/domain/session.go` | 新增方法 |
| 领域错误类型 | `internal/backend/domain/errors.go` | 新增文件 |
| 统计读取模型 | `internal/backend/domain/stats.go` | 新增文件 |
| 领域模型测试 | `internal/backend/domain/issue_test.go` | 新增文件 |
| 领域模型测试 | `internal/backend/domain/session_test.go` | 新增文件 |
| StatsStore 实现 | `internal/backend/store/stats_store.go` | 新增文件 |
| StatusService 重构 | `internal/backend/service/status_service.go` | 修改 |
| Server 依赖注入 | `internal/backend/server.go` | 修改 |
| IssueService 重构 | `internal/backend/service/issue_service.go` | 修改 |
| Service 测试更新 | `internal/backend/service/*_test.go` | 修改 |
| DTO 类型迁移 | `internal/backend/domain/issue_dto.go` | 新增文件 |
| Store 接口更新 | `internal/backend/store/store.go` | 修改（类型引用） |
| Store 实现更新 | `internal/backend/store/issue_store.go` | 修改（返回类型） |

---

## 五、验收检查点

```bash
# 检查点 1：domain 层测试全通过
go test ./internal/backend/domain/... -v
# 期望：状态机全部流转场景、行为方法、领域错误测试全部 PASS

# 检查点 2：Store 层测试无回归
go test ./internal/backend/store/... -cover
# 期望：与 Phase 1 结果一致

# 检查点 3：Service 层测试无回归
go test ./internal/backend/service/... -cover
# 期望：全部 PASS，StatusService 测试改用 mock StatsReadModel

# 检查点 4：API 集成测试无回归
go test ./tests/backend/... -count=1
# 期望：26 个 API 测试全部 PASS（API 契约未变）

# 检查点 5：BDD 场景无回归
go test ./tests/backend/... -run "场景" -v
# 期望：13 个场景全部 PASS

# 检查点 6：StatusService 依赖简化
grep -c "store\." internal/backend/service/status_service.go
# 期望：从 ~10 处引用降到 ~2 处（构造+接口）

# 检查点 7：domain/ 无外部 import
grep -r "import" internal/backend/domain/*.go | grep -v "standard"
# 期望：domain/ 只 import 标准库（time, fmt, errors）
```
