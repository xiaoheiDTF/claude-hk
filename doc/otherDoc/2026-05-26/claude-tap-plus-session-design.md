# claude-tap-plus 会话管理设计：本地代理改造 + 后端关联

> 创建时间：2026-05-26
> 模块：claude-tap-plus
> 简述：利用 hooks SessionStart/SessionEnd 实现会话注册与注销，通过 transcript_path 关联本地 trace 文件，后端只存会话元数据不存消息内容

---

## 一、设计原则

1. **消息存储在本地**：API 调用的请求/响应由 claude-tap-plus 代理写入本地 JSONL trace 文件，后端不存储消息内容
2. **后端只存关联表**：后端仅记录 session → 机器 → 项目 → trace 文件路径 的映射关系
3. **trace 路径跟 Claude Code 一致**：复用 Claude Code 的 transcript_path 结构，便于溯源

---

## 二、可获取的数据源

### 2.1 SessionStart Hook 提供的字段

```json
{
  "session_id": "3cc83fcb-2849-4beb-9df0-6c638ab60566",
  "transcript_path": "C:\\Users\\Administrator\\.claude\\projects\\D--CodeDevelopment-CodeProject-claude-hk\\3cc83fcb-2849-4beb-9df0-6c638ab60566.jsonl",
  "cwd": "D:\\CodeDevelopment\\CodeProject\\claude-hk",
  "hook_event_name": "SessionStart",
  "source": "startup",
  "model": "GLM-5.1"
}
```

| 字段 | 说明 | 用途 |
|------|------|------|
| `session_id` | UUID，唯一标识一次会话 | 后端注册的主键 |
| `transcript_path` | 本地会话 JSONL 文件路径 | 消息溯源、提取项目标识 |
| `cwd` | 当前项目根目录 | 项目识别 |
| `source` | 启动来源（startup/resume） | 区分首次启动和恢复 |
| `model` | 使用的模型名称 | 环境记录 |

### 2.2 SessionEnd Hook 提供的字段

```json
{
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "transcript_path": "C:\\Users\\Administrator\\.claude\\projects\\...",
  "cwd": "D:\\CodeDevelopment\\CodeProject\\claude-hk",
  "hook_event_name": "SessionEnd",
  "reason": "prompt_input_exit"
}
```

| 字段 | 说明 | 用途 |
|------|------|------|
| `session_id` | 同上 | 注销匹配 |
| `reason` | 退出原因 | 判断正常退出还是异常中断 |

### 2.3 平台信息（hooks/platform.sh 已有）

| 数据 | 获取方式 | 示例值 |
|------|----------|--------|
| OS 类型 | `uname -s` | windows / linux / macos |
| 机器名 | `hostname` | DESKTOP-XXX |
| 用户 | `whoami` | Administrator |

### 2.4 claude-tap-plus 代理已有的数据

代理拦截每次 API 调用，已记录到本地 JSONL：

| 数据 | 说明 |
|------|------|
| request body | 完整请求 JSON（含 messages、tools） |
| response body | 完整响应 JSON（含 content、usage） |
| session_id | 从请求元数据中提取 |
| timestamp | 每次调用时间 |
| duration_ms | 请求耗时 |
| token 统计 | input/output/cache tokens |
| model | 实际调用的模型 |

---

## 三、trace 存储路径设计

### 3.1 路径规则

复用 Claude Code 的 transcript_path 结构，trace 文件路径格式：

```
.claude-tap-plus/.traces/{machine_id}/{project_slug}/{session_id}.jsonl
```

示例：
```
.claude-tap-plus/.traces/Administrator@DESKTOP-ABC123/D--CodeDevelopment-CodeProject-claude-hk/bf15cac4-7235-48ce-8853-5d4598547f31.jsonl
```

四层目录：
1. `.claude-tap-plus/` — 工具根目录
2. `.traces/` — trace 存储目录
3. `{machine_id}/` — 机器标识（`whoami@hostname`）
4. `{project_slug}/` — 项目标识（从 transcript_path 提取）

### 3.2 从 transcript_path 解析标识

transcript_path 包含所有需要的信息：

```
C:\Users\Administrator\.claude\projects\D--CodeDevelopment-CodeProject-claude-hk\bf15cac4-7235-48ce-8853-5d4598547f31.jsonl
│                                          │                                              │
│                                          │                                              └─ session_id
│                                          │
│                                          └─ project_slug（Claude Code 内置的项目标识）
│                                             由 cwd 的路径分隔符替换为 - 得到
│
└─ 用户主目录 → 机器/用户标识
```

解析方式：

```bash
# 提取 project_slug
project_slug=$(echo "$transcript_path" | sed -E 's|.*/projects/([^/]+)/.*|\1|')

# 提取用户主目录
home_projects=$(echo "$transcript_path" | sed -E 's|(.*/.claude)/projects/.*|\1|')
user_home=$(echo "$home_projects" | sed 's|/\.claude$||')
```

### 3.3 最终标识组合

```
project_id = "{hostname}/{project_slug}"
```

| 示例 | 含义 |
|------|------|
| `DESKTOP-ABC123/D--CodeDevelopment-CodeProject-claude-hk` | DESKTOP-ABC123 上的 claude-hk 项目 |

---

## 四、后端数据结构

后端只存一个关联表，trace 文件内容在本地：

```json
{
  "bf15cac4-7235-48ce-8853-5d4598547f31": {
    "machine_id": "Administrator@DESKTOP-ABC123",
    "os": "windows",
    "project_slug": "D--CodeDevelopment-CodeProject-claude-hk",
    "project_cwd": "D:\\CodeDevelopment\\CodeProject\\claude-hk",
    "trace_path": "C:\\Users\\Administrator\\.claude\\projects\\D--CodeDevelopment-CodeProject-claude-hk\\bf15cac4-7235-48ce-8853-5d4598547f31.jsonl",
    "model": "GLM-5.1",
    "registered_at": "2026-05-26T21:06:16Z",
    "closed_at": null,
    "close_reason": null
  }
}
```

> **注**：`trace_path` 字段存储 Claude Code 原始 `transcript_path`，便于溯源；本地重组后的 trace 路径由代理内部管理，不发送到后端。

### 写入时机

| 时机 | 动作 | 写什么 |
|------|------|--------|
| SessionStart | 注册 | session_id、machine_id、os、project_slug、project_cwd、trace_path、model |
| SessionEnd | 注销 | 关闭时间、退出原因 |

---

## 五、实现方案

### 5.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      Claude Code 进程                         │
│                                                               │
│  SessionStart hook ──→ 检测代理状态 ──→ POST /api/session/...│
│  SessionEnd   hook ──→ 检测代理状态 ──→ POST /api/session/...│
└──────────────────────────┬──────────────────────────────────┘
                           │ 代理状态：$HOME/.claude-tap-plus/.proxy.json（仅 PID）
                           │ 后端地址：$CLAUDE_PROJECT_DIR/.claude/backend.conf
                           ▼
┌──────────────────────────────────────────────────┐
│            claude-tap-plus 本地代理                │
│                                                    │
│  启动时写 .proxy.json（pid）                       │
│  退出时删除 .proxy.json                             │
│  拦截请求/响应 ──→ 写入本地 JSONL trace 文件        │
│                  （不发送到后端）                   │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────┐
│              后端服务器 (待开发)                     │
│                                                    │
│  /api/session/register  → 注册会话                 │
│  /api/session/close     → 注销会话                 │
│  /api/issue/*           → Issue 管理接口           │
└────────────────────────────────────────────────────┘
```

#### 代理状态检测机制

Hook 通过读取代理状态文件判断 claude-tap-plus 是否在运行，不需要额外配置文件。

**状态文件**：`$HOME/.claude-tap-plus/.proxy.json`（仅包含 PID，用于检测代理是否在运行）

```json
{ "pid": 12345 }
```

**后端地址配置**：`$CLAUDE_PROJECT_DIR/.claude/backend.conf`（静态配置，与 Issue 技能共用）

```
BACKEND_URL=http://localhost:8080
```

**写入时机**：`.proxy.json` 在代理启动时创建，正常退出时删除。

**检测逻辑**（共享函数，放入 `hooks/base.sh`）：

```bash
# 检测 claude-tap-plus 代理是否在运行
# 成功时设置 BACKEND_URL，返回 0
# 代理未运行返回 1（静默跳过）
check_proxy_active() {
  local state_file="$HOME/.claude-tap-plus/.proxy.json"
  [ ! -f "$state_file" ] && return 1

  local pid
  pid=$(jq -r '.pid // empty' "$state_file" 2>/dev/null)
  [ -z "$pid" ] && return 1

  # 检查进程是否存活（处理异常退出残留的文件）
  kill -0 "$pid" 2>/dev/null || return 1

  # 后端地址从 backend.conf 读取（不依赖 .proxy.json）
  BACKEND_URL=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)
  [ -n "$BACKEND_URL" ] && return 0
  return 1
}
```

**覆盖的场景**：

| 场景 | 状态文件 | PID | 行为 |
|------|----------|-----|------|
| 用户直接运行 `claude` | 不存在 | - | 跳过，不发送请求 |
| 用户通过 `claude-tap-plus claude` 启动 | 存在 | 存活 | 正常注册/注销 |
| 代理异常退出（kill -9） | 存在 | 已死 | 跳过（kill -0 失败） |
| 代理正常退出 | 已删除 | - | 跳过 |
| backend.conf 未配置 | 存在 | 存活 | 跳过（BACKEND_URL 为空） |

### 5.2 C1: 启动时注册（SessionStart hook）

**改造位置**：`.claude/hooks/01-session-start/base.sh`

```bash
register_session() {
  # 检测代理是否在运行，不在则跳过
  check_proxy_active || return 0

  local session_id=$(json_get '.session_id')
  local transcript_path=$(json_get '.transcript_path')
  local cwd=$(json_get '.cwd')
  local model=$(json_get '.model')
  local source=$(json_get '.source')

  # 组装机器信息
  local machine_id="$(whoami)@$(hostname)"
  local os_type="$OS_TYPE"

  # 从 transcript_path 提取 project_slug
  local project_slug=""
  project_slug=$(echo "$transcript_path" | sed -E 's|.*/projects/([^/]+)/.*|\1|')

  # 发送注册请求（BACKEND_URL 由 check_proxy_active 从 backend.conf 读取）
  curl -s --max-time 5 -X POST "$BACKEND_URL/api/session/register" \
    -H "Content-Type: application/json" \
    -d "{
      \"session_id\":\"$session_id\",
      \"machine_id\":\"$machine_id\",
      \"os\":\"$os_type\",
      \"project_slug\":\"$project_slug\",
      \"project_cwd\":\"$cwd\",
      \"trace_path\":\"$transcript_path\",
      \"model\":\"$model\",
      \"source\":\"$source\"
    }" > /dev/null 2>&1

  log "INFO" "Session registered to backend: $session_id"
}
```

**后端地址来源**：从 `.claude/backend.conf` 静态配置读取，由 `check_proxy_active()` 设置到 `BACKEND_URL` 变量。`.proxy.json` 仅用于 PID 检测。

### 5.3 C2: 消息存储（claude-tap-plus 代理）

**改造位置**：`claude_tap_plus/internal/trace/writer.go`

代理在每次 API 调用完成时，将请求/响应写入本地 JSONL trace 文件。**不发送到后端**。

```
请求完成 → trace writer 写本地 JSONL（完成）
```

trace 文件路径改为：
```
.claude-tap-plus/.traces/{machine_id}/{project_slug}/{session_id}.jsonl
```

改造点：`trace/writer.go` 的 `NewTracePath()` 中加入 machine_id 层级，session_id 从代理拦截的请求里 `extractSessionID()` 获取，首次拦截到时创建文件，后续同 session 追加写入。

### 5.4 C3: 退出时注销（SessionEnd hook）

**改造位置**：`.claude/hooks/29-session-end/base.sh`

```bash
unregister_session() {
  # 检测代理是否在运行，不在则跳过
  check_proxy_active || return 0

  local session_id=$(json_get '.session_id')
  local reason=$(json_get '.reason')

  curl -s --max-time 5 -X POST "$BACKEND_URL/api/session/close" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$session_id\",\"reason\":\"$reason\"}" \
    > /dev/null 2>&1

  log "INFO" "Session unregistered from backend: $session_id"
}
```

> **SessionEnd hook 双重职责**：除了注销会话，SessionEnd hook 还需调用 `/api/issue/release-session` 释放该 session 领取的所有 issue（见 skill-integration-design.md 5.9 节）。建议执行顺序：先释放 issue，再注销 session。

---

## 六、后端 API 清单

| 接口 | 方法 | 说明 | 调用方 |
|------|------|------|--------|
| `/api/session/register` | POST | 注册会话，记录机器/项目/模型/trace 路径信息 | SessionStart hook |
| `/api/session/close` | POST | 注销会话，记录退出原因 | SessionEnd hook |
| `/api/issue/*` | - | Issue 管理接口 | 各 Issue skill |
| `/health` | GET | 健康检查，返回 `{"status":"ok"}` | `_backend_available()` |
| `/api/sessions` | GET | 查询会话列表，支持按 machine_id / project_slug / status 过滤 | 后端管理/调试 |
| `/api/session/:id` | GET | 查询单个会话详情（含关联 machine 和 project 信息） | 后端管理/调试 |

> 注：`/api/message/store` 和 `/api/session/heartbeat` 不需要，消息存储在本地 JSONL，心跳由代理自身管理。

---

## 七、需要的依赖和前置条件

### 7.1 已有的（无需额外安装）

- `curl` — hooks 环境中已可用
- `json_get()` — hooks/base.sh 已提供 JSON 解析能力
- `platform.sh` — 已有 OS 检测
- `jq` 或 Python JSON 解析 — 已有降级链

### 7.2 需要新增的

| 项目 | 说明 |
|------|------|
| `.claude/backend.conf` | 后端地址配置文件，格式 `BACKEND_URL=http://localhost:8080`（与 Issue 技能共用） |
| `hooks/base.sh` 改造 | 新增 `check_proxy_active()` 共享函数 |
| `trace/writer.go` 改造 | 加入 machine_id 层级，session_id 从请求提取 |
| 代理启动/退出改造 | 启动时写 `$HOME/.claude-tap-plus/.proxy.json`（仅 PID），退出时删除 |
| 后端服务 | 接收注册/注销请求，存会话元数据 |

### 7.3 后端服务技术选型

| 项目 | 选择 | 理由 |
|------|------|------|
| 语言 | Go | 与 claude-tap-plus 同技术栈，可复用内部包 |
| 数据库 | SQLite | 单机部署，零配置，足够起步 |
| HTTP 框架 | `net/http` 或 `gin` | 轻量即可 |
| 数据库访问 | `database/sql` + `modernc.org/sqlite` | 纯 Go SQLite 驱动，无 CGO 依赖 |

### 7.4 后端服务代码结构

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

---

## 八、数据流总结

```
用户启动 claude-tap-plus claude
  │
  ├─ 代理启动，监听本地端口
  ├─ Claude Code 启动
  │    │
  │    ├─ SessionStart hook 触发
  │    │    └─ POST /api/session/register → 后端记录会话元数据
  │    │
  │    ├─ 用户交互中...
  │    │    │
  │    │    ├─ 每次 API 请求 → 代理拦截
  │    │    │    ├─ 转发到上游 API
  │    │    │    └─ 本地 JSONL trace 文件写入（不发送到后端）
  │    │    │
  │    │    └─ Issue 技能调用 → 查询后端状态
  │    │
  │    └─ SessionEnd hook 触发
  │         ├─ POST /api/issue/release-session → 释放该 session 领取的所有 issue
  │         └─ POST /api/session/close → 后端注销会话
  │
  └─ 代理退出，打印汇总
```

---

## 九、查询示例

### 按 session_id 查询

```
session_id: bf15cac4-7235-48ce-8853-5d4598547f31
  → machine:  Administrator@DESKTOP-ABC123 (windows)
  → project:  D--CodeDevelopment-CodeProject-claude-hk
  → cwd:      D:\CodeDevelopment\CodeProject\claude-hk
  → trace:    .claude-tap-plus/.traces/Administrator@DESKTOP-ABC123/D--CodeDevelopment-CodeProject-claude-hk/bf15cac4-7235-48ce-8853-5d4598547f31.jsonl
```

### 按项目查所有会话

```
project_slug: D--CodeDevelopment-CodeProject-claude-hk
  → session bf15cac4... (已结束, 15次调用)
  → session 3cc83fcb... (活跃, 28次调用)
  → session 6b6f69d3... (活跃, 3次调用)
```

需要看消息内容时，去对应机器的 trace JSONL 文件里找。

---

## 十、数据库表设计

> **设计原则**：后端只存**确定能从 hook 获取到**的元数据。消息内容、调用次数、token 数等由本地 JSONL 存储，后端不存。

### 10.1 会话表（sessions）

```sql
CREATE TABLE sessions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id      TEXT NOT NULL UNIQUE,           -- UUID，SessionStart 传入
    machine_id      TEXT NOT NULL,                  -- whoami@hostname，系统命令获取
    os              TEXT NOT NULL,                  -- windows/linux/macos，platform.sh
    project_slug    TEXT NOT NULL,                  -- 从 transcript_path 解析
    project_cwd     TEXT NOT NULL,                  -- SessionStart 的 cwd 字段
    trace_path      TEXT NOT NULL,                  -- transcript_path（Claude Code 的原始路径）
    model           TEXT,                           -- SessionStart 的 model 字段
    source          TEXT,                           -- startup/resume，SessionStart 的 source
    status          TEXT NOT NULL DEFAULT 'active', -- active / closed
    registered_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at       DATETIME,                       -- SessionEnd 时更新
    close_reason    TEXT                            -- SessionEnd 的 reason 字段
);

-- 索引
CREATE INDEX idx_sessions_machine ON sessions(machine_id);
CREATE INDEX idx_sessions_project ON sessions(project_slug);
CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_sessions_registered ON sessions(registered_at);
```

**字段来源说明**：

| 字段 | 来源 | 获取方式 |
|------|------|----------|
| session_id | SessionStart/End | `json_get '.session_id'` |
| machine_id | 系统命令 | `whoami` + `@` + `hostname` |
| os | platform.sh | `uname -s` 已解析为 OS_TYPE |
| project_slug | transcript_path | `sed` 从路径中提取 |
| project_cwd | SessionStart | `json_get '.cwd'` |
| trace_path | SessionStart | `json_get '.transcript_path'`（直接用 Claude Code 的路径） |
| model | SessionStart | `json_get '.model'` |
| source | SessionStart | `json_get '.source'` |
| close_reason | SessionEnd | `json_get '.reason'` |

### 10.2 机器表（machines）

```sql
CREATE TABLE machines (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    machine_id      TEXT NOT NULL UNIQUE,           -- whoami@hostname
    os              TEXT NOT NULL,                  -- platform.sh
    hostname        TEXT NOT NULL,                  -- `hostname` 命令
    username        TEXT NOT NULL,                  -- `whoami` 命令
    first_seen_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at    DATETIME                        -- 每次 SessionStart 更新
);

-- 索引
CREATE INDEX idx_machines_hostname ON machines(hostname);
```

> **字段来源说明**：`username` 和 `hostname` 从请求中的 `machine_id` 字段按 `@` 分割解析获取，无需客户端单独传入。

### 10.3 项目表（projects）

```sql
CREATE TABLE projects (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    project_slug    TEXT NOT NULL UNIQUE,           -- 从 transcript_path 解析
    project_cwd     TEXT NOT NULL,                  -- 首次出现的 cwd
    first_seen_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at    DATETIME                        -- 每次该项目有新会话时更新
);

-- 索引
CREATE INDEX idx_projects_slug ON projects(project_slug);
```

### 10.4 ER 关系图

```
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│  machines   │       │  sessions   │       │  projects   │
├─────────────┤       ├─────────────┤       ├─────────────┤
│ machine_id  │◄──────│ machine_id  │       │ project_slug│◄──┐
│ os          │  1:N  │ project_slug│──────►│ project_cwd │   │
│ hostname    │       │ session_id  │  N:1  │ first_seen  │   │
│ username    │       │ trace_path  │       │ last_seen   │   │
│ first_seen  │       │ status      │       └─────────────┘   │
│ last_seen   │       │ registered  │                         │
└─────────────┘       │ closed_at   │                         │
                      │ close_reason│                         │
                      └─────────────┘                         │
                                                              │
                      ┌───────────────────────────────────────┘
                      │
                      ▼
              ┌───────────────┐
              │ 本地 JSONL    │
              │ trace 文件    │
              │ (不存到DB)    │
              └───────────────┘
```

### 10.5 核心查询示例

```sql
-- 1. 查询某机器的所有会话
SELECT session_id, project_slug, status, registered_at, closed_at
FROM sessions
WHERE machine_id = 'Administrator@DESKTOP-ABC123'
ORDER BY registered_at DESC;

-- 2. 查询某项目的所有会话
SELECT session_id, machine_id, status, registered_at, closed_at
FROM sessions
WHERE project_slug = 'D--CodeDevelopment-CodeProject-claude-hk'
ORDER BY registered_at DESC;

-- 3. 查询某会话详情（含 trace 路径，直接去本地文件看消息）
SELECT s.*, m.hostname, m.os, p.project_cwd
FROM sessions s
JOIN machines m ON s.machine_id = m.machine_id
JOIN projects p ON s.project_slug = p.project_slug
WHERE s.session_id = 'bf15cac4-7235-48ce-8853-5d4598547f31';

-- 4. 统计某项目的会话数（仅注册数，不统计调用次数）
SELECT 
    COUNT(*) as session_count,
    COUNT(CASE WHEN status = 'active' THEN 1 END) as active_count,
    COUNT(CASE WHEN status = 'closed' THEN 1 END) as closed_count
FROM sessions
WHERE project_slug = 'D--CodeDevelopment-CodeProject-claude-hk';

-- 5. 列出所有机器及其项目
SELECT 
    m.machine_id,
    m.os,
    m.hostname,
    COUNT(DISTINCT s.project_slug) as project_count,
    COUNT(s.session_id) as total_sessions
FROM machines m
LEFT JOIN sessions s ON m.machine_id = s.machine_id
GROUP BY m.machine_id;

-- 6. 查询会话列表（GET /api/sessions，支持过滤）
SELECT session_id, machine_id, project_slug, status, registered_at, closed_at
FROM sessions
WHERE (? IS NULL OR machine_id = ?)
  AND (? IS NULL OR project_slug = ?)
  AND (? IS NULL OR status = ?)
ORDER BY registered_at DESC;

-- 7. 查询单个会话详情（GET /api/session/:id，含关联信息）
SELECT s.*, m.hostname, m.os, m.username, p.project_cwd
FROM sessions s
JOIN machines m ON s.machine_id = m.machine_id
JOIN projects p ON s.project_slug = p.project_slug
WHERE s.session_id = ?;
```

### 10.6 写入时序

| 时机 | 表 | 操作 | 说明 |
|------|-----|------|------|
| SessionStart | machines | INSERT OR IGNORE / UPDATE last_seen_at | 机器首次出现则插入，否则更新最后出现时间 |
| SessionStart | projects | INSERT OR IGNORE / UPDATE last_seen_at | 项目首次出现则插入，否则更新最后出现时间 |
| SessionStart | sessions | INSERT | 注册新会话 |
| SessionEnd | sessions | UPDATE status='closed', closed_at, close_reason | 标记会话关闭 |

> **注意**：代理不上报 API 调用数据到后端，因此 sessions 表中的 call_count、total_tokens、last_active_at 等字段**不存在**。需要统计调用次数或查看消息内容时，直接读取本地 trace JSONL 文件。

---

## 十一、验证标准

### 11.1 代理 trace 路径重构（对应 C2）

- [ ] trace 文件路径包含 `machine_id` 和 `project_slug` 层级
- [ ] 目录不存在时自动创建（`os.MkdirAll`）
- [ ] 同一 session 的多次 API 调用追加写入同一 trace 文件
- [ ] 跨平台路径分隔符处理正确（Windows `\` vs Linux `/`）
- [ ] 现有 trace 写入内容格式不变（只改路径，不改内容）

### 11.2 SessionStart 会话注册（对应 C1）

- [ ] 代理运行时（`.proxy.json` 存在且 PID 存活，`backend.conf` 已配置），启动 Claude Code 后能在后端 DB 中查到会话记录
- [ ] 代理未运行时（`.proxy.json` 不存在或 PID 已死），启动无报错，不发送请求
- [ ] backend.conf 未配置时，启动无报错，不发送请求
- [ ] 后端不可达时，启动无报错，curl 静默失败
- [ ] 注册请求包含全部 8 个字段（session_id / machine_id / os / project_slug / project_cwd / trace_path / model / source）
- [ ] `source` 正确区分 `startup` 和 `resume`
- [ ] machines 表和 projects 表自动 INSERT OR IGNORE，`last_seen_at` 更新

### 11.3 SessionEnd 会话注销（对应 C3）

- [ ] 正常退出 Claude Code 后，后端 session 状态变为 `closed`
- [ ] 后端记录了 `close_reason` 和 `closed_at` 时间
- [ ] 代理未运行时，退出无报错
- [ ] 后端不可达时，退出无报错

### 11.4 后端服务

- [ ] `go build ./cmd/claude-tap-server` 编译通过
- [ ] POST `/api/session/register` 正确写入 sessions / machines / projects 三张表
- [ ] POST `/api/session/close` 正确更新 session 状态
- [ ] GET `/api/sessions` 支持按 machine_id / project_slug / status 过滤
- [ ] GET `/api/session/:id` 返回完整会话详情（含关联的 machine 和 project 信息）
- [ ] SQLite 数据库文件自动创建和迁移
- [ ] 重复注册同一 session_id 返回 `409 Conflict`，不重复创建
- [ ] 关闭不存在的 session 返回 `404 Not Found`

### 11.5 端到端集成验证

完整流程 6 步：

```
1. 启动后端服务：go run ./cmd/claude-tap-server
   → 验证：服务监听在配置的端口

2. 确认代理状态文件和后端配置
   → 验证：`$HOME/.claude-tap-plus/.proxy.json` 包含 pid
   → 验证：`.claude/backend.conf` 包含 BACKEND_URL

3. 通过 claude-tap-plus 启动 Claude Code
   → 验证 trace 路径结构（11.1）
   → 验证会话注册（11.2）

4. 进行几次用户交互（触发 API 调用）
   → 验证 trace 文件正确追加写入

5. 退出 Claude Code
   → 验证会话注销（11.3）

6. 查询后端
   → 验证 GET /api/sessions 返回正确的会话列表
   → 验证 GET /api/session/:id 返回完整元数据
```

---

## 十二、异常场景与容错策略

### 12.1 静默失败原则

Hook 中的注册/注销请求采用**尽力而为**策略，任何失败都不应阻塞 Claude Code 的正常启动或退出：

| 失败场景 | 行为 | 原因 |
|----------|------|------|
| `.proxy.json` 不存在（用户直接运行 `claude`） | 跳过注册/注销，不发送请求 | 代理未启动，不需要会话管理 |
| `.proxy.json` 存在但 PID 已死 | 跳过（`kill -0` 失败） | 代理异常退出残留文件，自动忽略 |
| `.proxy.json` 存在且 PID 存活，但 backend.conf 未配置 | 跳过（BACKEND_URL 为空） | 后端地址缺失，不发送请求 |
| `.proxy.json` 存在且 PID 存活，但后端不可达 | curl 静默失败（`-s` + `> /dev/null 2>&1`） | 后端宕机不应影响 Claude Code 使用 |
| 后端返回非 200（如 409/500） | 忽略响应，继续执行 | Hook 不处理业务层错误 |

### 12.2 异常退出场景

Claude Code 异常退出（kill -9、系统断电、OOM 等）时，SessionEnd hook **不会触发**，后端 session 状态将停留在 `active`。

| 场景 | 后端状态 | 处理方式 |
|------|----------|----------|
| 用户输入 `exit` 正常退出 | session → `closed`，记录 `close_reason` | SessionEnd hook 正常触发 |
| Ctrl+C 中断退出 | session → `closed`，reason 待观察 | SessionEnd hook 是否触发取决于 Claude Code 实现 |
| kill -9 强杀进程 | session 保持 `active` | SessionEnd 不触发，需要后端清理 |
| 系统断电/崩溃 | session 保持 `active` | SessionEnd 不触发，需要后端清理 |

**超时清理策略**（后端可选实现）：

```sql
-- 将超过 24 小时仍为 active 的会话标记为 closed
UPDATE sessions
SET status = 'closed',
    closed_at = datetime('now'),
    close_reason = 'timeout_cleanup'
WHERE status = 'active'
  AND registered_at < datetime('now', '-24 hours');
```

建议作为后端定时任务或启动时执行一次。

### 12.3 异常场景测试矩阵

| 场景 | 预期行为 | 验证方法 |
|------|----------|----------|
| 后端未启动就开 Claude Code | 注册失败静默忽略，trace 正常写本地 | 直接运行 `claude`（不通过代理），检查无报错 |
| 会话中途后端重启 | 注销请求失败，session 停留 `active` | 会话中途 kill 后端，退出后查询 DB |
| 同一 session_id 重复注册 | 后端返回 `409 Conflict`，不重复创建 | 手动 curl 发两次相同注册请求 |
| 异常退出（kill -9） | SessionEnd 不触发，session 保持 `active` | kill -9 后查询后端 DB |
| 代理异常退出残留 `.proxy.json` | PID 已死，`check_proxy_active` 返回失败，跳过注册 | 手动创建含过期 PID 的 `.proxy.json`，启动无报错 |
| 不使用 claude-tap-plus 直接运行 `claude` | `.proxy.json` 不存在，完全跳过 | 直接运行 `claude`，检查 hook 日志无注册请求 |

### 12.4 退出原因（reason）枚举

SessionEnd hook 的 `reason` 字段记录退出原因：

| reason | 含义 |
|--------|------|
| `prompt_input_exit` | 用户主动输入 exit 退出 |
| 其他值 | 待观察实际行为，可能包括异常中断等 |

> `reason` 的完整枚举值需要通过实际使用观察补充。Claude Code 文档未列出所有可能值。
