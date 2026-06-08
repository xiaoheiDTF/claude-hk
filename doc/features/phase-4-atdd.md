# Phase 4：ATDD — 验收测试驱动

> **前置依赖**：Phase 2 完成（领域模型重构就位）
> **预估工时**：3-5 天
> **目标**：建立端到端验收测试体系，覆盖完整业务链路、并发安全、降级策略

---

## 一、任务清单

### 阶段 4a：端到端验收测试（1.5-2 天）

#### 任务 4a.1：创建验收测试基础设施

| 序号 | 操作 | 文件 | 说明 |
|------|------|------|------|
| 4a.1.1 | 创建 `tests/acceptance/` 目录 | `tests/acceptance/` | 新增目录 |
| 4a.1.2 | 新增 `AcceptanceTestEnv` struct | `tests/acceptance/helper.go` | 封装 server + store + baseURL |
| 4a.1.3 | 新增 `SetupAcceptanceTest(t) *AcceptanceTestEnv` | 同上 | 启动真实 backend（随机端口） |
| 4a.1.4 | 新增 `APIClient` struct | 同上 | 封装 HTTP 请求方法 |
| 4a.1.5 | 实现 `APIClient` 的各个方法 | 同上 | ClaimIssue、UpdateStatus、ReleaseIssue 等 |

**AcceptanceTestEnv 设计**：

```go
type AcceptanceTestEnv struct {
    Server  *backend.Server
    Store   store.Store
    BaseURL string      // 如 http://127.0.0.1:38217
    Client  *APIClient
}

type APIClient struct {
    BaseURL    string
    HTTPClient *http.Client
}

// APIClient 方法 — 每个方法封装 HTTP 调用 + 状态码检查
func (c *APIClient) ClaimIssue(t *testing.T, repo string, number int, sessionID, title string) *ClaimResponse
func (c *APIClient) UpdateStatus(t *testing.T, repo string, number int, sessionID, newStatus string) *UpdateStatusResponse
func (c *APIClient) ReleaseIssue(t *testing.T, repo string, number int, sessionID string) bool
func (c *APIClient) ReleaseSession(t *testing.T, sessionID string) []int
func (c *APIClient) RegisterSession(t *testing.T, s store.Session) error
func (c *APIClient) CloseSession(t *testing.T, sessionID, reason string) error
func (c *APIClient) ListIssues(t *testing.T, filter store.IssueFilter) ([]store.IssueListItem, int)
func (c *APIClient) GetStatus(t *testing.T) *domain.SystemStatus
func (c *APIClient) HealthCheck(t *testing.T) bool
```

---

#### 任务 4a.2：E2E-1 Issue 完整生命周期

| 序号 | 验收条件 | 测试代码 | 文件 |
|------|---------|---------|------|
| 4a.2.1 | claim issue #42 → Success=true | `api.ClaimIssue(t, "owner/repo", 42, "sess-A", "Bug fix")` | `tests/acceptance/issue_e2e_test.go` |
| 4a.2.2 | claimed → fixing, previous=claimed | `api.UpdateStatus(t, ..., "fixing")` | 同上 |
| 4a.2.3 | fixing → ready-for-pr | `api.UpdateStatus(t, ..., "ready-for-pr")` | 同上 |
| 4a.2.4 | ready-for-pr → pr-created | `api.UpdateStatus(t, ..., "pr-created")` | 同上 |
| 4a.2.5 | pr-created → testing | `api.UpdateStatus(t, ..., "testing")` | 同上 |
| 4a.2.6 | testing → reviewing | `api.UpdateStatus(t, ..., "reviewing")` | 同上 |
| 4a.2.7 | reviewing → merged（终态） | `api.UpdateStatus(t, ..., "merged")` | 同上 |
| 4a.2.8 | 终态 release → false | `api.ReleaseIssue(t, ..., "sess-A")` → false | 同上 |
| 4a.2.9 | 列表过滤 status=merged 包含 #42 | `api.ListIssues(t, {Status: "merged"})` | 同上 |

```go
func TestIssueCompleteLifecycle(t *testing.T) {
    env := SetupAcceptanceTest(t)
    api := env.Client

    t.Run("验收1: 首次领取", func(t *testing.T) {
        result := api.ClaimIssue(t, "owner/repo", 42, "sess-A", "Bug fix")
        assertEqual(t, result.Success, true)
    })

    t.Run("验收2-7: 正向流转", func(t *testing.T) {
        transitions := []struct{ from, to string }{
            {"claimed", "fixing"},
            {"fixing", "ready-for-pr"},
            {"ready-for-pr", "pr-created"},
            {"pr-created", "testing"},
            {"testing", "reviewing"},
            {"reviewing", "merged"},
        }
        for i, tr := range transitions {
            result := api.UpdateStatus(t, "owner/repo", 42, "sess-A", tr.to)
            assertEqual(t, result.PreviousStatus, tr.from,
                "验收%d: %s → %s", i+2, tr.from, tr.to)
        }
    })

    t.Run("验收8: 终态不可释放", func(t *testing.T) {
        released := api.ReleaseIssue(t, "owner/repo", 42, "sess-A")
        assertEqual(t, released, false)
    })

    t.Run("验收9: 列表包含已合并issue", func(t *testing.T) {
        items, _ := api.ListIssues(t, store.IssueFilter{
            RepoFullName: ptr("owner/repo"),
            Status:       ptr("merged"),
        })
        found := false
        for _, item := range items {
            if item.IssueNumber == 42 { found = true; break }
        }
        assertEqual(t, found, true)
    })
}
```

---

#### 任务 4a.3：E2E-2 Session 生命周期 + 自动释放

| 序号 | 验收条件 | 文件 |
|------|---------|------|
| 4a.3.1 | 注册 session-A | `tests/acceptance/session_e2e_test.go` |
| 4a.3.2 | session-A claim issue #1, #2, #3 | 同上 |
| 4a.3.3 | 关闭 session-A | 同上 |
| 4a.3.4 | release-session → 返回 [1, 2, 3] | 同上 |
| 4a.3.5 | session-B 可以 claim issue #1 | 同上 |

---

#### 任务 4a.4：E2E-3 多 Issue 交叉场景

| 序号 | 验收条件 | 文件 |
|------|---------|------|
| 4a.4.1 | session-A claim #1, #2；session-B claim #3, #4 | `tests/acceptance/cross_session_test.go` |
| 4a.4.2 | session-A 关闭 → #1, #2 释放；#3, #4 不受影响 | 同上 |
| 4a.4.3 | session-B 更新 #3 状态为 fixing → 成功 | 同上 |
| 4a.4.4 | session-C claim #1（已被释放）→ 成功 | 同上 |

---

### 阶段 4b：并发安全验收（1-1.5 天）

#### 任务 4b.1：并发 claim 竞争

| 序号 | 验收条件 | 文件 |
|------|---------|------|
| 4b.1.1 | 10 个 goroutine 同时 claim 同一 issue，恰好 1 个成功 | `tests/acceptance/concurrency_e2e_test.go` |
| 4b.1.2 | 成功者的 session_id 与 issue 持有者一致 | 同上 |
| 4b.1.3 | 9 个失败者都收到 ClaimedBy 信息 | 同上 |

```go
func TestConcurrentClaim(t *testing.T) {
    env := SetupAcceptanceTest(t)
    api := env.Client

    const workers = 10
    results := make(chan *ClaimResponse, workers)

    for i := 0; i < workers; i++ {
        go func(idx int) {
            result := api.ClaimIssue(t, "owner/repo", 99,
                fmt.Sprintf("session-%d", idx), "concurrent test")
            results <- result
        }(i)
    }

    successCount := 0
    var winner string
    for i := 0; i < workers; i++ {
        r := <-results
        if r.Success {
            successCount++
            winner = r.SessionID  // 假设响应包含 session_id
        }
    }

    assertEqual(t, successCount, 1, "恰好 1 个成功")
    // 验证 winner 是有效的 session
    assertNotEqual(t, winner, "")
}
```

---

#### 任务 4b.2：并发状态更新

| 序号 | 验收条件 | 文件 |
|------|---------|------|
| 4b.2.1 | 同一 session 并发更新同一 issue 的状态，最终状态一致 | `tests/acceptance/concurrency_e2e_test.go` |
| 4b.2.2 | 不同 session 并发更新同一 issue，只有持有者成功 | 同上 |

---

#### 任务 4b.3：并发 session 关闭

| 序号 | 验收条件 | 文件 |
|------|---------|------|
| 4b.3.1 | 同时关闭 session 和 claim issue，无 data race | `tests/acceptance/concurrency_e2e_test.go` |
| 4b.3.2 | 用 `-race` flag 运行无报警 | 同上 |

```bash
go test ./tests/acceptance/... -race -v -count=1
# 期望：无 DATA RACE 报警
```

---

### 阶段 4c：降级验收（1 天）

#### 任务 4c.1：Backend 不可用时 shell 侧降级

| 序号 | 验收条件 | 文件 |
|------|---------|------|
| 4c.1.1 | HTTP 请求到未启动的端口 → 连接拒绝 | `tests/acceptance/degradation_e2e_test.go` |
| 4c.1.2 | 模拟 shell 侧 `_backend_available()` 返回 false → 跳过 API 调用 | 同上 |
| 4c.1.3 | 启动 backend → 重复调用 → 正常工作 | 同上 |

**降级测试设计**：

由于 shell 侧的 `_backend_available()` 是 bash 脚本（在 `.claude/` 中），本测试用 Go 模拟 shell 的行为：

```go
func TestBackendDegradation(t *testing.T) {
    t.Run("backend不可用时HTTP调用失败但不崩溃", func(t *testing.T) {
        // 模拟 shell 侧的 _call_backend 行为
        // 连接到随机未监听端口
        _, err := http.Post("http://127.0.0.1:1/api/issue/release-session", ...)
        assert(t, err != nil, "连接应失败")
        // 关键：不 panic，不崩溃
    })

    t.Run("backend恢复后正常工作", func(t *testing.T) {
        env := SetupAcceptanceTest(t)
        // 正常 API 调用
        result := env.Client.HealthCheck(t)
        assertEqual(t, result, true)
    })
}
```

---

#### 任务 4c.2：Backend 异常响应时降级

| 序号 | 验收条件 | 文件 |
|------|---------|------|
| 4c.2.1 | Backend 返回 500 → shell 侧不崩溃 | `tests/acceptance/degradation_e2e_test.go` |
| 4c.2.2 | Backend 返回超时 → shell 侧不阻塞 | 同上 |
| 4c.2.3 | Backend 返回非 JSON → shell 侧解析错误不崩溃 | 同上 |

---

#### 任务 4c.3：Store 层错误时 API 降级

| 序号 | 验收条件 | 文件 |
|------|---------|------|
| 4c.3.1 | SQLite 文件权限错误 → API 返回 500，不 panic | `tests/acceptance/degradation_e2e_test.go` |
| 4c.3.2 | SQLite busy timeout → API 返回 503，不 panic | 同上 |

---

### 阶段 4d：验收文档生成（0.5 天）

#### 任务 4d.1：生成 API 验收条件文档

| 序号 | 操作 | 文件 | 说明 |
|------|------|------|------|
| 4d.1.1 | 汇总每条 API 的验收条件 | `tests/acceptance/ACCEPTANCE.md` | 从 RouteDef + 测试用例自动/手动生成 |
| 4d.1.2 | 汇总端到端场景的验收条件 | 同上 | 4 个 E2E 场景的通过标准 |
| 4d.1.3 | 汇总并发/降级验收条件 | 同上 | 非功能性验收标准 |

**ACCEPTANCE.md 格式**：

```markdown
# API 验收条件

## GET /health
- 返回 HTTP 200
- 响应体包含 {"status": "healthy"}
- 响应时间 < 50ms

## POST /api/issue/claim
- 首次领取成功时返回 HTTP 200 + {"success": true}
- 冲突时返回 HTTP 200 + {"success": false, "claimed_by": "..."}
- 请求体缺少必填字段返回 HTTP 400
...
```

---

## 二、修改原则

### 原则 P4-1：验收测试测行为不测实现

- 验收测试只通过 HTTP 接口与系统交互
- 不直接调用 store/service/domain 的内部方法
- 验收测试只关心"给我这个输入，我得到这个输出"

### 原则 P4-2：每个验收条件可追溯

- 测试名包含验收条件编号（`验收1:`、`验收2:`）
- 测试断言的 error message 包含验收条件描述
- `ACCEPTANCE.md` 中的每条验收条件都有对应的测试函数

### 原则 P4-3：真实环境测试

- 验收测试启动真实的 backend HTTP server（`httptest.NewServer` 或随机端口 `Server.Start`）
- 使用真实的 SQLite（`:memory:`），不用 mock
- 模拟真实的 HTTP 请求/响应，不用函数调用

### 原则 P4-4：并发测试必须用 -race

- 所有并发验收测试必须通过 `go test -race`
- 不允许出现 DATA RACE 警告
- 并发测试的 goroutine 数量不少于 10（验证 SQLite 的 busy timeout 机制）

### 原则 P4-5：降级测试验证"不崩溃"

- 降级测试的核心断言是"不 panic"而非"返回正确结果"
- 使用 `recover()` 或 `assert.NotPanics` 风格的检查
- 降级测试的失败场景必须是可预期的（连接拒绝、超时、非 JSON）

---

## 三、约束条件

### 约束 C4-1：不修改业务代码

- 本 Phase **只新增测试文件**，不修改 `store/`、`service/`、`domain/`、`api/` 中的任何业务代码
- 唯一例外：如果验收测试发现 bug，先记录到 bug 清单，不在本 Phase 修复
- 验收测试的目的是 **验证系统行为**，不是修复问题

### 约束 C4-2：不引入外部测试库

- 继续使用 Go 标准库 `testing`
- HTTP 测试用 `net/http/httptest`
- 断言用 `testutil` 中的辅助函数
- 不使用 testify、resty 等第三方 HTTP 客户端

### 约束 C4-3：验收测试独立于 BDD 测试

- `tests/acceptance/` 与 `tests/backend/`（BDD 场景）是独立的测试套件
- 验收测试关注端到端链路（多个 API 调用的组合）
- BDD 测试关注单个行为的 Given-When-Then
- 两者可共存，不冲突

### 约束 C4-4：并发测试不依赖执行顺序

- 每个并发测试独立初始化数据
- 不依赖上一个测试创建的状态
- 测试可随机排序执行（`go test -shuffle=on`）

### 约束 C4-5：降级测试不依赖外部服务

- 不依赖真实的 `.claude/` shell 脚本运行
- 用 Go 代码模拟 shell 侧的 HTTP 调用行为
- 不启动真实的 Claude Code CLI

### 约束 C4-6：测试资源必须清理

- 每个 `SetupAcceptanceTest` 注册 `t.Cleanup` 关闭 server 和 DB
- 临时 SQLite 文件用 `t.TempDir()` 自动清理
- 不遗留端口占用（server 关闭后端口释放）

---

## 四、产出物

| 产出物 | 路径 | 操作 |
|--------|------|------|
| 验收测试基础设施 | `tests/acceptance/helper.go` | 新增 |
| Issue 完整链路测试 | `tests/acceptance/issue_e2e_test.go` | 新增 |
| Session 生命周期测试 | `tests/acceptance/session_e2e_test.go` | 新增 |
| 多 session 交叉测试 | `tests/acceptance/cross_session_test.go` | 新增 |
| 并发安全测试 | `tests/acceptance/concurrency_e2e_test.go` | 新增 |
| 降级验收测试 | `tests/acceptance/degradation_e2e_test.go` | 新增 |
| API 验收条件文档 | `tests/acceptance/ACCEPTANCE.md` | 新增 |

---

## 五、验收检查点

```bash
# 检查点 1：端到端 Issue 生命周期
go test ./tests/acceptance/... -run "TestIssueCompleteLifecycle" -v
# 期望：9 个验收条件全部 PASS

# 检查点 2：端到端 Session 生命周期
go test ./tests/acceptance/... -run "TestSessionLifecycle" -v
# 期望：5 个验收条件全部 PASS

# 检查点 3：并发安全
go test ./tests/acceptance/... -run "TestConcurrent" -race -v
# 期望：无 DATA RACE，恰好 1 个成功

# 检查点 4：降级验收
go test ./tests/acceptance/... -run "TestBackendDegradation" -v
# 期望：全部 PASS，无 panic

# 检查点 5：全部验收测试
go test ./tests/acceptance/... -v -count=1
# 期望：0 FAIL

# 检查点 6：无回归（全量测试）
go test ./... -count=1
# 期望：0 FAIL（包含 Phase 0-3 的所有测试）

# 检查点 7：验收文档
cat tests/acceptance/ACCEPTANCE.md | wc -l
# 期望：> 50 行（覆盖全部 API 端点的验收条件）
```
