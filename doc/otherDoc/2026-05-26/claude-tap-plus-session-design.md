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
    "trace_path": ".claude-tap-plus/.traces/Administrator@DESKTOP-ABC123/D--CodeDevelopment-CodeProject-claude-hk/bf15cac4-7235-48ce-8853-5d4598547f31.jsonl",
    "model": "GLM-5.1",
    "registered_at": "2026-05-26T21:06:16Z",
    "closed_at": null,
    "close_reason": null
  }
}
```

### 写入时机

| 时机 | 动作 | 写什么 |
|------|------|--------|
| SessionStart | 注册 | session_id、machine_id、os、project_slug、project_cwd、trace_path、model |
| SessionEnd | 注销 | 关闭时间、退出原因 |

---

## 五、实现方案

### 5.1 整体架构

```
┌─────────────────────────────────────────────────┐
│                  Claude Code 进程                 │
│                                                   │
│  SessionStart hook ──→ POST /api/session/register │
│                          (注册会话)                │
│                                                   │
│  SessionEnd hook   ──→ POST /api/session/close    │
│                          (注销会话)                │
└──────────────────┬──────────────────────────────┘
                   │ API 请求
                   ▼
┌──────────────────────────────────────────────────┐
│            claude-tap-plus 本地代理                │
│                                                    │
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

### 5.2 C1: 启动时注册（SessionStart hook）

**改造位置**：`.claude/hooks/01-session-start/base.sh`

```bash
register_session() {
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

  # 后端地址从配置文件读取，不存在则跳过
  local backend_url
  backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)
  [ -z "$backend_url" ] && return 0

  # 发送注册请求
  curl -s -X POST "$backend_url/api/session/register" \
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

**需要新增的配置文件**：`.claude/backend.conf`
```
BACKEND_URL=http://localhost:8080
```

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
  local session_id=$(json_get '.session_id')
  local reason=$(json_get '.reason')

  local backend_url
  backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)
  [ -z "$backend_url" ] && return 0

  curl -s -X POST "$backend_url/api/session/close" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$session_id\",\"reason\":\"$reason\"}" \
    > /dev/null 2>&1

  log "INFO" "Session unregistered from backend: $session_id"
}
```

---

## 六、后端 API 清单

| 接口 | 方法 | 说明 | 调用方 |
|------|------|------|--------|
| `/api/session/register` | POST | 注册会话，记录机器/项目/模型/trace 路径信息 | SessionStart hook |
| `/api/session/close` | POST | 注销会话，记录退出原因 | SessionEnd hook |
| `/api/issue/*` | - | Issue 管理接口 | 各 Issue skill |

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
| `.claude/backend.conf` | 后端地址配置文件，不存在时静默跳过 |
| `trace/writer.go` 改造 | 加入 machine_id 层级，session_id 从请求提取 |
| 后端服务 | 接收注册/注销请求，存会话元数据 |

### 7.3 后端服务技术选型建议

- HTTP API 服务，能接收 JSON
- 持久化存储（SQLite 足够起步，后续可换 PostgreSQL）
- Issue 状态管理需要内存缓存 + 持久化

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
```

### 10.6 写入时序

| 时机 | 表 | 操作 | 说明 |
|------|-----|------|------|
| SessionStart | machines | INSERT OR IGNORE / UPDATE last_seen_at | 机器首次出现则插入，否则更新最后出现时间 |
| SessionStart | projects | INSERT OR IGNORE / UPDATE last_seen_at | 项目首次出现则插入，否则更新最后出现时间 |
| SessionStart | sessions | INSERT | 注册新会话 |
| SessionEnd | sessions | UPDATE status='closed', closed_at, close_reason | 标记会话关闭 |

> **注意**：代理不上报 API 调用数据到后端，因此 sessions 表中的 call_count、total_tokens、last_active_at 等字段**不存在**。需要统计调用次数或查看消息内容时，直接读取本地 trace JSONL 文件。
