# claude-tap-plus 共享参考：API 清单与数据模型

> 拆分自：`2026-06-01-可视化页面线框图设计-v2.md`
> 原始日期：2026-06-01 | 校验日期：2026-06-06（对照实际代码验证）
> 严格基于后端已有 API 和数据库表结构设计，不假设任何未实现的接口。

---

## 一、已有 API 清单（严格限定）

| 端点 | 方法 | 功能 | 请求/响应 |
|------|------|------|-----------|
| `/health` | GET | 健康检查 | `{"status":"ok"}` |
| `/api/issue/check` | POST | 检查 Issue 状态 | `CheckIssuesRequest` / `CheckIssuesResponse` |
| `/api/issue/claim` | POST | 领取 Issue | `ClaimIssueRequest` / `ClaimIssueResponse` |
| `/api/issue/release` | POST | 释放 Issue | `ReleaseIssueRequest` / `ReleaseIssueResponse` |
| `/api/issue/release-session` | POST | 释放会话下所有 Issue | `ReleaseSessionRequest` / `ReleaseSessionResponse` |
| `/api/issue/status` | POST | 更新 Issue 状态 | `UpdateStatusRequest` / `UpdateStatusResponse` |
| `/api/issues` | GET | Issue 列表（过滤+分页） | Query: `repo`, `status`, `session_id`, `page`, `page_size` / `IssuesListResponse` |
| `/api/session/register` | POST | 注册会话 | `RegisterSessionRequest` / `{"status":"registered"}` |
| `/api/session/close` | POST | 关闭会话 | `CloseSessionRequest` / `{"status":"closed"}` |
| `/api/sessions` | GET | 获取会话列表（支持过滤） | Query: `machine_id`, `project_slug`, `status` / `SessionListResponse` |
| `/api/session/{id}` | GET | 获取单个会话详情 | Path: session_id / `SessionDetail` |
| `/api/session/{id}/issues` | GET | 获取会话关联 Issue | Path: session_id / `SessionIssuesResponse` |
| `/api/session/{id}/tokens` | GET | 获取会话 Token 统计 | Path: session_id / `SessionTokensResponse` |
| `/api/session/{id}/traces` | GET | 获取会话 Trace 文件 | Path: session_id / `SessionTracesResponse` |
| `/api/machines` | GET | 获取机器列表 | Query: `os`, `hostname` / `MachinesListResponse` |
| `/api/projects` | GET | 获取项目列表 | — / `ProjectsListResponse` |
| `/api/proxy/register` | POST | 代理注册 | `{"pid":"","proxy_url":""}` |
| `/api/proxy/unregister` | POST | 代理注销 | `{"pid":""}` |
| `/api/proxy/trace-init` | POST | 转发 trace-init | 请求体原样转发 |
| `/api/proxies` | GET | 列出代理注册列表 | Query: `status`, `project` / `ProxiesResponse` |
| `/api/status` | GET | 系统聚合状态 | `StatusResponse`（含 `SystemStats`） |
| `/api/logs` | GET | 查询后端日志 | Query: `level`, `date`, `limit` / `LogsResponse` |
| `/api/config` | GET | 获取系统配置 | — / `ConfigResponse` |
| `/api/config` | PUT | 更新系统配置 | `map[string]interface{}` / `ConfigResponse` |

---

## 二、已有数据模型

### 2.1 IssueClaim（issue_claims 表）

```go
type IssueClaim struct {
    ID           int64       // 自增主键
    RepoFullName string      // 仓库全名
    IssueNumber  int         // GitHub issue 编号
    IssueTitle   string      // issue 标题（缓存）
    Status       IssueStatus // idle/claimed/fixing/ready-for-pr/pr-created/testing/reviewing/merged/rejected
    SessionID    string      // 领取者 session_id，idle 时为空
    ClaimedAt    *time.Time  // 领取时间
    UpdatedAt    time.Time   // 最后更新时间
}
```

### 2.2 Session（sessions 表）

```go
type Session struct {
    ID             int64      // 自增主键
    SessionID      string     // UUID
    MachineID      string     // whoami@hostname
    OS             string     // windows/linux/macos
    ProjectSlug    string     // 项目标识
    ProjectCwd     string     // 工作目录
    TranscriptPath string     // 对话记录路径
    LocalTracePath string     // 本地 trace 文件路径
    Model          string     // 使用的模型
    Source         string     // startup/resume
    Status         string     // active/closed
    RegisteredAt   time.Time  // 注册时间
    ClosedAt       *time.Time // 关闭时间
    CloseReason    string     // 关闭原因
}
```

### 2.3 Machine（machines 表）

```go
type Machine struct {
    ID          int64     // 自增主键
    MachineID   string    // whoami@hostname，UNIQUE
    OS          string    // windows/linux/macos
    Hostname    string    // 主机名
    Username    string    // 用户名
    FirstSeenAt time.Time // 首次注册时间
    LastSeenAt  time.Time // 最后活跃时间
}
```

### 2.4 Project（projects 表）

```go
type Project struct {
    ID          int64     // 自增主键
    ProjectSlug string    // 项目标识，UNIQUE
    ProjectCwd  string    // 工作目录
    FirstSeenAt time.Time // 首次出现时间
    LastSeenAt  time.Time // 最后活跃时间
}
```

### 2.5 Proxy（proxies 表）

> **注意**：Proxy 在代码中没有独立的 `domain` struct，仅存在于数据库表 `proxies` 和 API 响应 `ProxyItem` 中。

```go
// proxies 表结构（store/migrations.go）
// proxy_id       TEXT    PRIMARY KEY
// project_slug   TEXT    NOT NULL
// status         TEXT    NOT NULL DEFAULT 'active'
// registered_at  DATETIME NOT NULL
// last_ping_at   DATETIME

// API 响应结构（api/response.go → ProxyItem）
type ProxyItem struct {
    ProxyID      string     `json:"proxy_id"`
    ProjectSlug  string     `json:"project_slug"`
    Status       string     `json:"status"`       // active / offline
    RegisteredAt time.Time  `json:"registered_at"`
    LastPingAt   *time.Time `json:"last_ping_at"`
}
```

### 2.6 TokenStats（无数据库表，运行时计算）

> 来源：`domain/token.go`，由 `GET /api/session/{id}/tokens` 解析 trace 文件动态计算。

```go
type TokenStats struct {
    APICalls    int // API 调用次数
    InputTokens int // 输入 Token 数
    OutputTokens int // 输出 Token 数
    CacheRead   int // 缓存读取 Token 数
    CacheCreate int // 缓存创建 Token 数
}

func (t *TokenStats) Total() int {
    return t.InputTokens + t.OutputTokens
}
```

### 2.7 Config（config 表）

> 键值对配置表，无独立 `domain` struct。

```go
// config 表结构（store/migrations.go）
// key         TEXT    PRIMARY KEY
// value       TEXT    NOT NULL
// updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP

// API 响应结构（api/response.go → ConfigResponse）
type ConfigResponse struct {
    Config map[string]interface{} `json:"config"`
}
```
