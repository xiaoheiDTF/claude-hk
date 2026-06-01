# claude-tap-plus 数据库表结构汇总

> 创建时间：2026-05-28
> 来源：汇总 `doc/otherDoc/2026-05-26` 和 `2026-05-27` 所有设计文档中的表结构定义
> 数据库：SQLite（单机部署，零配置）

---

## 总览

共 4 张表，分为两大业务域：

| 业务域 | 表名 | 说明 | 归属包 |
|--------|------|------|--------|
| 会话管理 | `machines` | 机器信息 | `internal/backend/store/session_store.go` |
| 会话管理 | `projects` | 项目信息 | `internal/backend/store/session_store.go` |
| 会话管理 | `sessions` | 会话注册记录 | `internal/backend/store/session_store.go` |
| Issue 管理 | `issue_claims` | Issue 领取与状态追踪 | `internal/backend/store/issue_store.go` |

---

## ER 关系图

```
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│  machines   │       │  sessions   │       │  projects   │
├─────────────┤       ├─────────────┤       ├─────────────┤
│ machine_id  │◄──────│ machine_id  │       │ project_slug│◄──┐
│ (PK)        │  1:N  │ project_slug│──────►│ (PK)        │   │
│             │       │ session_id  │  N:1  │             │   │
│             │       │ (UQ)        │       │             │   │
└─────────────┘       │transcript_  │       └─────────────┘   │
                      │  path       │                         │
                      │local_trace_ │                         │
                      │  path       │                         │
                      │ status      │                         │
                      └──────┬──────┘                         │
                             │                                │
                             │ session_id                     │
                             ▼                                │
                      ┌──────────────┐                        │
                      │issue_claims  │                        │
                      ├──────────────┤                        │
                      │ session_id   │────────────────────────┘
                      │ repo_full_name│   (通过 project_slug 关联)
                      │ issue_number │
                      │ status       │
                      └──────────────┘
```

---

## 1. machines 表

**用途**：记录使用 claude-tap-plus 的机器信息，`machine_id` 格式为 `whoami@hostname`。

```sql
CREATE TABLE machines (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    machine_id      TEXT NOT NULL UNIQUE,           -- whoami@hostname
    os              TEXT NOT NULL,                  -- windows/linux/macos
    hostname        TEXT NOT NULL,                  -- hostname 命令
    username        TEXT NOT NULL,                  -- whoami 命令
    first_seen_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at    DATETIME                        -- 每次 SessionStart 更新
);

CREATE INDEX idx_machines_hostname ON machines(hostname);
```

### 字段来源

| 字段 | 来源 | 获取方式 |
|------|------|----------|
| machine_id | 系统命令 | `whoami` + `@` + `hostname` |
| os | platform.sh | `uname -s` 解析为 OS_TYPE |
| hostname | 从 machine_id 解析 | 按 `@` 分割取后半部分 |
| username | 从 machine_id 解析 | 按 `@` 分割取前半部分 |

### 写入时机

| 时机 | 操作 |
|------|------|
| SessionStart (POST /api/session/register) | INSERT OR IGNORE + UPDATE last_seen_at |

---

## 2. projects 表

**用途**：记录 Claude Code 工作过的项目，`project_slug` 从 transcript_path 解析得到。

```sql
CREATE TABLE projects (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    project_slug    TEXT NOT NULL UNIQUE,           -- 从 transcript_path 解析
    project_cwd     TEXT NOT NULL,                  -- 首次出现的 cwd
    first_seen_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at    DATETIME                        -- 每次该项目有新会话时更新
);

CREATE INDEX idx_projects_slug ON projects(project_slug);
```

### 字段来源

| 字段 | 来源 | 获取方式 |
|------|------|----------|
| project_slug | transcript_path | `sed` 从路径中提取（如 `D--CodeDevelopment-CodeProject-claude-hk`） |
| project_cwd | SessionStart | `json_get '.cwd'` |

### 写入时机

| 时机 | 操作 |
|------|------|
| SessionStart (POST /api/session/register) | INSERT OR IGNORE + UPDATE last_seen_at |

---

## 3. sessions 表

**用途**：记录每次 Claude Code 会话的元数据。后端只存元数据，消息内容存储在本地 JSONL trace 文件中。

```sql
CREATE TABLE sessions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id      TEXT NOT NULL UNIQUE,           -- UUID，SessionStart 传入
    machine_id      TEXT NOT NULL,                  -- whoami@hostname
    os              TEXT NOT NULL,                  -- windows/linux/macos
    project_slug    TEXT NOT NULL,                  -- 从 transcript_path 解析
    project_cwd     TEXT NOT NULL,                  -- SessionStart 的 cwd 字段
    transcript_path  TEXT NOT NULL,                  -- Claude Code 原生 transcript 路径（SessionStart hook 提供）
    local_trace_path TEXT,                           -- proxy 本地 trace 文件路径（代理写入时构造）
    model           TEXT,                           -- SessionStart 的 model 字段
    source          TEXT,                           -- startup/resume
    status          TEXT NOT NULL DEFAULT 'active', -- active / closed
    registered_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at       DATETIME,                       -- SessionEnd 时更新
    close_reason    TEXT                            -- SessionEnd 的 reason 字段
);

CREATE INDEX idx_sessions_machine ON sessions(machine_id);
CREATE INDEX idx_sessions_project ON sessions(project_slug);
CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_sessions_registered ON sessions(registered_at);
```

### 字段来源

| 字段 | 来源 | 获取方式 |
|------|------|----------|
| session_id | SessionStart/End | `json_get '.session_id'` |
| machine_id | 系统命令 | `whoami` + `@` + `hostname` |
| os | platform.sh | `uname -s` 已解析为 OS_TYPE |
| project_slug | transcript_path | `sed` 从路径中提取 |
| project_cwd | SessionStart | `json_get '.cwd'` |
| transcript_path | SessionStart | `json_get '.transcript_path'` |
| local_trace_path | proxy 写入时构造 | `.claude-tap-plus/.traces/{machine_id}/{project_slug}/{session_id}.jsonl` |
| model | SessionStart | `json_get '.model'` |
| source | SessionStart | `json_get '.source'` |
| close_reason | SessionEnd | `json_get '.reason'` |

### 状态枚举

| 状态 | 说明 |
|------|------|
| `active` | 活跃会话 |
| `closed` | 已关闭 |

### 写入时机

| 时机 | 操作 |
|------|------|
| SessionStart (POST /api/session/register) | INSERT（含 transcript_path） |
| proxy 首次拦截到该 session 的 API 调用 | UPDATE local_trace_path（构造路径） |
| SessionEnd (POST /api/session/close) | UPDATE status='closed', closed_at, close_reason |

### 超时清理

```sql
-- 将超过 24 小时仍为 active 的会话标记为 closed
UPDATE sessions
SET status = 'closed',
    closed_at = datetime('now'),
    close_reason = 'timeout_cleanup'
WHERE status = 'active'
  AND registered_at < datetime('now', '-24 hours');
```

---

## 4. issue_claims 表

**用途**：存储 Issue 领取关系和状态流转。只存"哪个 session 领取了哪个 issue"及其状态，不存 issue 完整内容（内容在 GitHub 上）。

```sql
CREATE TABLE issue_claims (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_full_name  TEXT NOT NULL,                  -- xiaoheiDTF/claude-hk
    issue_number    INTEGER NOT NULL,               -- GitHub issue 编号
    issue_title     TEXT,                           -- issue 标题（缓存，方便展示）
    status          TEXT NOT NULL DEFAULT 'idle',   -- 见状态枚举
    session_id      TEXT,                           -- 领取者的 session_id，idle 时为 null
    claimed_at      DATETIME,                       -- 领取时间
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(repo_full_name, issue_number)
);

CREATE INDEX idx_issue_claims_repo ON issue_claims(repo_full_name);
CREATE INDEX idx_issue_claims_session ON issue_claims(session_id);
CREATE INDEX idx_issue_claims_status ON issue_claims(status);
```

### 字段来源

| 字段 | 来源 | 获取方式 |
|------|------|----------|
| repo_full_name | gh repo view | `gh repo view --json nameWithOwner --jq '.nameWithOwner'` |
| issue_number | gh issue list/view | `.number` |
| issue_title | gh issue view | `.title` |
| status | 后端维护 | 根据 API 调用更新 |
| session_id | 请求体传入 | API 调用方提供 |
| claimed_at | 后端生成 | 领取时 CURRENT_TIMESTAMP |

### 状态枚举与流转

```
            ┌─────────┐
 ┌─────────►│  idle   │◄──────────────────┐
 │          └────┬────┘                    │
 │               │ claim                   │ release / session close
 │               ▼                         │
 │          ┌─────────┐                     │
 │    ┌────►│ claimed │◄──┐                 │
 │    │     └────┬────┘   │                 │
 │    │          │ fix    │                 │
 │    │          ▼        │                 │
 │    │     ┌─────────┐◄──┤ reject          │
 │    │     │ fixing  │   │ (from reviewing)│
 │    │     └────┬────┘   │                 │
 │    │          │ done   │                 │
 │    │          ▼        │                 │
 │    │    ┌─────────────┐│                 │
 │    └────│ ready-for-pr││                 │
 │         └──────┬──────┘│                 │
 │                │ pr    │                 │
 │                ▼       │                 │
 │           ┌──────────┐ │                 │
 │           │pr-created│ │                 │
 │           └─────┬────┘ │                 │
 │                 │ test │                 │
 │                 ▼      │                 │
 │            ┌─────────┐ │                 │
 │            │ testing │ │                 │
 │            └────┬────┘ │                 │
 │                 │ review                  │
 │                 ▼      │                 │
 │           ┌───────────┐│                 │
 │           │ reviewing ├┘                 │
 │           └─────┬─────┘                  │
 │                 │ merge                   │
 │                 ▼                         │
 │            ┌─────────┐                    │
 │            │ merged  │────────────────────┘
 │            └─────────┘
 │
 └─ 注：rejected 状态在 SessionEnd 时自动释放为 idle（可重新领取）
```

| 状态 | 说明 | 对应 GitHub 操作 |
|------|------|-----------------|
| `idle` | 空闲，无人领取 | — |
| `claimed` | 已被某 session 领取 | gh issue edit --add-label "in-progress" |
| `fixing` | 正在开发中 | gh issue edit --add-label "fixing" |
| `ready-for-pr` | 开发完成，等待提 PR | gh issue edit --add-label "ready-for-pr" --remove-label "in-progress" |
| `pr-created` | PR 已创建 | gh issue edit --add-label "pr-created" |
| `testing` | 测试中 | gh issue edit --add-label "testing" |
| `reviewing` | 审核中 | gh issue edit --add-label "reviewing" |
| `merged` | 已合并（终态） | gh issue close |
| `rejected` | 被打回 | gh issue edit --add-label "rejected" |

### 写入时机

| 时机 | 操作 | 调用方 |
|------|------|--------|
| 首次 check/claim | INSERT OR IGNORE (idle) | 后端自动 |
| claim 成功 | UPDATE status='claimed', session_id=xxx | 001-4-issue-claim |
| 创建分支后 | UPDATE status='fixing' | 001-5-issue-fix |
| 开发完成后 | UPDATE status='ready-for-pr' | 001-6-issue-done |
| PR 创建后 | UPDATE status='pr-created' | 001-7-issue-pr |
| 开始测试时 | UPDATE status='testing' | 001-8-issue-test |
| 开始审核时 | UPDATE status='reviewing' | 001-9-issue-review |
| 合并后 | UPDATE status='merged' | 001-9-issue-review |
| 打回后 | UPDATE status='rejected' | 001-9-issue-review |
| 手动释放 | UPDATE status='idle', session_id=NULL | 异常放弃时 |
| SessionEnd | UPDATE status='idle', session_id=NULL（排除 merged） | SessionEnd hook |

---

## 完整建表脚本

```sql
-- claude-tap-plus 后端数据库建表脚本
-- 数据库类型：SQLite

-- ========== 会话管理域 ==========

-- 1. 机器表
CREATE TABLE IF NOT EXISTS machines (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    machine_id      TEXT NOT NULL UNIQUE,
    os              TEXT NOT NULL,
    hostname        TEXT NOT NULL,
    username        TEXT NOT NULL,
    first_seen_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at    DATETIME
);

CREATE INDEX IF NOT EXISTS idx_machines_hostname ON machines(hostname);

-- 2. 项目表
CREATE TABLE IF NOT EXISTS projects (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    project_slug    TEXT NOT NULL UNIQUE,
    project_cwd     TEXT NOT NULL,
    first_seen_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at    DATETIME
);

CREATE INDEX IF NOT EXISTS idx_projects_slug ON projects(project_slug);

-- 3. 会话表
CREATE TABLE IF NOT EXISTS sessions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id      TEXT NOT NULL UNIQUE,
    machine_id      TEXT NOT NULL,
    os              TEXT NOT NULL,
    project_slug    TEXT NOT NULL,
    project_cwd     TEXT NOT NULL,
    transcript_path  TEXT NOT NULL,                  -- Claude Code 原生 transcript 路径
    local_trace_path TEXT,                           -- proxy 本地 trace 文件路径
    model           TEXT,
    source          TEXT,
    status          TEXT NOT NULL DEFAULT 'active',
    registered_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at       DATETIME,
    close_reason    TEXT
);

CREATE INDEX IF NOT EXISTS idx_sessions_machine ON sessions(machine_id);
CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project_slug);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_registered ON sessions(registered_at);

-- ========== Issue 管理域 ==========

-- 4. Issue 领取表
CREATE TABLE IF NOT EXISTS issue_claims (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_full_name  TEXT NOT NULL,
    issue_number    INTEGER NOT NULL,
    issue_title     TEXT,
    status          TEXT NOT NULL DEFAULT 'idle',
    session_id      TEXT,
    claimed_at      DATETIME,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(repo_full_name, issue_number)
);

CREATE INDEX IF NOT EXISTS idx_issue_claims_repo ON issue_claims(repo_full_name);
CREATE INDEX IF NOT EXISTS idx_issue_claims_session ON issue_claims(session_id);
CREATE INDEX IF NOT EXISTS idx_issue_claims_status ON issue_claims(status);
```

---

## 核心查询汇总

### 会话管理查询

```sql
-- 查询某机器的所有会话
SELECT session_id, project_slug, status, registered_at, closed_at
FROM sessions WHERE machine_id = 'Administrator@DESKTOP-ABC123'
ORDER BY registered_at DESC;

-- 查询某项目的所有会话
SELECT session_id, machine_id, status, registered_at, closed_at
FROM sessions WHERE project_slug = 'D--CodeDevelopment-CodeProject-claude-hk'
ORDER BY registered_at DESC;

-- 查询会话详情（含关联 machine、project 信息和本地 trace 路径）
SELECT s.session_id, s.machine_id, s.project_slug, s.status,
       s.transcript_path, s.local_trace_path,
       s.registered_at, s.closed_at,
       m.hostname, m.os, m.username, p.project_cwd
FROM sessions s
JOIN machines m ON s.machine_id = m.machine_id
JOIN projects p ON s.project_slug = p.project_slug
WHERE s.session_id = 'bf15cac4-7235-48ce-8853-5d4598547f31';

-- 统计某项目的会话数
SELECT
    COUNT(*) as session_count,
    COUNT(CASE WHEN status = 'active' THEN 1 END) as active_count,
    COUNT(CASE WHEN status = 'closed' THEN 1 END) as closed_count
FROM sessions WHERE project_slug = 'D--CodeDevelopment-CodeProject-claude-hk';

-- 会话列表（支持过滤，对应 GET /api/sessions）
SELECT session_id, machine_id, project_slug, status, registered_at, closed_at
FROM sessions
WHERE (? IS NULL OR machine_id = ?)
  AND (? IS NULL OR project_slug = ?)
  AND (? IS NULL OR status = ?)
ORDER BY registered_at DESC;

-- 列出所有机器及其项目
SELECT m.machine_id, m.os, m.hostname,
    COUNT(DISTINCT s.project_slug) as project_count,
    COUNT(s.session_id) as total_sessions
FROM machines m
LEFT JOIN sessions s ON m.machine_id = s.machine_id
GROUP BY m.machine_id;

-- 超时清理：将超过 24 小时仍为 active 的会话标记为 closed
UPDATE sessions
SET status = 'closed', closed_at = datetime('now'), close_reason = 'timeout_cleanup'
WHERE status = 'active' AND registered_at < datetime('now', '-24 hours');
```

### Issue 管理查询

```sql
-- 查询某仓库所有 issue 状态
SELECT issue_number, issue_title, status, session_id, claimed_at
FROM issue_claims WHERE repo_full_name = 'xiaoheiDTF/claude-hk'
ORDER BY issue_number;

-- 查询某 session 领取的所有 issue
SELECT issue_number, repo_full_name, status, claimed_at
FROM issue_claims
WHERE session_id = 'bf15cac4-7235-48ce-8853-5d4598547f31'
  AND status != 'idle';

-- 批量查询指定 issue 状态（/api/issue/check 用）
SELECT issue_number, status, session_id, claimed_at
FROM issue_claims
WHERE repo_full_name = 'xiaoheiDTF/claude-hk'
  AND issue_number IN (9, 10, 11, 12);

-- 原子领取（/api/issue/claim 用）
SELECT status, session_id FROM issue_claims
WHERE repo_full_name = 'xiaoheiDTF/claude-hk' AND issue_number = 10;
-- 如果是 idle，则：
UPDATE issue_claims
SET status = 'claimed', session_id = 'session_xxx', claimed_at = CURRENT_TIMESTAMP
WHERE repo_full_name = 'xiaoheiDTF/claude-hk' AND issue_number = 10 AND status = 'idle';

-- 释放某 session 的所有 issue（/api/issue/release-session 用）
UPDATE issue_claims
SET status = 'idle', session_id = NULL, claimed_at = NULL
WHERE session_id = 'bf15cac4-7235-48ce-8853-5d4598547f31'
  AND status NOT IN ('merged');

-- 统计某仓库各状态 issue 数量
SELECT status, COUNT(*) as count
FROM issue_claims WHERE repo_full_name = 'xiaoheiDTF/claude-hk'
GROUP BY status;
```

---

## 索引汇总

共 9 个索引：

| 表 | 索引名 | 字段 | 用途 |
|----|--------|------|------|
| machines | `idx_machines_hostname` | hostname | 按主机名查询 |
| projects | `idx_projects_slug` | project_slug | 项目查找（已由 UNIQUE 隐含） |
| sessions | `idx_sessions_machine` | machine_id | 按机器查询会话 |
| sessions | `idx_sessions_project` | project_slug | 按项目查询会话 |
| sessions | `idx_sessions_status` | status | 按状态过滤 |
| sessions | `idx_sessions_registered` | registered_at | 按时间排序 |
| issue_claims | `idx_issue_claims_repo` | repo_full_name | 按仓库查询 issue |
| issue_claims | `idx_issue_claims_session` | session_id | 按 session 查询领取的 issue |
| issue_claims | `idx_issue_claims_status` | status | 按状态过滤 |
