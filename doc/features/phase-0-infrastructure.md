# Phase 0：基础设施

> **前置依赖**：无
> **预估工时**：1-2 天
> **目标**：修复失败的测试、建立测试工具链，为后续 Phase 提供基础设施

---

## 一、任务清单

### 任务 0.1：修复 e2e 测试 fixture

**背景**：`tests/e2e/` 目录下 6 个测试因缺少 fixture 文件全部 FAIL。`testutil.LoadFixture()` 从 `testdata/fixtures/` 目录加载 JSON，但该目录不存在。

**操作步骤**：

| 序号 | 操作 | 文件 |
|------|------|------|
| 0.1.1 | 创建 `tests/e2e/testdata/fixtures/` 目录 | `tests/e2e/testdata/fixtures/` |
| 0.1.2 | 创建 `nonstreaming_anthropic.json` fixture | `tests/e2e/testdata/fixtures/nonstreaming_anthropic.json` |
| 0.1.3 | 创建 `streaming_anthropic.json` fixture | `tests/e2e/testdata/fixtures/streaming_anthropic.json` |
| 0.1.4 | 创建 `gzip_response.json` fixture | `tests/e2e/testdata/fixtures/gzip_response.json` |
| 0.1.5 | 验证 6 个 e2e 测试全部 PASS | `go test ./tests/e2e/... -v` |

**Fixture 格式要求**（对齐 `testutil.Fixture` struct）：

```json
{
  "name": "nonstreaming_anthropic",
  "description": "Anthropic 非流式 messages 响应",
  "request": {
    "method": "POST",
    "path": "/v1/messages",
    "headers": {
      "Content-Type": "application/json",
      "X-Api-Key": "sk-ant-test-key-12345678",
      "Authorization": "Bearer sk-ant-test-key-12345678"
    },
    "body": { ... }
  },
  "response": {
    "status": 200,
    "headers": { "Content-Type": "application/json" },
    "body": { "type": "message", "id": "msg_test", ... },
    "sse_events": null
  },
  "expected_trace": {
    "request_method": "POST",
    "request_path": "/v1/messages",
    "response_status": 200,
    "contains_fields": ["session_id", "turn", "request", "response"]
  }
}
```

**验收**：
```bash
go test ./tests/e2e/... -v
# 6 个测试全部 PASS
```

---

### 任务 0.2：创建内存 SQLite 测试 DB 工具

**背景**：Store 层测试需要一个可快速创建/销毁的 SQLite 实例。当前 `store.NewSQLiteStore()` 只支持文件路径，不支持 `:memory:`。

**操作步骤**：

| 序号 | 操作 | 文件 |
|------|------|------|
| 0.2.1 | 新增 `OpenTestDB(t *testing.T) store.Store` 函数 | `internal/testutil/testdb.go` |
| 0.2.2 | 函数内部：用 `sql.Open("sqlite", ":memory:?_pragma=busy_timeout(5000)")` 打开内存 DB | — |
| 0.2.3 | 执行 `runMigrations(db)` 初始化表结构 | — |
| 0.2.4 | 构造 `store.SQLiteStore` 并返回 `store.Store` 接口 | — |
| 0.2.5 | 注册 `t.Cleanup(func())` 确保连接关闭 | — |

**需要处理的可见性问题**：

`runMigrations()` 和 `SQLiteStore` 的内部字段（`db`）可能在 `store` 包外不可访问。解决方案：

- 方案 A（推荐）：在 `store` 包内导出一个 `NewInMemoryStore() (*SQLiteStore, error)` 函数，testutil 调用它
- 方案 B：testutil 直接调用 `store.NewSQLiteStore(":memory:")`（如果 SQLite 驱动支持）

**产出文件**：
```
internal/testutil/testdb.go     ← 新增
internal/backend/store/sqlite.go ← 新增 NewInMemoryStore()（如需方案 A）
```

**验收**：
```go
func TestOpenTestDB(t *testing.T) {
    db := testutil.OpenTestDB(t)
    // 能正常调用 db.Issues().ClaimIssue(...) 不报错
    // 能正常调用 db.Sessions().RegisterSession(...) 不报错
}
```

---

### 任务 0.3：创建 Store 接口的 Mock 实现

**背景**：Service 层测试需要 mock Store 接口。项目无外部依赖，手写 mock。

**操作步骤**：

| 序号 | 操作 | 文件 |
|------|------|------|
| 0.3.1 | 创建 `MockIssueStore` | `internal/testutil/mock_issue_store.go` |
| 0.3.2 | 创建 `MockSessionStore` | `internal/testutil/mock_session_store.go` |
| 0.3.3 | 创建 `MockMachineStore` | `internal/testutil/mock_machine_store.go` |
| 0.3.4 | 创建 `MockProjectStore` | `internal/testutil/mock_project_store.go` |
| 0.3.5 | 创建 `MockConfigStore` | `internal/testutil/mock_config_store.go` |
| 0.3.6 | 创建 `MockProxyStore` | `internal/testutil/mock_proxy_store.go` |

**Mock 模式**（每个 Mock 遵循相同模式）：

```go
type MockIssueStore struct {
    // 每个 interface 方法对应一个 Func 字段
    CheckIssuesFunc         func(ctx context.Context, repo string, numbers []int) ([]store.IssueCheckResult, error)
    ClaimIssueFunc          func(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*store.ClaimResult, error)
    UpdateIssueStatusFunc   func(ctx context.Context, repo string, number int, sessionID string, newStatus string) (*store.UpdateStatusResult, error)
    ReleaseIssueFunc        func(ctx context.Context, repo string, number int, sessionID string) (bool, error)
    ReleaseSessionIssuesFunc func(ctx context.Context, sessionID string) ([]int, error)
    ListIssuesFunc          func(ctx context.Context, filter store.IssueFilter) ([]store.IssueListItem, int, error)
    ListIssuesBySessionFunc func(ctx context.Context, sessionID string) ([]store.IssueListItem, error)

    // 调用追踪
    Calls map[string]int
}

// 每个方法实现：记录调用 + 调用 Func
func (m *MockIssueStore) ClaimIssue(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*store.ClaimResult, error) {
    m.Calls["ClaimIssue"]++
    if m.ClaimIssueFunc != nil {
        return m.ClaimIssueFunc(ctx, repo, number, sessionID, issueTitle)
    }
    return nil, nil
}
// ... 其余方法同理
```

**验收**：
```go
func TestMockIssueStore(t *testing.T) {
    mock := &MockIssueStore{
        ClaimIssueFunc: func(ctx context.Context, repo string, number int, sessionID string, issueTitle string) (*store.ClaimResult, error) {
            return &store.ClaimResult{Success: true, Status: "claimed"}, nil
        },
    }
    result, err := mock.ClaimIssue(ctx, "owner/repo", 1, "sess-1", "title")
    // result.Success == true, err == nil, mock.Calls["ClaimIssue"] == 1
}
```

---

### 任务 0.4：修复 SSE 测试失败

**背景**：`tests/sse/reassembler_test.go` 的 `TestChatCompletionsUsageNormalization` 失败，`expected usage to be map[string]int64, got <nil>`。

**操作步骤**：

| 序号 | 操作 | 文件 |
|------|------|------|
| 0.4.1 | 定位 `reassembler_test.go:105` 的断言逻辑 | `tests/sse/reassembler_test.go` |
| 0.4.2 | 排查 SSE reassembler 对 OpenAI 格式 usage 的解析逻辑 | `internal/sse/reassembler.go` |
| 0.4.3 | 修复解析逻辑或调整测试预期 | — |
| 0.4.4 | 验证 `go test ./tests/sse/... -v` PASS | — |

**验收**：
```bash
go test ./tests/sse/... -v
# TestChatCompletionsUsageNormalization PASS
```

---

## 二、修改原则

### 原则 P0-1：最小侵入

- 只新增文件，不修改已有业务逻辑代码
- fixture 是纯数据文件，不影响任何运行时代码
- mock 和 testdb 只在 `internal/testutil/` 内，不修改 `store/` 的已有接口

### 原则 P0-2：零外部依赖

- 不引入 testify、mockgen、gomock 等测试库
- 使用 Go 标准库 `testing` 包
- mock 手写，保持项目依赖链只有 `modernc.org/sqlite`

### 原则 P0-3：接口忠实

- Mock 实现必须 **精确匹配** `store/` 中的 interface 定义
- 方法签名（参数类型、返回类型）必须一致
- 如果 store 接口未来变化，mock 必须同步更新

### 原则 P0-4：自清理

- 所有测试资源（DB 连接、临时文件、goroutine）通过 `t.Cleanup()` 注册清理
- 不依赖测试执行顺序
- 并行测试时互不干扰

---

## 三、约束条件

### 约束 C0-1：禁止修改 Store 接口

- `store/store.go` 中定义的 `IssueStore`、`SessionStore` 等接口 **不可修改**
- 如果发现接口设计不合理，记录到 Phase 2（DDD）的需求清单，不在本阶段处理
- 只允许在 `store/sqlite.go` 中 **新增** `NewInMemoryStore()` 一个导出函数

### 约束 C0-2：禁止修改已有测试的断言逻辑

- `tests/backend/` 下 26 个已有集成测试文件的断言不可修改
- 如果 fixture 值需要调整以匹配已有断言，调整 fixture 而非测试
- 唯一例外：`tests/sse/reassembler_test.go` 的 `TestChatCompletionsUsageNormalization` 可修复

### 约束 C0-3：Fixture 必须可手工验证

- 每个 fixture JSON 必须是人类可读的，不能是 hex dump 或 base64
- Fixture 的 `expected_trace` 必须与 proxy 的实际 trace 输出格式对齐
- Fixture 文件不超过 200 行

### 约束 C0-4：Testutil 不依赖 tests/

- `internal/testutil/` 是公共测试工具，不能 import `tests/` 下的任何包
- `tests/` 下的测试文件可以 import `internal/testutil/`

---

## 四、产出物

| 产出物 | 路径 | 状态 |
|--------|------|------|
| e2e fixture 文件 ×3 | `tests/e2e/testdata/fixtures/*.json` | 新增 |
| 内存 DB 工具 | `internal/testutil/testdb.go` | 新增 |
| Mock IssueStore | `internal/testutil/mock_issue_store.go` | 新增 |
| Mock SessionStore | `internal/testutil/mock_session_store.go` | 新增 |
| Mock MachineStore | `internal/testutil/mock_machine_store.go` | 新增 |
| Mock ProjectStore | `internal/testutil/mock_project_store.go` | 新增 |
| Mock ConfigStore | `internal/testutil/mock_config_store.go` | 新增 |
| Mock ProxyStore | `internal/testutil/mock_proxy_store.go` | 新增 |
| （可能）Store 新增导出函数 | `internal/backend/store/sqlite.go` | 修改 |

---

## 五、验收检查点

```bash
# 检查点 1：全部测试通过
cd claude_tap_plus
go test ./...
# 期望：0 FAIL，所有包 PASS 或 [no test files]

# 检查点 2：e2e 测试覆盖
go test ./tests/e2e/... -v -count=1
# 期望：7 个测试全部 PASS（含已有的 TestBlockedPath、TestTraceInitEndpoint 等）

# 检查点 3：SSE 测试
go test ./tests/sse/... -v -count=1
# 期望：全部 PASS，无 nil usage

# 检查点 4：testutil 编译
go build ./internal/testutil/
# 期望：编译通过，无错误

# 检查点 5：已有测试无回归
go test ./tests/backend/... -count=1
# 期望：与 Phase 0 之前结果一致，全部 PASS
```
