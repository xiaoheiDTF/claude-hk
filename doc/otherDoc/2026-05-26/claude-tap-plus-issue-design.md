# claude-tap-plus Issue 全局管理设计

> 创建时间：2026-05-26
> 模块：claude-tap-plus / 模块 B
> 简述：为单 GitHub 账号多 Agent 场景设计 Issue 领取去重与状态流转机制，通过后端中心化服务解决并发冲突

---

## 一、设计原则

1. **GitHub 是数据源，后端是锁**：GitHub 上的 issue 状态是权威数据源，后端只负责记录"哪个 session 正在处理哪个 issue"，防止多 Agent 同时领取同一 issue
2. ** session 绑定，非用户绑定**：用 session_id 标识领取者（而非 GitHub 账号），支持同一账号下多 Agent 并行
3. **状态流转由技能触发**：后端不主动感知 GitHub 状态变化，只响应技能发来的状态变更请求
4. **会话关闭自动释放**：session 关闭（SessionEnd hook）时，后端自动释放该 session 领取的所有 issue

---

## 二、问题背景

### 2.1 当前问题

- 只有一个 GitHub 账号，多个 Agent（不同机器/不同会话）共用
- `003-4-issue-claim` 用 `gh issue edit --add-assignee @me` 领取，但所有 Agent 都是同一个 "me"
- 导致多个 Agent 可能同时领取同一个 issue，产生冲突

### 2.2 解决思路

```
Before（有冲突）：
  Agent A ──→ gh issue edit #10 --add-assignee @me  ✓ 领取成功
  Agent B ──→ gh issue edit #10 --add-assignee @me  ✓ 也领取成功（同一账号）
  → 两个 Agent 同时处理 #10，冲突！

After（后端去重）：
  Agent A ──→ 后端查询 #10 状态 → 空闲 → 标记为 Agent A 的 session_1 领取 → gh assign
  Agent B ──→ 后端查询 #10 状态 → 已被 session_1 领取 → 过滤掉，不显示
  → Agent B 只能看到其他空闲 issue
```

---

## 三、可获取的数据源

### 3.1 Issue Claim 技能能提供的字段

```bash
# gh issue view 能拿到的数据
gh issue view <编号> --json number,title,labels,assignees,state,url
```

```json
{
  "number": 10,
  "title": "优化 issue 模板",
  "labels": [{"name": "enhancement"}],
  "assignees": [{"login": "xiaoheiDTF"}],
  "state": "OPEN",
  "url": "https://github.com/xiaoheiDTF/claude-hk/issues/10"
}
```

| 字段 | 说明 | 用途 |
|------|------|------|
| `number` | Issue 编号 | 后端存储的主键 |
| `title` | Issue 标题 | 展示用 |
| `labels` | 标签列表 | 判断类型（bug/enhancement） |
| `assignees` | 指派人 | GitHub 层面的 assignee（同一账号） |
| `state` | OPEN/CLOSED | 是否已关闭 |
| `url` | 完整 URL | 唯一标识项目 + issue |

### 3.2 项目信息

```bash
# 获取当前项目的 GitHub 信息
gh repo view --json url,nameWithOwner
# → {"url":"https://github.com/xiaoheiDTF/claude-hk","nameWithOwner":"xiaoheiDTF/claude-hk"}
```

| 字段 | 说明 | 用途 |
|------|------|------|
| `repo_owner` | 仓库所有者 | xiaoheiDTF |
| `repo_name` | 仓库名 | claude-hk |
| `repo_full_name` | 完整名 | xiaoheiDTF/claude-hk |

### 3.3 Session 信息

从 SessionStart 或当前环境获取：

| 字段 | 来源 | 说明 |
|------|------|------|
| `session_id` | SessionStart | 当前会话唯一标识 |
| `machine_id` | whoami@hostname | 当前机器 |

---

## 四、Issue 状态定义

### 4.1 后端内部状态（与 GitHub label 解耦）

| 状态 | 说明 | 对应 GitHub 操作 |
|------|------|-----------------|
| `idle` | 空闲，无人领取 | — |
| `claimed` | 已被某 session 领取 | gh issue edit --add-label "in-progress" |
| `fixing` | 正在开发中 | gh issue edit --add-label "fixing" |
| `ready-for-pr` | 开发完成，等待提 PR | gh issue edit --add-label "ready-for-pr" --remove-label "in-progress" |
| `pr-created` | PR 已创建 | gh issue edit --add-label "pr-created" |
| `testing` | 测试中 | gh issue edit --add-label "testing" |
| `reviewing` | 审核中 | gh issue edit --add-label "reviewing" |
| `merged` | 已合并 | gh issue close |
| `rejected` | 被打回 | gh issue edit --add-label "rejected" |

### 4.2 状态流转图

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
---

## 五、实现方案

### 5.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        GitHub                                │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐        │
│  │ issue#9 │  │issue#10 │  │issue#11 │  │issue#12 │        │
│  │  OPEN   │  │  OPEN   │  │ CLOSED  │  │  OPEN   │        │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘        │
│       └─────────────┴─────────────┴─────────────┘           │
│                      GitHub API                              │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       │ gh issue list / view / edit
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                     Agent (Claude Code)                      │
│                                                              │
│  003-4-issue-claim ──→ 调用后端 API 去重/领取               │
│  003-5-issue-fix   ──→ 调用后端 API 标记 fixing             │
│  003-6-issue-done  ──→ 调用后端 API 标记 ready-for-pr       │
│  003-7-issue-pr    ──→ 调用后端 API 标记 pr-created         │
│  003-8-issue-test  ──→ 调用后端 API 标记 testing            │
│  003-9-issue-review ──→ 调用后端 API 标记 merged/rejected   │
│                                                              │
│  SessionEnd hook   ──→ 调用后端 API 释放该 session 的 issue │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTP
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              claude-tap-plus 后端服务器                       │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Issue 状态表 (issue_claims)              │   │
│  │  issue_id │ repo_full_name │ status │ session_id     │   │
│  │     10    │ xiaoheiDTF/... │claimed │ session_abc    │   │
│  │     12    │ xiaoheiDTF/... │ idle   │ null           │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  API:                                                        │
│  POST /api/issue/check   → 批量查询 issue 状态               │
│  POST /api/issue/claim   → 领取 issue                        │
│  POST /api/issue/release → 释放 issue                        │
│  POST /api/issue/status  → 更新 issue 状态                   │
│  POST /api/issue/release-session → SessionEnd 释放所有       │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 B1: Issue 状态查询

**API**: `POST /api/issue/check`

**请求**：
```json
{
  "repo_full_name": "xiaoheiDTF/claude-hk",
  "issue_numbers": [9, 10, 11, 12]
}
```

**响应**：
```json
{
  "issues": [
    {"number": 9,  "status": "idle",    "session_id": null,         "claimed_at": null},
    {"number": 10, "status": "claimed", "session_id": "session_abc", "claimed_at": "2026-05-26T10:00:00Z"},
    {"number": 11, "status": "merged",  "session_id": null,         "claimed_at": null},
    {"number": 12, "status": "idle",    "session_id": null,         "claimed_at": null}
  ]
}
```

**003-4-issue-claim 脚本中的使用**：

```bash
# 1. 从 gh 获取 open issues 列表
issues_json=$(gh issue list --state open --json number,title,labels)

# 2. 提取 issue 编号列表
issue_numbers=$(echo "$issues_json" | jq -r '.[].number' | jq -R -s -c 'split("\n")[:-1] | map(tonumber)')

# 3. 调用后端去重
repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner')
backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)

available_issues=$(curl -s -X POST "$backend_url/api/issue/check" \
  -H "Content-Type: application/json" \
  -d "{\"repo_full_name\":\"$repo\",\"issue_numbers\":$issue_numbers}")

# 4. 过滤出 status=idle 的 issue
idle_issues=$(echo "$available_issues" | jq '.issues[] | select(.status == "idle")')
```

### 5.3 B2: Issue 批量去重

与 B1 是同一个 API，在脚本中组合使用：

```bash
# 完整去重流程（在 003-4-issue-claim 脚本中）

# Step 1: 获取 GitHub 上的 open issues
gh_issues=$(gh issue list --state open --json number,title,labels)

# Step 2: 提取编号
numbers=$(echo "$gh_issues" | jq '[.[].number]')

# Step 3: 调后端查询哪些已被领取
repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner')
backend_url=... # 从 backend.conf 读取

check_result=$(curl -s -X POST "$backend_url/api/issue/check" \
  -H "Content-Type: application/json" \
  -d "{\"repo_full_name\":\"$repo\",\"issue_numbers\":$numbers}")

# Step 4: 过滤出 idle 的
idle_numbers=$(echo "$check_result" | jq '[.issues[] | select(.status == "idle") | .number]')

# Step 5: 从 gh_issues 中只保留 idle 的
echo "$gh_issues" | jq --argjson idle "$idle_numbers" '[.[] | select(.number | IN($idle[]))]'
```

### 5.4 B3: Issue 领取

**API**: `POST /api/issue/claim`

**请求**：
```json
{
  "repo_full_name": "xiaoheiDTF/claude-hk",
  "issue_number": 10,
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "issue_title": "优化 issue 模板"
}
```

**响应（成功）**：
```json
{"success": true, "status": "claimed", "claimed_at": "2026-05-26T10:05:00Z"}
```

**响应（失败，已被领取）**：
```json
{"success": false, "error": "already_claimed", "claimed_by": "session_abc", "claimed_at": "2026-05-26T10:00:00Z"}
```

> **注**：`merged` 和 `rejected` 状态的 issue 也返回 `already_claimed` 错误；`rejected` 状态的 issue 被 SessionEnd 释放后可重新领取；`issue_title` 为非必填字段。

**003-4-issue-claim 脚本中的使用**：

```bash
# 用户确认领取 #10 后
session_id=$(json_get '.session_id')  # 从 hook 输入获取
repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner')
issue_title=$(gh issue view 10 --json title --jq '.title')

# 先调后端领取（原子操作）
claim_result=$(curl -s -X POST "$backend_url/api/issue/claim" \
  -H "Content-Type: application/json" \
  -d "{
    \"repo_full_name\":\"$repo\",
    \"issue_number\":10,
    \"session_id\":\"$session_id\",
    \"issue_title\":\"$issue_title\"
  }")

# 检查后端是否成功
if echo "$claim_result" | jq -e '.success' > /dev/null; then
  # 后端领取成功，再操作 GitHub
  gh issue edit 10 --add-assignee @me --add-label "in-progress"
  log "INFO" "Issue #10 claimed"
else
  # 后端领取失败（已被其他 session 领取）
  claimed_by=$(echo "$claim_result" | jq -r '.claimed_by')
  echo "Issue #10 已被 $claimed_by 领取"
fi
```

### 5.5 B4: Issue 状态流转

**API**: `POST /api/issue/status`

**请求**：
```json
{
  "repo_full_name": "xiaoheiDTF/claude-hk",
  "issue_number": 10,
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "status": "fixing"
}
```

**响应（成功）**：
```json
{"success": true, "previous_status": "claimed", "status": "fixing", "updated_at": "2026-05-26T10:30:00Z"}
```

**响应（失败，非 owner）**：
```json
{"success": false, "error": "not_owner", "message": "Only the claimant can update status"}
```

**响应（失败，非法状态流转）**：
```json
{"success": false, "error": "invalid_transition", "message": "Cannot transition from merged to fixing"}
```

**各技能调用时机**：

| 技能 | 调用时机 | status 值 | GitHub 操作 | 自动同步 label |
|------|----------|-----------|------------|----------------|
| 003-5-issue-fix | 创建分支后 | `fixing` | gh issue comment | gh issue edit --add-label "fixing" |
| 003-6-issue-done | 开发完成后 | `ready-for-pr` | gh issue edit --remove-label "in-progress" --add-label "ready-for-pr" | —（已有手动操作） |
| 003-7-issue-pr | PR 创建后 | `pr-created` | — | gh issue edit --add-label "pr-created" |
| 003-8-issue-test | 开始测试 | `testing` | — | gh issue edit --add-label "testing" |
| 003-9-issue-review（开始审核时） | 审核开始 | `reviewing` | — | gh issue edit --add-label "reviewing" |
| 003-9-issue-review merge | 合并后 | `merged` | gh issue close | —（终态） |
| 003-9-issue-review reject | 打回后 | `rejected` | gh issue edit --add-label "rejected" | —（已有手动操作）；状态回到 `fixing` |

**003-5-issue-fix 脚本示例**：

```bash
# 创建分支后，向后端更新状态
session_id=$(json_get '.session_id')
repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner')

curl -s -X POST "$backend_url/api/issue/status" \
  -H "Content-Type: application/json" \
  -d "{
    \"repo_full_name\":\"$repo\",
    \"issue_number\":$ISSUE_NUM,
    \"session_id\":\"$session_id\",
    \"status\":\"fixing\"
  }" > /dev/null 2>&1
```

### 5.6 B5: Issue 释放

**场景 1：SessionEnd 自动释放**

**API**: `POST /api/issue/release-session`

**请求**：
```json
{"session_id": "bf15cac4-7235-48ce-8853-5d4598547f31"}
```

**响应**：
```json
{"released": [10, 12], "count": 2}
```

**在 SessionEnd hook 中调用**：

```bash
# .claude/hooks/29-session-end/base.sh 末尾

release_session_issues() {
  local session_id=$(json_get '.session_id')
  local backend_url
  backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)
  [ -z "$backend_url" ] && return 0

  local result
  result=$(curl -s --max-time 5 -X POST "$backend_url/api/issue/release-session" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$session_id\"}")

  local count
  count=$(echo "$result" | jq -r '.count // 0')
  [ "$count" -gt 0 ] && log "INFO" "Released $count issues for session $session_id"
}

release_session_issues
```

**场景 2：手动释放（issue 完成或放弃时）**

**API**: `POST /api/issue/release`

**请求**：
```json
{
  "repo_full_name": "xiaoheiDTF/claude-hk",
  "issue_number": 10,
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31"
}
```

---

## 六、后端 API 清单

| 接口 | 方法 | 说明 | 调用方 |
|------|------|------|--------|
| `/api/issue/check` | POST | 批量查询 issue 状态（idle/claimed/merged 等） | 003-4-issue-claim |
| `/api/issue/claim` | POST | 领取 issue，绑定 session_id | 003-4-issue-claim |
| `/api/issue/status` | POST | 更新 issue 状态（fixing/pr-created/testing 等） | 003-5 至 003-9 |
| `/api/issue/release` | POST | 手动释放单个 issue | 异常放弃时 |
| `/api/issue/release-session` | POST | SessionEnd 时释放该 session 所有 issue | SessionEnd hook |
| `/health` | GET | 健康检查，返回 `{"status":"ok"}` | `_backend_available()` |

---

## 七、需要的依赖和前置条件

### 7.1 已有的

- `curl` — hooks 和脚本中已可用
- `gh` — GitHub CLI，已有
- `jq` — JSON 处理，已有
- `json_get()` — hooks/base.sh 已提供

### 7.2 需要新增的

| 项目 | 说明 |
|------|------|
| `.claude/backend.conf` | 后端地址配置（与模块 C 共用） |
| 后端 `/api/issue/*` 接口 | 5 个 API，见第六节 |
| 各 Issue 技能脚本改造 | 在现有 gh 操作前后增加后端调用 |

### 7.3 改造清单

| 技能 | 改造内容 |
|------|----------|
| 003-4-issue-claim | 获取 issue 列表后调 `/api/issue/check` 去重；用户确认后调 `/api/issue/claim` 原子领取 |
| 003-5-issue-fix | 创建分支后调 `/api/issue/status` 标记 `fixing` |
| 003-6-issue-done | 标记完成后调 `/api/issue/status` 标记 `ready-for-pr` |
| 003-7-issue-pr | PR 创建后调 `/api/issue/status` 标记 `pr-created` |
| 003-8-issue-test | 开始测试时调 `/api/issue/status` 标记 `testing` |
| 003-9-issue-review | merge 后调 `/api/issue/status` 标记 `merged`；reject 后标记 `rejected` |
| SessionEnd hook | 调 `/api/issue/release-session` 自动释放 |

---

## 八、数据流总结

```
用户执行 /003-4-issue-claim
  │
  ├─ 1. gh issue list → 获取 GitHub 上的 open issues
  ├─ 2. POST /api/issue/check → 后端过滤掉已被领取的
  ├─ 3. 展示空闲 issue 列表给用户
  ├─ 4. 用户选择 #10
  ├─ 5. POST /api/issue/claim → 后端原子标记为已领取
  ├─ 6. gh issue edit #10 --add-assignee @me --add-label "in-progress"
  │
  └─ 后续状态流转：
       ├─ /003-5-issue-fix → POST /api/issue/status (fixing)
       ├─ /003-6-issue-done → POST /api/issue/status (ready-for-pr)
       ├─ /003-7-issue-pr → POST /api/issue/status (pr-created)
       ├─ /003-8-issue-test → POST /api/issue/status (testing)
       └─ /003-9-issue-review merge → POST /api/issue/status (merged)

SessionEnd hook 触发
  │
  └─ POST /api/issue/release-session → 自动释放该 session 所有 issue
```

---

## 九、冲突处理示例

### 场景：两个 Agent 同时想领取 #10

```
时间线:

T1: Agent A ──→ POST /api/issue/claim (#10, session_A)
                后端: #10 状态 idle → 更新为 claimed, session=session_A
                响应: {"success": true}

T2: Agent A ──→ gh issue edit #10 --add-assignee @me
                GitHub: #10 assignee = xiaoheiDTF

T3: Agent B ──→ POST /api/issue/claim (#10, session_B)
                后端: #10 状态 claimed, session=session_A ≠ session_B
                响应: {"success": false, "error": "already_claimed", "claimed_by": "session_A"}

T4: Agent B ──→ 显示 "#10 已被 session_A 领取，请选择其他 issue"
```

### 场景：Agent A 崩溃，未执行 SessionEnd

```
Agent A 领取 #10 后崩溃，SessionEnd hook 未触发

后续处理方案:
1. 后端可设置 claim 超时（如 24 小时），超时自动释放
2. 或管理员手动调 POST /api/issue/release 释放
3. 或其他 Agent 可联系管理员确认后强制释放
```

---

## 十、数据库表设计

> **设计原则**：只存"哪个 session 领取了哪个 issue"，不存 issue 的完整内容（内容在 GitHub 上）。

### 10.1 Issue 领取表（issue_claims）

```sql
CREATE TABLE issue_claims (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_full_name  TEXT NOT NULL,                  -- xiaoheiDTF/claude-hk
    issue_number    INTEGER NOT NULL,               -- GitHub issue 编号
    issue_title     TEXT,                           -- issue 标题（缓存，方便展示）
    status          TEXT NOT NULL DEFAULT 'idle',   -- idle/claimed/fixing/ready-for-pr/pr-created/testing/reviewing/merged/rejected
    session_id      TEXT,                           -- 领取者的 session_id，idle 时为 null
    claimed_at      DATETIME,                       -- 领取时间
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(repo_full_name, issue_number)
);

-- 索引
CREATE INDEX idx_issue_claims_repo ON issue_claims(repo_full_name);
CREATE INDEX idx_issue_claims_session ON issue_claims(session_id);
CREATE INDEX idx_issue_claims_status ON issue_claims(status);
```

**字段来源说明**：

| 字段 | 来源 | 获取方式 |
|------|------|----------|
| repo_full_name | gh repo view | `gh repo view --json nameWithOwner --jq '.nameWithOwner'` |
| issue_number | gh issue list/view | `.number` |
| issue_title | gh issue view | `.title` |
| status | 后端维护 | 根据 API 调用更新 |
| session_id | SessionStart / API 请求 | `json_get '.session_id'` 或请求体传入 |
| claimed_at | 后端生成 | 领取时 `CURRENT_TIMESTAMP` |

### 10.2 ER 关系图

```
┌─────────────────┐         ┌─────────────────┐
│  sessions (C)   │         │  issue_claims   │
├─────────────────┤         ├─────────────────┤
│ session_id (PK) │◄────────│ session_id (FK) │
│ machine_id      │   1:N   │ repo_full_name  │
│ project_slug    │         │ issue_number    │
│ ...             │         │ status          │
└─────────────────┘         │ claimed_at      │
                            └─────────────────┘
```

> sessions 表在模块 C（session-design.md）中定义，issue_claims 通过 session_id 关联。

### 10.3 核心查询示例

```sql
-- 1. 查询某仓库所有 issue 状态
SELECT issue_number, issue_title, status, session_id, claimed_at
FROM issue_claims
WHERE repo_full_name = 'xiaoheiDTF/claude-hk'
ORDER BY issue_number;

-- 2. 查询某 session 领取的所有 issue
SELECT issue_number, repo_full_name, status, claimed_at
FROM issue_claims
WHERE session_id = 'bf15cac4-7235-48ce-8853-5d4598547f31'
  AND status != 'idle';

-- 3. 批量查询指定 issue 的状态（/api/issue/check 的实现）
SELECT issue_number, status, session_id, claimed_at
FROM issue_claims
WHERE repo_full_name = 'xiaoheiDTF/claude-hk'
  AND issue_number IN (9, 10, 11, 12);

-- 4. 释放某 session 的所有 issue（SessionEnd 时）
UPDATE issue_claims
SET status = 'idle', session_id = NULL, claimed_at = NULL
WHERE session_id = 'bf15cac4-7235-48ce-8853-5d4598547f31'
  AND status NOT IN ('merged');

-- 5. 原子领取（/api/issue/claim 的实现）
-- 先检查：
SELECT status, session_id FROM issue_claims
WHERE repo_full_name = 'xiaoheiDTF/claude-hk' AND issue_number = 10;
-- 如果是 idle，则 UPDATE：
UPDATE issue_claims
SET status = 'claimed', session_id = 'session_xxx', claimed_at = CURRENT_TIMESTAMP
WHERE repo_full_name = 'xiaoheiDTF/claude-hk' AND issue_number = 10 AND status = 'idle';

-- 6. 统计某仓库各状态 issue 数量
SELECT status, COUNT(*) as count
FROM issue_claims
WHERE repo_full_name = 'xiaoheiDTF/claude-hk'
GROUP BY status;
```

### 10.4 写入时序

| 时机 | 操作 | 说明 |
|------|------|------|
| 首次 check/claim | INSERT OR IGNORE | issue 首次出现时插入，状态为 idle |
| claim 成功 | UPDATE status='claimed', session_id=xxx | 原子操作，需检查当前为 idle |
| 状态流转 | UPDATE status=新状态 | fixing/ready-for-pr/pr-created/testing/reviewing/merged/rejected |
| release | UPDATE status='idle', session_id=NULL | 手动释放或 SessionEnd 自动释放 |
| merge | UPDATE status='merged' | 合并后保持 merged，不清除 |

---

## 十一、与现有技能的对应关系

| 现有技能 | 当前做法 | 改造后做法 |
|----------|----------|-----------|
| 003-4-issue-claim | `gh issue edit --add-assignee @me` | 先调后端 claim，成功后再 gh assign |
| 003-5-issue-fix | 直接创建分支 | claim 后调后端标记 fixing |
| 003-6-issue-done | `gh issue edit --remove-label "in-progress" --add-label "ready-for-pr"` | 先调后端更新状态，再操作 GitHub |
| 003-7-issue-pr | `gh pr create` | PR 创建后调后端标记 pr-created |
| 003-8-issue-test | 本地执行测试 | 开始测试时调后端标记 testing |
| 003-9-issue-review | `gh pr merge` / `gh issue edit --add-label "rejected"` | 调后端标记 merged/rejected，再操作 GitHub |
