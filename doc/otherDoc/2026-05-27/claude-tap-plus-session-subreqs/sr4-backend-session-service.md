# SR-4: 后端会话管理服务

> 创建时间：2026-05-27
> 模块：claude-tap-plus / 后端
> 简述：新建后端 HTTP 服务，提供会话注册/注销 API 和 SQLite 持久化存储

---

## 目标

构建一个轻量级后端服务，接收 hooks 发来的会话注册/注销请求，持久化会话元数据到 SQLite，提供查询接口。

## 技术选型

| 项目 | 选择 | 理由 |
|------|------|------|
| 语言 | Go | 与 claude-tap-plus 同技术栈，可复用内部包 |
| 数据库 | SQLite | 单机部署，零配置，足够起步 |
| HTTP 框架 | `net/http` 或 `gin` | 轻量即可 |
| 数据库访问 | `database/sql` + `modernc.org/sqlite` | 纯 Go SQLite 驱动，无 CGO 依赖 |

## API 清单

### 1. POST `/api/session/register`

注册新会话。

**请求体：**
```json
{
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "machine_id": "Administrator@DESKTOP-ABC123",
  "os": "windows",
  "project_slug": "D--CodeDevelopment-CodeProject-claude-hk",
  "project_cwd": "D:\\CodeDevelopment\\CodeProject\\claude-hk",
  "transcript_path": "C:\\Users\\Administrator\\.claude\\projects\\...",
  "local_trace_path": ".claude-tap-plus/.traces/Administrator@DESKTOP-ABC123/D--CodeDevelopment-CodeProject-claude-hk/bf15cac4-...jsonl",
  "model": "GLM-5.1",
  "source": "startup"
}
```

**处理逻辑：**
1. INSERT OR IGNORE 到 `machines` 表，UPDATE `last_seen_at`。`username` 和 `hostname` 从 `machine_id` 按 `@` 分割解析获取
2. INSERT OR IGNORE 到 `projects` 表，UPDATE `last_seen_at`
3. INSERT 到 `sessions` 表

**响应：** `200 OK` 或 `409 Conflict`（session_id 已存在）

### 2. POST `/api/session/close`

注销会话。

**请求体：**
```json
{
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "reason": "prompt_input_exit"
}
```

**处理逻辑：**
1. UPDATE `sessions` SET `status='closed'`, `closed_at=NOW`, `close_reason`
2. 如果 session 不存在或已关闭，返回 `404` 或 `409`

**响应：** `200 OK` 或 `404 Not Found`

### 3. GET `/api/sessions`

查询会话列表，支持过滤。

**查询参数：**
| 参数 | 说明 |
|------|------|
| `machine_id` | 按机器过滤 |
| `project_slug` | 按项目过滤 |
| `status` | 按状态过滤（active/closed） |

### 4. GET `/api/session/:id`

查询单个会话详情。

### Issue 管理 API

后端还包含 `/api/issue/*` 系列 API，由独立的 `issue-management-reqs/` 模块定义。Session 会话管理和 Issue 管理共享同一个后端服务实例。

## 数据库设计

### sessions 表

```sql
CREATE TABLE sessions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id      TEXT NOT NULL UNIQUE,
    machine_id      TEXT NOT NULL,
    os              TEXT NOT NULL,
    project_slug    TEXT NOT NULL,
    project_cwd     TEXT NOT NULL,
    transcript_path  TEXT NOT NULL,
    local_trace_path TEXT,
    model           TEXT,
    source          TEXT,
    status          TEXT NOT NULL DEFAULT 'active',
    registered_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at       DATETIME,
    close_reason    TEXT
);

CREATE INDEX idx_sessions_machine ON sessions(machine_id);
CREATE INDEX idx_sessions_project ON sessions(project_slug);
CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_sessions_registered ON sessions(registered_at);
```

### machines 表

```sql
CREATE TABLE machines (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    machine_id      TEXT NOT NULL UNIQUE,
    os              TEXT NOT NULL,
    hostname        TEXT NOT NULL,
    username        TEXT NOT NULL,
    first_seen_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at    DATETIME
);

CREATE INDEX idx_machines_hostname ON machines(hostname);
```

### projects 表

```sql
CREATE TABLE projects (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    project_slug    TEXT NOT NULL UNIQUE,
    project_cwd     TEXT NOT NULL,
    first_seen_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at    DATETIME
);

CREATE INDEX idx_projects_slug ON projects(project_slug);
```

### ER 关系

```
machines (1:N) ──→ sessions (N:1) ←── projects
                      │
                      └→ 本地 JSONL trace 文件（不存 DB）
```

## 写入时序

| 时机 | 表 | 操作 |
|------|-----|------|
| POST /register | machines | INSERT OR IGNORE / UPDATE last_seen_at |
| POST /register | projects | INSERT OR IGNORE / UPDATE last_seen_at |
| POST /register | sessions | INSERT（含 transcript_path） |
| proxy 首次拦截 | sessions | UPDATE local_trace_path（构造路径） |
| POST /close | sessions | UPDATE status, closed_at, close_reason |

## 项目结构建议

```
claude_tap_plus/
├── cmd/
│   ├── claude-tap/          # 现有代理 CLI
│   └── claude-tap-server/   # 新增：后端服务 CLI
├── internal/
│   ├── session/             # 现有：session push/pull
│   ├── backend/             # 新增：后端服务
│   │   ├── server.go        # HTTP server 生命周期
│   │   ├── handler.go       # API handler
│   │   ├── db.go            # SQLite 初始化 + 迁移
│   │   └── model.go         # 数据模型
│   └── ...
```

## 启动时超时清理

后端服务启动时建议执行一次超时清理，将上次运行中残留的 `active` 会话标记为 `closed`：

```sql
UPDATE sessions
SET status = 'closed',
    closed_at = datetime('now'),
    close_reason = 'timeout_cleanup'
WHERE status = 'active'
  AND registered_at < datetime('now', '-24 hours');
```

## 验证标准

- [ ] `go build ./cmd/claude-tap-server` 编译通过
- [ ] POST `/api/session/register` 正确写入三张表
- [ ] POST `/api/session/close` 正确更新 session 状态
- [ ] GET `/api/sessions` 支持按 machine/project/status 过滤
- [ ] GET `/api/session/:id` 返回完整会话详情
- [ ] SQLite 数据库文件自动创建和迁移
- [ ] 重复注册同一 session_id 返回 409 而非报错
- [ ] 关闭不存在的 session 返回 404

## 依赖

无外部依赖，可独立开发。后续由 SR-5 进行集成测试。
