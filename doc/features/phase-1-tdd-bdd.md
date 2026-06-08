# Phase 1：TDD + BDD — 测试保护网

> **前置依赖**：Phase 0 完成
> **预估工时**：5-8 天
> **目标**：为 domain/store/service 三层建立单元测试，用 BDD 场景覆盖核心业务流，建立后续重构的安全网

---

## 一、任务清单

### 阶段 1a：Store 层单元测试（2-3 天）

#### 任务 1a.1：IssueStore 测试

| 序号 | 测试函数 | 场景 | 文件 |
|------|---------|------|------|
| 1a.1.1 | `TestClaimIssue` | 首次领取成功：idle → claimed，返回 `Success=true` | `internal/backend/store/issue_store_test.go` |
| 1a.1.2 | — | 已被他人领取：session-A claim 后 session-B claim，返回 `Success=false, ClaimedBy=session-A` | 同上 |
| 1a.1.3 | — | 同一 session 重复领取（幂等）：返回 `Success=true` | 同上 |
| 1a.1.4 | `TestUpdateIssueStatus` | 正向流转：claimed → fixing，返回 `PreviousStatus=claimed` | 同上 |
| 1a.1.5 | — | 非持有者更新：session-B 更新 session-A 持有的 issue，`Updated=false` | 同上 |
| 1a.1.6 | — | 无效状态值：传入不存在的 status，`Updated=false` | 同上 |
| 1a.1.7 | `TestReleaseIssue` | 非终态释放：fixing 状态 → idle | 同上 |
| 1a.1.8 | — | 终态不释放：merged 状态，返回 `false` | 同上 |
| 1a.1.9 | — | 非持有者释放：返回 `false` | 同上 |
| 1a.1.10 | `TestReleaseSessionIssues` | 释放 session 持有的 3 个 issue，返回 `[1, 2, 3]` | 同上 |
| 1a.1.11 | — | session 无 claim 时，返回空列表 `[]` | 同上 |
| 1a.1.12 | — | 终态 issue 不被释放（session 持有 3 个，其中 1 个 merged，只释放 2 个） | 同上 |
| 1a.1.13 | `TestCheckIssues` | 批量检查 3 个 issue 的状态 | 同上 |
| 1a.1.14 | — | 检查不存在的 issue（返回 idle） | 同上 |
| 1a.1.15 | `TestListIssues` | 默认分页：返回 20 条 | 同上 |
| 1a.1.16 | — | 按状态过滤 | 同上 |
| 1a.1.17 | — | 按 repo 过滤 | 同上 |
| 1a.1.18 | — | 按 session 过滤 | 同上 |

**测试模式**：每个方法一个 `TestXxx` 函数，内部用 `t.Run("场景名", ...)` 组织子测试。

```go
func TestClaimIssue(t *testing.T) {
    db := testutil.OpenTestDB(t)
    ctx := context.Background()
    issues := db.Issues()

    t.Run("首次领取成功", func(t *testing.T) {
        result, err := issues.ClaimIssue(ctx, "owner/repo", 1, "sess-A", "title")
        require.NoError(t, err)
        assert.True(t, result.Success)
        assert.Equal(t, "claimed", result.Status)
    })

    t.Run("已被他人领取", func(t *testing.T) {
        // 预置：sess-A 已领取
        issues.ClaimIssue(ctx, "owner/repo", 2, "sess-A", "title")
        result, err := issues.ClaimIssue(ctx, "owner/repo", 2, "sess-B", "title")
        require.NoError(t, err)
        assert.False(t, result.Success)
        assert.NotNil(t, result.ClaimedBy)
        assert.Equal(t, "sess-A", *result.ClaimedBy)
    })
}
```

---

#### 任务 1a.2：SessionStore 测试

| 序号 | 测试函数 | 场景 | 文件 |
|------|---------|------|------|
| 1a.2.1 | `TestRegisterSession` | 首次注册成功，状态为 active | `internal/backend/store/session_store_test.go` |
| 1a.2.2 | — | 重复注册返回 `ErrSessionExists` | 同上 |
| 1a.2.3 | `TestCloseSession` | 关闭活跃 session，状态变 closed，ClosedAt 非 nil | 同上 |
| 1a.2.4 | — | 关闭已关闭 session 返回 `ErrSessionNotFound` | 同上 |
| 1a.2.5 | `TestGetSession` | 获取存在的 session | 同上 |
| 1a.2.6 | — | 获取不存在的 session 返回 nil, nil | 同上 |
| 1a.2.7 | `TestListSessions` | 无过滤返回全部 | 同上 |
| 1a.2.8 | — | 按状态过滤 | 同上 |
| 1a.2.9 | `TestCleanupTimedOut` | 清理超时 session | 同上 |

---

#### 任务 1a.3：其他 Store 测试

| 序号 | 测试内容 | 文件 |
|------|---------|------|
| 1a.3.1 | MachineStore：ListMachines、GetMachine | `internal/backend/store/machine_store_test.go` |
| 1a.3.2 | ProjectStore：ListProjects、GetProject | `internal/backend/store/project_store_test.go` |
| 1a.3.3 | ConfigStore：GetConfig、UpdateConfig | `internal/backend/store/config_store_test.go` |
| 1a.3.4 | SQLite 初始化：migration 正确执行、WAL 模式设置 | `internal/backend/store/sqlite_test.go` |

---

### 阶段 1b：Service 层单元测试（2-3 天）

#### 任务 1b.1：IssueService 测试

**核心价值**：验证 service 层的日志记录和错误传播，而非 SQL 逻辑（store 层已覆盖）。

| 序号 | 测试函数 | 场景 | 文件 |
|------|---------|------|------|
| 1b.1.1 | `TestIssueService_Check` | mock 返回结果，验证透传 | `internal/backend/service/issue_service_test.go` |
| 1b.1.2 | `TestIssueService_Claim` | mock 成功 + mock 错误 | 同上 |
| 1b.1.3 | `TestIssueService_UpdateStatus` | mock 成功 + mock 错误 | 同上 |
| 1b.1.4 | `TestIssueService_Release` | mock 返回 true / false | 同上 |
| 1b.1.5 | `TestIssueService_ReleaseSession` | mock 返回列表 | 同上 |
| 1b.1.6 | `TestIssueService_List` | 过滤参数正确传递 | 同上 |

```go
func TestIssueService_Claim(t *testing.T) {
    t.Run("store返回成功", func(t *testing.T) {
        mock := &testutil.MockIssueStore{
            ClaimIssueFunc: func(ctx context.Context, repo string, number int, sessionID string, title string) (*store.ClaimResult, error) {
                return &store.ClaimResult{Success: true, Status: "claimed"}, nil
            },
        }
        svc := service.NewIssueService(mock)
        result, err := svc.Claim(context.Background(), "owner/repo", 1, "sess-A", "title")
        require.NoError(t, err)
        assert.True(t, result.Success)
        assert.Equal(t, 1, mock.Calls["ClaimIssue"])
    })

    t.Run("store返回错误", func(t *testing.T) {
        mock := &testutil.MockIssueStore{
            ClaimIssueFunc: func(ctx context.Context, repo string, number int, sessionID string, title string) (*store.ClaimResult, error) {
                return nil, errors.New("db error")
            },
        }
        svc := service.NewIssueService(mock)
        _, err := svc.Claim(context.Background(), "owner/repo", 1, "sess-A", "title")
        assert.Error(t, err)
    })
}
```

---

#### 任务 1b.2：其他 Service 测试

| 序号 | 测试内容 | 文件 |
|------|---------|------|
| 1b.2.1 | SessionService：Register/Close/List 用 mock 验证 | `internal/backend/service/session_service_test.go` |
| 1b.2.2 | StatusService：mock 5 个 store，验证聚合逻辑正确 | `internal/backend/service/status_service_test.go` |
| 1b.2.3 | CleanupService：验证超时判定 + 批量清理 | `internal/backend/service/cleanup_service_test.go` |
| 1b.2.4 | IdleWatchdog：验证定时触发 + context 取消停止 | `internal/backend/service/idle_watchdog_test.go` |
| 1b.2.5 | MachineService/ProjectService/ProxyService：简单透传测试 | 各 `*_service_test.go` |

---

### 阶段 1c：BDD 场景测试（1-2 天）

#### 任务 1c.1：Issue 生命周期 BDD 场景

| 序号 | 场景编号 | Given-When-Then | 文件 |
|------|---------|-----------------|------|
| 1c.1.1 | 1.1 | Given idle Issue / When session-A claim / Then success + claimed | `tests/backend/issue_lifecycle_test.go` |
| 1c.1.2 | 1.2 | Given session-A claimed / When session-B claim / Then fail + ClaimedBy | 同上 |
| 1c.1.3 | 1.3 | Given session-A claimed / When session-A claim again / Then success（幂等） | 同上 |
| 1c.1.4 | 1.4 | Given claimed / When update to fixing / Then previous=claimed | 同上 |
| 1c.1.5 | 1.5 | Given session-A owned / When session-B update / Then fail | 同上 |
| 1c.1.6 | 1.6 | Given merged / When release / Then false | 同上 |
| 1c.1.7 | 1.7 | Given 3 issues / When session close / Then all idle | 同上 |
| 1c.1.8 | 1.8 | Given mixed status / When filter by status / Then only matched | 同上 |

#### 任务 1c.2：Session 生命周期 BDD 场景

| 序号 | 场景编号 | Given-When-Then | 文件 |
|------|---------|-----------------|------|
| 1c.2.1 | 2.1 | Given 无同名 / When register / Then active | `tests/backend/session_lifecycle_test.go` |
| 1c.2.2 | 2.2 | Given 已注册 / When 再次注册 / Then ErrSessionExists | 同上 |
| 1c.2.3 | 2.3 | Given active / When close / Then closed + ClosedAt | 同上 |
| 1c.2.4 | 2.4 | Given closed / When 再次 close / Then ErrSessionNotFound | 同上 |
| 1c.2.5 | 2.5 | Given 超时 active / When cleanup / Then closed + timeout | 同上 |

#### 任务 1c.3：Proxy 拦截 BDD 场景

| 序号 | 场景编号 | Given-When-Then | 文件 |
|------|---------|-----------------|------|
| 1c.3.1 | 3.1 | Given 上游可用 / When 非流式请求 / Then 转发成功 + trace | `tests/proxy/intercept_lifecycle_test.go` |
| 1c.3.2 | 3.2 | Given SSE 上游 / When 流式请求 / Then 逐事件转发 + trace | 同上 |
| 1c.3.3 | 3.3 | Given 上游不可用 + 兜底配置 / When 请求 / Then 切换兜底 | 同上 |
| 1c.3.4 | 3.4 | Given 兜底模式 / When 上游恢复 / Then 切回 | 同上 |
| 1c.3.5 | 3.5 | Given 非白名单路径 / When 请求 / Then 拒绝 | 同上 |

---

### 阶段 1d：Proxy 核心路径测试（1-2 天）

#### 任务 1d.1：扩展已有 Proxy 测试

已有 4 个测试文件，扩展覆盖率：

| 序号 | 测试内容 | 文件 |
|------|---------|------|
| 1d.1.1 | `TestFallbackDecision` — 更多边界条件 | `internal/proxy/fallback_test.go` 扩展 |
| 1d.1.2 | `TestModelRewrite` — 空 model / 相同 model | `internal/proxy/model_rewrite_test.go` 扩展 |
| 1d.1.3 | `TestThinkingInject` — 旧格式兼容 | `internal/proxy/thinking_inject_test.go` 扩展 |
| 1d.1.4 | `TestReasoningCache` — 并发读写 | `internal/proxy/reasoning_cache_test.go` 扩展 |

#### 任务 1d.2：SSE Reassembler 单元测试

| 序号 | 测试内容 | 文件 |
|------|---------|------|
| 1d.2.1 | Anthropic SSE 解析 | `internal/sse/reassembler_test.go` 扩展 |
| 1d.2.2 | OpenAI SSE 解析（修复已有失败测试） | 同上 |
| 1d.2.3 | Gemini SSE 解析 | 同上 |
| 1d.2.4 | 不完整 chunk 边界处理 | 同上 |

---

## 二、修改原则

### 原则 P1-1：先测后改

- 每个测试文件 **先写失败测试，确认编译通过**，再考虑后续 Phase 的修改
- 测试是 Phase 1 的主要产出，不是附属品
- 如果某个方法无法测试（如私有方法），记录到 Phase 2 待重构清单

### 原则 P1-2：中文场景名

- 所有 `t.Run()` 的场景名使用中文
- 格式：`"场景X.Y: 简短描述"`
- 目的：`go test -v` 输出可直接作为行为文档

```go
// ✓ 正确
t.Run("场景1.2: 领取已被占用的Issue", func(t *testing.T) { ... })

// ✗ 错误
t.Run("TestClaimConflict", func(t *testing.T) { ... })
```

### 原则 P1-3：测试独立

- 每个子测试（`t.Run`）独立创建数据，不依赖前一个子测试的状态
- 使用不同的 issue number 或 session ID 避免冲突
- 不使用 `t.Parallel()`（SQLite 不支持并发写入）

### 原则 P1-4：三层测试隔离

| 层 | 测试目标 | 依赖 |
|----|---------|------|
| Store 层 | SQL 逻辑正确性 | 内存 SQLite（`testutil.OpenTestDB`） |
| Service 层 | 业务编排逻辑 | Mock Store（`testutil.MockXxxStore`） |
| BDD 层 | 行为场景完整性 | 内存 SQLite（真实 DB，非 mock） |

### 原则 P1-5：断言风格

- 不使用 `testify/assert`，用 Go 标准库
- 用 `t.Fatalf` 标记致命错误（中断当前子测试）
- 用 `t.Errorf` 标记非致命错误（继续执行，收集更多错误）
- 辅助函数：可复用 Phase 0 创建的 `testutil` 工具

```go
// 自定义辅助
func requireNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil { t.Fatalf("unexpected error: %v", err) }
}
func assertEqual(t *testing.T, got, want interface{}, msg ...string) {
    t.Helper()
    if got != want { t.Errorf("got %v, want %v: %s", got, want, strings.Join(msg, " ")) }
}
```

---

## 三、约束条件

### 约束 C1-1：禁止修改业务代码

- `store/*.go`（非 test）、`service/*.go`（非 test）、`api/*.go`、`domain/*.go` **全部不可修改**
- 唯一例外：如果为了测试需要导出某个内部函数（如 `runMigrations`），可在 `store/` 中新增一个 `export_test.go` 文件
- `export_test.go` 是 Go 惯例：只在测试时导出内部符号

### 约束 C1-2：禁止引入测试框架

- 不使用 testify、ginkgo、gomega、gomock
- 理由：项目当前零外部依赖，保持这一特性
- 如果标准库断言不够用，在 `testutil/assert.go` 中新增辅助函数

### 约束 C1-3：禁止修改已有测试

- `tests/backend/` 下已有的 26 个 API 测试文件 **不可修改**
- 新增的 BDD 场景测试放在新文件中（`issue_lifecycle_test.go`、`session_lifecycle_test.go`）
- 已有的 proxy 单元测试（`fallback_test.go` 等）**可以扩展**（新增 `t.Run`），但不可修改已有 `t.Run` 的断言

### 约束 C1-4：覆盖率目标

| 包 | 最低覆盖率 | 不达标时 |
|----|-----------|---------|
| `internal/backend/store/` | > 80% | 不阻塞，记录未覆盖场景到清单 |
| `internal/backend/service/` | > 70% | 同上 |
| `internal/proxy/` | > 60% | 同上 |
| `internal/sse/` | > 60% | 同上 |

覆盖率是 **参考指标，不是硬门槛**。质量优先于数字。

### 约束 C1-5：测试不可有外部依赖

- 不依赖网络（HTTP 测试用 `httptest.NewServer`）
- 不依赖文件系统路径（用 `t.TempDir()`）
- 不依赖时间（需要时可注入 `time.Now` 的替代）
- 不依赖环境变量

---

## 四、产出物

| 产出物 | 路径 | 数量 |
|--------|------|------|
| Store 层单元测试 | `internal/backend/store/*_test.go` | 5 个文件 |
| Service 层单元测试 | `internal/backend/service/*_test.go` | 5-6 个文件 |
| Issue 生命周期 BDD | `tests/backend/issue_lifecycle_test.go` | 1 个文件，8 个场景 |
| Session 生命周期 BDD | `tests/backend/session_lifecycle_test.go` | 1 个文件，5 个场景 |
| Proxy 拦截 BDD | `tests/proxy/intercept_lifecycle_test.go` | 1 个文件，5 个场景 |
| Proxy 单元测试扩展 | `internal/proxy/*_test.go` | 扩展 4 个文件 |
| SSE 单元测试扩展 | `internal/sse/reassembler_test.go` | 扩展 1 个文件 |
| 断言辅助函数（如需） | `internal/testutil/assert.go` | 0-1 个文件 |

---

## 五、验收检查点

```bash
# 检查点 1：Store 层覆盖率
go test ./internal/backend/store/... -cover -coverprofile=store.out
go tool cover -func=store.out | tail -1
# 期望：> 80%

# 检查点 2：Service 层覆盖率
go test ./internal/backend/service/... -cover -coverprofile=service.out
go tool cover -func=service.out | tail -1
# 期望：> 70%

# 检查点 3：BDD 场景全通过
go test ./tests/backend/... -run "场景" -v
# 期望：13 个场景（8 Issue + 5 Session）全部 PASS

# 检查点 4：Proxy 测试
go test ./internal/proxy/... -cover
# 期望：> 60%

# 检查点 5：无回归
go test ./... 2>&1 | grep -i fail
# 期望：无 FAIL 输出

# 检查点 6：BDD 输出可读性
go test ./tests/backend/... -run "TestIssueLifecycle" -v | head -30
# 期望：输出包含中文场景名，可读作行为规格文档
```
