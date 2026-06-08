# 后端服务功能与配置

> 最后更新：2026-06-06

---

## 功能概述

后端服务（`backend` 子命令）是一个独立的 HTTP 服务器，使用 SQLite 存储提供 REST API，支撑 Issue 原子领取、Session 管理、系统状态查询等能力。Shell 侧的 Hook 和 Skill 通过 HTTP 调用后端 API。

### 核心能力

| 能力 | 说明 |
|------|------|
| **Issue 原子领取** | 通过 SQL 事务实现多 Agent 并发安全领取 |
| **Session 注册/追踪** | 记录会话的机器、项目、模型、状态等元信息 |
| **Token 统计** | 解析 Trace 文件汇总 Token 使用量 |
| **系统聚合状态** | 一次查询获取全局统计指标 |
| **配置管理** | 运行时读取/更新系统配置 |
| **代理注册管理** | 记录活跃代理实例 |

---

## 24 个 API 端点

### 健康检查

| 端点 | 方法 | 说明 |
|------|------|------|
| `/health` | GET | 返回 `{"status":"ok"}` |

### Issue 管理（6 个）

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/issue/check` | POST | 批量检查 Issue 状态（`CheckIssuesRequest` → `CheckIssuesResponse`） |
| `/api/issue/claim` | POST | 原子领取 Issue（`ClaimIssueRequest` → `ClaimIssueResponse`，冲突返回 409） |
| `/api/issue/release` | POST | 释放单个 Issue（`ReleaseIssueRequest` → `ReleaseIssueResponse`） |
| `/api/issue/release-session` | POST | 释放会话下所有 Issue（`ReleaseSessionRequest` → `ReleaseSessionResponse`） |
| `/api/issue/status` | POST | 更新 Issue 状态，返回 `previous_status`（`UpdateStatusRequest` → `UpdateStatusResponse`） |
| `/api/issues` | GET | Issue 列表（`?repo=&status=&session_id=&page=&page_size=`） |

### Session 管理（7 个）

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/session/register` | POST | 注册会话（`RegisterSessionRequest`） |
| `/api/session/close` | POST | 关闭会话（`CloseSessionRequest`） |
| `/api/sessions` | GET | 会话列表（`?machine_id=&project_slug=&status=`） |
| `/api/session/{id}` | GET | 单个会话详情（`SessionDetail`） |
| `/api/session/{id}/issues` | GET | 会话关联的 Issue 列表 |
| `/api/session/{id}/tokens` | GET | Token 统计（解析 Trace 文件计算） |
| `/api/session/{id}/traces` | GET | Trace 文件元数据列表 |

### Proxy 管理（4 个）

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/proxy/register` | POST | 代理注册 |
| `/api/proxy/unregister` | POST | 代理注销 |
| `/api/proxy/trace-init` | POST | 转发 trace-init |
| `/api/proxies` | GET | 代理列表（`?status=&project=`） |

### 基础设施（6 个）

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/machines` | GET | 机器列表（`?os=&hostname=`） |
| `/api/projects` | GET | 项目列表 |
| `/api/status` | GET | 系统聚合状态（`StatusResponse` 含 `SystemStats`） |
| `/api/logs` | GET | 日志查询（`?level=&date=&limit=`） |
| `/api/config` | GET | 获取系统配置 |
| `/api/config` | PUT | 更新系统配置（部分更新） |

---

## 数据模型

### 数据库表（6 个）

#### sessions 表

| 列 | 类型 | 约束 |
|----|------|------|
| id | INTEGER | PK AUTOINCREMENT |
| session_id | TEXT | NOT NULL UNIQUE |
| machine_id | TEXT | NOT NULL |
| os | TEXT | NOT NULL |
| project_slug | TEXT | NOT NULL |
| project_cwd | TEXT | NOT NULL |
| transcript_path | TEXT | NOT NULL |
| local_trace_path | TEXT | |
| model | TEXT | |
| source | TEXT | |
| status | TEXT | NOT NULL DEFAULT 'active' |
| registered_at | DATETIME | NOT NULL DEFAULT CURRENT_TIMESTAMP |
| closed_at | DATETIME | |
| close_reason | TEXT | |

索引：`machine_id`, `project_slug`, `status`, `registered_at`

#### issue_claims 表

| 列 | 类型 | 约束 |
|----|------|------|
| id | INTEGER | PK AUTOINCREMENT |
| repo_full_name | TEXT | NOT NULL |
| issue_number | INTEGER | NOT NULL |
| issue_title | TEXT | |
| status | TEXT | NOT NULL DEFAULT 'idle' |
| session_id | TEXT | |
| claimed_at | DATETIME | |
| updated_at | DATETIME | NOT NULL DEFAULT CURRENT_TIMESTAMP |

UNIQUE：`(repo_full_name, issue_number)`
索引：`repo_full_name`, `session_id`, `status`

#### machines 表

| 列 | 类型 | 约束 |
|----|------|------|
| id | INTEGER | PK AUTOINCREMENT |
| machine_id | TEXT | NOT NULL UNIQUE |
| os | TEXT | NOT NULL |
| hostname | TEXT | NOT NULL |
| username | TEXT | NOT NULL |
| first_seen_at | DATETIME | NOT NULL DEFAULT CURRENT_TIMESTAMP |
| last_seen_at | DATETIME | |

索引：`hostname`

#### projects 表

| 列 | 类型 | 约束 |
|----|------|------|
| id | INTEGER | PK AUTOINCREMENT |
| project_slug | TEXT | NOT NULL UNIQUE |
| project_cwd | TEXT | NOT NULL |
| first_seen_at | DATETIME | NOT NULL DEFAULT CURRENT_TIMESTAMP |
| last_seen_at | DATETIME | |

索引：`project_slug`

#### proxies 表

| 列 | 类型 | 约束 |
|----|------|------|
| proxy_id | TEXT | PK |
| project_slug | TEXT | NOT NULL |
| status | TEXT | NOT NULL DEFAULT 'active' |
| registered_at | DATETIME | NOT NULL |
| last_ping_at | DATETIME | |

索引：`status`, `project`

#### config 表

| 列 | 类型 | 约束 |
|----|------|------|
| key | TEXT | PK |
| value | TEXT | NOT NULL |
| updated_at | DATETIME | NOT NULL DEFAULT CURRENT_TIMESTAMP |

---

## Domain 结构体

### IssueStatus 枚举

```
idle → claimed → fixing → ready-for-pr → pr-created → testing → reviewing → merged
                                                                                  └→ rejected
```

### SessionStatus 枚举

```
active → closed
```

### TokenStats（运行时计算，无表）

```go
type TokenStats struct {
    APICalls     int
    InputTokens  int
    OutputTokens int
    CacheRead    int
    CacheCreate  int
}
// Total() = InputTokens + OutputTokens
```

---

## 配置说明

### 启动参数

```bash
go run ./cmd/claude-tap backend [选项]

选项:
  --host string    绑定地址（默认 "127.0.0.1"）
  --port / -p int  端口号（默认 8080）
  --db / -d string  SQLite 数据库路径（默认 "backend.db"）
```

### 配置桥接（`~/.claude-tap-plus/backend.json`）

```json
{"host": "127.0.0.1", "port": 8080}
```

| 时机 | 操作 |
|------|------|
| 后端启动 | 写入 `backend.json` |
| 后端退出 | 删除 `backend.json` |

### Shell 侧读取方式

`.claude/lib/config.sh` → `load_backend_config()`：
1. 检查 `backend.json` 是否存在
2. 解析 `host` 和 `port`
3. 设置 `$BACKEND_URL`（如 `http://127.0.0.1:8080`）

### 存储引擎

| 项目 | 说明 |
|------|------|
| 数据库 | SQLite |
| 模式 | WAL（Write-Ahead Logging） |
| 迁移 | 自动执行（启动时检查并创建缺失的表和索引） |
| 依赖 | `modernc.org/sqlite`（纯 Go，无需 CGO） |

---

## Shell ↔ Go 调用关系

| Shell 调用方 | HTTP 端点 | 触发时机 |
|-------------|-----------|---------|
| `hooks/01-session-start` | `POST /api/session/register` | 会话启动 |
| `hooks/01-session-start` | `POST /api/proxy/trace-init` | 会话启动 |
| `skills/001-4-issue-claim` | `POST /api/issue/check` | 领取前检查 |
| `skills/001-4-issue-claim` | `POST /api/issue/claim` | 原子领取 |
| `skills/001-5~9` | `POST /api/issue/status` | 状态流转 |
| `hooks/29-session-end` | `POST /api/issue/release-session` | 会话结束释放 |
| `hooks/29-session-end` | `POST /api/session/close` | 会话关闭 |

> **失败模式**：所有 Shell → 后端调用均为**静默降级**。后端不可用时 Shell 侧跳过 API 调用继续执行。
