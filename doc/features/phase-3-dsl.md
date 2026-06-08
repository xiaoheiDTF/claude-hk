# Phase 3：DSL — 领域特定语言

> **前置依赖**：Phase 2 完成（富领域模型就位）
> **预估工时**：3-4 天
> **目标**：用声明式/函数式模式提升代码可读性，将隐式规则变为显式表达

---

## 一、任务清单

### 阶段 3a：Proxy Option 模式（1-1.5 天）

#### 任务 3a.1：定义 ProxyOption 类型

| 序号 | 操作 | 文件 | 说明 |
|------|------|------|------|
| 3a.1.1 | 新增 `ProxyOption func(*ReverseProxy)` 类型 | `internal/proxy/options.go` | 新文件 |
| 3a.1.2 | 新增 `WithModel(model string) ProxyOption` | 同上 | 设置 `p.model` |
| 3a.1.3 | 新增 `WithFallback(cfg FallbackConfig) ProxyOption` | 同上 | 设置 `p.fallbackConfig` |
| 3a.1.4 | 新增 `WithKimiMode() ProxyOption` | 同上 | 设置 `p.kimiMode = true` |
| 3a.1.5 | 新增 `WithSessionInit(fn func(string, string)) ProxyOption` | 同上 | 设置 `p.OnSessionInit` |

---

#### 任务 3a.2：改造 NewReverseProxy 签名

| 序号 | 操作 | 文件 | 说明 |
|------|------|------|------|
| 3a.2.1 | `NewReverseProxy` 新增 `opts ...ProxyOption` 参数 | `internal/proxy/reverse.go` | 修改 |
| 3a.2.2 | 构造函数末尾循环应用 `opts` | 同上 | `for _, opt := range opts { opt(p) }` |
| 3a.2.3 | 删除 `SetModel`、`SetFallback`、`SetKimiMode` 等分散的 setter | 同上 | 标记 deprecated 或删除 |
| 3a.2.4 | 更新 `cmd/claude-tap/main.go` 的调用方式 | `cmd/claude-tap/main.go` | 改为 Option 调用 |

**改造前后对比**：

```go
// 改造前（main.go）
rp := proxy.NewReverseProxy(target, traceDir)
if model != "" { rp.SetModel(model) }
if fallbackCfg != nil { rp.SetFallback(*fallbackCfg) }
if kimiMode { rp.SetKimiMode() }
rp.OnSessionInit = func(sessionID, projectSlug string) { ... }

// 改造后（main.go）
opts := []proxy.ProxyOption{
    proxy.WithSessionInit(onSessionInit),
}
if model != "" { opts = append(opts, proxy.WithModel(model)) }
if fallbackCfg != nil { opts = append(opts, proxy.WithFallback(*fallbackCfg)) }
if kimiMode { opts = append(opts, proxy.WithKimiMode()) }
rp := proxy.NewReverseProxy(target, traceDir, opts...)
```

---

#### 任务 3a.3：Option 模式测试

| 序号 | 测试函数 | 场景 | 文件 |
|------|---------|------|------|
| 3a.3.1 | `TestProxyOptions` | 全部 4 个 Option 分别设置成功 | `internal/proxy/options_test.go` |
| 3a.3.2 | — | 无 Option 时使用默认值 | 同上 |
| 3a.3.3 | — | 多个 Option 组合 | 同上 |
| 3a.3.4 | `TestNewReverseProxyWithOptions` | 构造后验证字段值 | 同上 |

---

### 阶段 3b：状态机 TransitionTable DSL（1 天）

> Phase 2 已定义了 `TransitionTable` 类型。本阶段增加 DSL 属性：自检、可视化、不变性。

#### 任务 3b.1：增强 TransitionTable

| 序号 | 操作 | 文件 | 说明 |
|------|------|------|------|
| 3b.1.1 | 新增 `String()` 方法（格式化输出流转图） | `internal/backend/domain/issue.go` | 人类可读的状态机描述 |
| 3b.1.2 | 新增 `AllSourceStatuses() []IssueStatus` | 同上 | 返回所有源状态 |
| 3b.1.3 | 新增 `AllTargetStatuses(source) []IssueStatus` | 同上 | 返回指定源的所有目标 |
| 3b.1.4 | 增强 `Validate()` 检查孤立状态 | 同上 | 有定义但无入边的非初始状态 |

**String() 输出示例**：

```
Issue State Machine:
  idle → [claimed]
  claimed → [fixing, idle]
  fixing → [ready-for-pr, idle]
  ready-for-pr → [pr-created, idle]
  pr-created → [testing, idle]
  testing → [reviewing, idle]
  reviewing → [merged, rejected, idle]
  rejected → [fixing, idle]
  merged → [] (terminal)
```

---

#### 任务 3b.2：状态机 DSL 测试

| 序号 | 测试函数 | 场景 | 文件 |
|------|---------|------|------|
| 3b.2.1 | `TestTransitionTableString` | 输出包含全部 9 种状态 | `internal/backend/domain/issue_test.go` 扩展 |
| 3b.2.2 | `TestTransitionTableValidate` | 合法表通过，篡改后报错 | 同上 |
| 3b.2.3 | `TestTransitionTableQueries` | `AllSourceStatuses`/`AllTargetStatuses` | 同上 |

---

### 阶段 3c：路由注册声明式（1-1.5 天）

#### 任务 3c.1：定义 RouteDef 和 Routes() 方法

| 序号 | 操作 | 文件 | 说明 |
|------|------|------|------|
| 3c.1.1 | 新增 `RouteDef` struct | `internal/backend/api/routes.go` | `Method`, `Path`, `Handler` |
| 3c.1.2 | 新增 `Handlers.Routes() []RouteDef` | 同上 | 声明式路由表 |
| 3c.1.3 | 新增 `NewRouterFromDefs(defs []RouteDef) http.Handler` | 同上 | 循环注册 |

---

#### 任务 3c.2：重构 NewRouter

| 序号 | 操作 | 文件 | 说明 |
|------|------|------|------|
| 3c.2.1 | `NewRouter` 改为调用 `Handlers.Routes()` + `NewRouterFromDefs` | `internal/backend/api/router.go` | 修改 |
| 3c.2.2 | 保留动态路由处理（`/api/session/` 前缀的子路由分发） | 同上 | 用 `RouteDef` + 自定义匹配器 |

**当前动态路由问题**：

```go
// 当前 router.go:41-53 — /api/session/ 前缀的动态路由
mux.HandleFunc("/api/session/", func(w http.ResponseWriter, r *http.Request) {
    path := r.URL.Path
    switch {
    case strings.HasSuffix(path, "/issues"):
        h.Session.GetIssues(w, r)
    case strings.HasSuffix(path, "/tokens"):
        h.Session.GetTokens(w, r)
    case strings.HasSuffix(path, "/traces"):
        h.Session.GetTraces(w, r)
    default:
        h.Session.Get(w, r)
    }
})
```

**处理方案**：动态路由仍用 `HandleFunc("/api/session/", ...)` 特殊处理，但在 `Routes()` 中标注：

```go
// RouteDef 中增加 PatternType 字段
type RouteDef struct {
    Method      string
    Path        string
    Handler     http.HandlerFunc
    IsPrefix    bool   // true 表示前缀匹配（如 /api/session/）
    Description string // 路由描述（DSL 文档）
}

// Routes() 中的声明
{"GET", "/api/session/{id}", h.Session.Get, false, "获取单个 session"},
{"GET", "/api/session/{id}/issues", h.Session.GetIssues, false, "获取 session 的 issues"},
{"GET", "/api/session/{id}/tokens", h.Session.GetTokens, false, "获取 session 的 tokens"},
{"GET", "/api/session/{id}/traces", h.Session.GetTraces, false, "获取 session 的 traces"},
```

> 注意：Go 1.22 的 `http.ServeMux` 已原生支持 `{id}` 路径参数。如果项目 Go 版本 >= 1.22，可直接利用；否则用前缀匹配。

---

#### 任务 3c.3：路由测试

| 序号 | 测试函数 | 场景 | 文件 |
|------|---------|------|------|
| 3c.3.1 | `TestRoutesCompleteness` | `Routes()` 返回的数量与 `NewRouter` 注册的路由数一致 | `internal/backend/api/routes_test.go` |
| 3c.3.2 | `TestRouteDefinitions` | 每个 RouteDef 的 Method/Path 非空 | 同上 |
| 3c.3.3 | `TestNoDuplicateRoutes` | 无重复 (Method, Path) 组合 | 同上 |
| 3c.3.4 | API 集成测试无回归 | `go test ./tests/backend/...` | 已有测试 |

---

## 二、修改原则

### 原则 P3-1：DSL 不改变运行时行为

- Option 模式只是构造方式的改变，`ReverseProxy` 的字段和行为不变
- `TransitionTable.String()` 是纯输出，不影响状态机逻辑
- `RouteDef` 只是注册方式的改变，路由匹配规则不变
- **原则**：DSL 是表达层的改进，不是功能变更

### 原则 P3-2：向后兼容

- `NewReverseProxy(target, traceDir)` 的旧调用方式仍然可用（`opts` 为可变参数，空时等价于旧行为）
- 旧的 `SetModel()` 等方法标记 deprecated 但保留一个版本，下一版本再删除
- `NewRouter` 的签名不变（仍接受 `Handlers` 参数）

### 原则 P3-3：声明式优于命令式

- 路由表声明在一处（`Routes()`），不在多处注册
- 状态机规则声明在一处（`IssueTransitions`），不散落在 if-else 中
- Option 声明式构造优于 setter 调用序列

### 原则 P3-4：DSL 可测试

- `TransitionTable.Validate()` 是 DSL 的自检机制
- `Routes()` 返回值可被测试验证（数量、唯一性）
- `ProxyOption` 的每个 With 函数可独立测试

### 原则 P3-5：代码即文档

- `RouteDef.Description` 字段提供人类可读的路由描述
- `TransitionTable.String()` 输出可作为文档
- Option 函数名即配置说明（`WithFallback` 比 `SetFallbackConfig` 更清晰）

---

## 三、约束条件

### 约束 C3-1：不引入外部 DSL 框架

- 不使用 Go code generation、不使用 protobuf、不使用 OpenAPI spec 生成路由
- 纯 Go 代码实现的内部 DSL
- 不引入任何新的第三方依赖

### 约束 C3-2：ReverseProxy 行为不变

- Option 模式改造后，proxy 的转发逻辑、SSE 处理、trace 记录 **全部不变**
- 只改构造方式，不改运行逻辑
- e2e 测试（`tests/e2e/`）必须全部 PASS

### 约束 C3-3：路由表不可遗漏

- 从 `NewRouter` 提取到 `Routes()` 时，每一条路由都必须对应
- `TestRoutesCompleteness` 必须验证路由数量与原始注册数一致
- 如有动态路由（前缀匹配），在 RouteDef 中标注 `IsPrefix=true`

### 约束 C3-4：Option 不可有副作用

- 每个 `ProxyOption` 函数只修改 `ReverseProxy` 的字段，不做 I/O
- 不在 Option 中启动 goroutine、打开文件、发起网络请求
- Option 应用时机：构造函数内部，在 proxy.Start() 之前

### 约束 C3-5：String() 输出稳定

- `TransitionTable.String()` 的输出格式在版本内保持稳定
- 输出只用于日志/文档，不被代码解析
- 输出编码为 UTF-8，不含特殊字符

---

## 四、产出物

| 产出物 | 路径 | 操作 |
|--------|------|------|
| Proxy Option 类型+函数 | `internal/proxy/options.go` | 新增文件 |
| Proxy Option 测试 | `internal/proxy/options_test.go` | 新增文件 |
| ReverseProxy 构造改造 | `internal/proxy/reverse.go` | 修改（签名+删除 setter） |
| main.go 调用方式更新 | `cmd/claude-tap/main.go` | 修改 |
| TransitionTable 增强 | `internal/backend/domain/issue.go` | 新增方法 |
| 路由声明式注册 | `internal/backend/api/routes.go` | 新增文件 |
| 路由注册重构 | `internal/backend/api/router.go` | 修改 |
| 路由测试 | `internal/backend/api/routes_test.go` | 新增文件 |

---

## 五、验收检查点

```bash
# 检查点 1：Proxy e2e 无回归
go test ./tests/e2e/... -v -count=1
# 期望：全部 PASS（Option 模式不改变转发行为）

# 检查点 2：Proxy 单元测试
go test ./internal/proxy/... -v
# 期望：Option 测试全部 PASS，已有测试无回归

# 检查点 3：状态机 String() 输出
go test ./internal/backend/domain/... -run "TestTransitionTableString" -v
# 期望：输出包含全部 9 种状态

# 检查点 4：路由测试
go test ./internal/backend/api/... -run "TestRoutes" -v
# 期望：完整性、唯一性测试通过

# 检查点 5：API 集成测试无回归
go test ./tests/backend/... -count=1
# 期望：26 个 API 测试全部 PASS

# 检查点 6：全部测试
go test ./...
# 期望：0 FAIL
```
