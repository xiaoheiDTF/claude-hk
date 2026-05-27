# claude-tap-plus Issue 技能集成设计：模块 D

> 创建时间：2026-05-26
> 模块：claude-tap-plus / 模块 D
> 简述：改造现有 003-x-issue 系列技能，接入后端 Issue 全局管理服务，实现单 GitHub 账号多 Agent 协作

---

## 一、设计原则

1. **后端优先，GitHub 次之**：所有涉及 Issue 状态变更的操作，先调后端 API 确认/更新，再操作 GitHub
2. **最小侵入改造**：现有技能的 GitHub 操作流程不变，只在关键节点插入后端调用
3. **失败静默**：后端服务不可用时，技能降级为原有行为（直接操作 GitHub），不阻塞工作流
4. **配置驱动**：通过 `.claude/backend.conf` 控制是否启用后端集成，未配置时完全走本地逻辑

---

## 二、现有技能梳理

### 2.1 技能调用链

```
003-2-issue (创建) → 003-3-issue-discuss (讨论)
                           ↓
              003-4-issue-claim (领取) ←──┐
                           ↓              │
              003-5-issue-fix (开发)       │
                           ↓              │ 拒绝后重新领取
              003-6-issue-done (完成)      │
                           ↓              │
              003-7-issue-pr (提PR)        │
                           ↓              │
              003-8-issue-test (测试)      │
                           ↓              │
              003-9-issue-review (审核) ───┘
                           ↓
                     merged / rejected
```

### 2.2 各技能现状

| 技能 | 当前操作 | 需要后端介入的环节 |
|------|----------|-------------------|
| 003-2-issue | `gh issue create` | 不需要（创建新 issue，无冲突） |
| 003-3-issue-discuss | `gh issue view`, `gh issue comment` | 不需要（只读/评论，不改变状态） |
| 003-4-issue-claim | `gh issue list` → `gh issue edit --add-assignee` | **需要**：list 后去重，edit 前原子领取 |
| 003-5-issue-fix | `gh issue view` → `git checkout -b` → `gh issue comment` | **需要**：标记 fixing 状态 |
| 003-6-issue-done | `git status` → `gh issue edit --remove-label --add-label` | **需要**：标记 ready-for-pr 状态 |
| 003-7-issue-pr | `gh pr create` | **需要**：标记 pr-created 状态 |
| 003-8-issue-test | `gh pr view` → 测试 → `gh pr edit` | **需要**：标记 testing 状态 |
| 003-9-issue-review | `gh pr merge` / `gh issue edit` | **需要**：标记 merged/rejected 状态 |

---

## 三、可获取的数据源

### 3.1 各技能脚本中已有的数据

```bash
# 项目信息
repo_full_name=$(gh repo view --json nameWithOwner --jq '.nameWithOwner')
# → xiaoheiDTF/claude-hk

# Issue 信息
issue_number=$(echo "$PROMPT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')
# → 10

issue_title=$(gh issue view "$issue_number" --json title --jq '.title')
# → "优化 issue 模板"

# Session 信息（从 hook 环境或请求体）
session_id=$(json_get '.session_id')
# → bf15cac4-7235-48ce-8853-5d4598547f31

# 后端地址
backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)
```

### 3.2 各技能需要传入后端的数据

| 技能 | 需要的数据 |
|------|-----------|
| 003-4-issue-claim | repo_full_name, issue_number, issue_title, session_id |
| 003-5-issue-fix | repo_full_name, issue_number, session_id, status="fixing" |
| 003-6-issue-done | repo_full_name, issue_number, session_id, status="ready-for-pr" |
| 003-7-issue-pr | repo_full_name, issue_number, session_id, status="pr-created" |
| 003-8-issue-test | repo_full_name, issue_number, session_id, status="testing" |
| 003-9-issue-review | repo_full_name, issue_number, session_id, status="merged"/"rejected" |

---

## 四、后端状态定义（复用模块 B）

| 状态 | 说明 |
|------|------|
| `idle` | 空闲 |
| `claimed` | 已被领取 |
| `fixing` | 开发中 |
| `ready-for-pr` | 开发完成 |
| `pr-created` | PR 已创建 |
| `testing` | 测试中 |
| `reviewing` | 审核中 |
| `merged` | 已合并 |
| `rejected` | 被打回 |

---

## 五、实现方案

### 5.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                     Claude Code 会话                         │
│                                                              │
│  003-4-issue-claim ──┐                                       │
│  003-5-issue-fix     │                                       │
│  003-6-issue-done    ├──→ 各技能脚本中的后端调用函数          │
│  003-7-issue-pr      │      (_call_backend)                  │
│  003-8-issue-test    │                                       │
│  003-9-issue-review ─┘                                       │
│                          ↓                                   │
│                   .claude/backend.conf                        │
│                          ↓                                   │
│              ┌─────────────────────┐                         │
│              │ 后端不可用/未配置    │                         │
│              │ → 降级为原有行为     │                         │
│              └─────────────────────┘                         │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTP (curl)
                           ▼
┌─────────────────────────────────────────────────────────────┐
│              claude-tap-plus 后端服务器                       │
│                                                              │
│  POST /api/issue/check   → 查询 issue 状态                   │
│  POST /api/issue/claim   → 领取 issue                        │
│  POST /api/issue/status  → 更新状态                          │
│  POST /api/issue/release → 释放 issue                        │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 公共后端调用函数

在各技能的 `scripts/` 目录中新增一个公共辅助函数文件，或每个技能脚本内嵌相同逻辑：

```bash
# ---- 后端调用公共函数 ----
# 建议放在 .claude/skills/backend.sh，各技能 source

BACKEND_URL=""

_load_backend_url() {
  if [ -z "$BACKEND_URL" ]; then
    BACKEND_URL=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)
  fi
}

# 检查后端是否可用
_backend_available() {
  _load_backend_url
  [ -n "$BACKEND_URL" ] && curl -s --max-time 2 "$BACKEND_URL/health" > /dev/null 2>&1
}

# 调用后端 API，失败时静默返回空
_call_backend() {
  local endpoint="$1"
  local data="$2"
  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 1

  local resp
  resp=$(curl -s --max-time 5 -X POST "$BACKEND_URL$endpoint" \
    -H "Content-Type: application/json" \
    -d "$data" 2>/dev/null)
  [ -n "$resp" ] && echo "$resp"
}

# 获取当前 session_id
_get_session_id() {
  # 优先从环境变量获取
  if [ -n "$CLAUDE_SESSION_ID" ]; then
    echo "$CLAUDE_SESSION_ID"
    return
  fi
  # 从 SessionStart hook 的输入 JSON 提取（如果在 hook 中）
  if command -v json_get >/dev/null 2>&1; then
    json_get '.session_id' 2>/dev/null
  fi
}
```

> **注**：`CLAUDE_SESSION_ID` 环境变量需要在代理启动时注入，见模块 C 改造。

### 5.3 D1: 003-4-issue-claim 改造

**改造位置**：`.claude/skills/003-4-issue-claim/scripts/03UserPromptSubmit.sh`

**当前流程**：
1. `gh issue list` 获取 open issues
2. 展示给用户
3. 用户选择后 `gh issue edit --add-assignee @me --add-label "in-progress"`

**改造后流程**：
1. `gh issue list` 获取 open issues
2. **新增**：调后端 `/api/issue/check` 过滤已领取的
3. 展示过滤后的空闲 issues
4. 用户选择后
5. **新增**：调后端 `/api/issue/claim` 原子领取
6. 后端成功后再 `gh issue edit --add-assignee @me --add-label "in-progress"`
7. 后端失败则提示已被领取

**脚本改造代码**：

```bash
# 在 003-4-issue-claim 的 03UserPromptSubmit.sh 中

# ---- 去重过滤 ----
filter_claimed_issues() {
  local issues_json="$1"
  
  # 未配置后端，直接返回全部
  _load_backend_url
  [ -z "$BACKEND_URL" ] && { echo "$issues_json"; return; }
  
  # 提取 issue 编号列表
  local numbers
  numbers=$(echo "$issues_json" | jq '[.[].number]')
  
  # 获取 repo 信息
  local repo
  repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner' 2>/dev/null)
  [ -z "$repo" ] && { echo "$issues_json"; return; }
  
  # 调后端查询状态
  local check_result
  check_result=$(_call_backend "/api/issue/check" "{\"repo_full_name\":\"$repo\",\"issue_numbers\":$numbers}")
  [ -z "$check_result" ] && { echo "$issues_json"; return; }
  
  # 过滤出 idle 的 issue 编号
  local idle_numbers
  idle_numbers=$(echo "$check_result" | jq '[.issues[] | select(.status == "idle") | .number]')
  
  # 从原始列表中只保留 idle 的
  echo "$issues_json" | jq --argjson idle "$idle_numbers" '[.[] | select(.number | IN($idle[]))]'
}

# ---- 原子领取 ----
claim_issue_backend() {
  local issue_num="$1"
  local issue_title="$2"
  
  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 0  # 未配置后端，跳过
  
  local session_id
  session_id=$(_get_session_id)
  [ -z "$session_id" ] && return 0
  
  local repo
  repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner' 2>/dev/null)
  [ -z "$repo" ] && return 0
  
  # 调后端领取
  local result
  result=$(_call_backend "/api/issue/claim" "{
    \"repo_full_name\":\"$repo\",
    \"issue_number\":$issue_num,
    \"session_id\":\"$session_id\",
    \"issue_title\":\"$issue_title\"
  }")
  
  # 检查结果
  if ! echo "$result" | jq -e '.success' > /dev/null 2>&1; then
    local claimed_by
    claimed_by=$(echo "$result" | jq -r '.claimed_by // "unknown"')
    echo "ERROR: Issue #$issue_num 已被 $claimed_by 领取"
    return 1
  fi
  
  return 0
}

# ---- 主流程中的使用 ----

# 1. 获取 issues
issues=$(gh issue list --state open --json number,title,labels 2>/dev/null)

# 2. 过滤已领取的
filtered=$(filter_claimed_issues "$issues")

# 3. 展示给用户（只展示过滤后的）
# ... 现有展示逻辑，改用 $filtered ...

# 4. 用户确认领取 #N 后
if claim_issue_backend "$ISSUE_NUM" "$ISSUE_TITLE"; then
  # 后端领取成功，再操作 GitHub
  gh issue edit "$ISSUE_NUM" --add-assignee @me --add-label "in-progress"
  log "INFO" "Issue #$ISSUE_NUM claimed"
else
  # 后端领取失败，提示用户
  echo "Issue #$ISSUE_NUM 无法领取，请选择其他 issue"
fi
```

### 5.4 D2: 003-5-issue-fix 改造

**改造位置**：`.claude/skills/003-5-issue-fix/scripts/03UserPromptSubmit.sh`

**新增**：创建分支后向后端标记 `fixing` 状态

```bash
# 在创建分支后、comment 前

update_issue_status() {
  local issue_num="$1"
  local new_status="$2"
  
  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 0
  
  local session_id
  session_id=$(_get_session_id)
  [ -z "$session_id" ] && return 0
  
  local repo
  repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner' 2>/dev/null)
  [ -z "$repo" ] && return 0
  
  _call_backend "/api/issue/status" "{
    \"repo_full_name\":\"$repo\",
    \"issue_number\":$issue_num,
    \"session_id\":\"$session_id\",
    \"status\":\"$new_status\"
  }" > /dev/null 2>&1
}

# 创建分支后
update_issue_status "$ISSUE_NUM" "fixing"

# 然后继续原有逻辑：gh issue comment ...
```

### 5.5 D3: 003-6-issue-done 改造

**新增**：标记完成后向后端更新 `ready-for-pr`

```bash
# 在移除 in-progress label、添加 ready-for-pr label 后

update_issue_status "$ISSUE_NUM" "ready-for-pr"
```

### 5.6 D4: 003-7-issue-pr 改造

**新增**：PR 创建后向后端更新 `pr-created`

```bash
# 在 gh pr create 成功后

update_issue_status "$ISSUE_NUM" "pr-created"
```

### 5.7 D5: 003-8-issue-test 改造

**新增**：开始测试时向后端更新 `testing`

```bash
# 在找到关联 PR、开始执行测试前

update_issue_status "$ISSUE_NUM" "testing"
```

### 5.8 D6: 003-9-issue-review 改造

**新增**：merge 后更新 `merged`，reject 后更新 `rejected`

```bash
# merge 分支
merge_issue() {
  local issue_num="$1"
  
  # 原有逻辑：gh pr merge ...
  
  # 新增：更新后端状态
  update_issue_status "$issue_num" "merged"
}

# reject 分支
reject_issue() {
  local issue_num="$1"
  
  # 原有逻辑：gh pr comment ... gh issue edit ...
  
  # 新增：更新后端状态
  update_issue_status "$issue_num" "rejected"
}
```

### 5.9 D7: SessionEnd hook 自动释放

**改造位置**：`.claude/hooks/29-session-end/base.sh`（或新增）

```bash
# 在 SessionEnd hook 末尾

release_session_issues() {
  local session_id=$(json_get '.session_id')
  local backend_url
  backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)
  [ -z "$backend_url" ] && return 0
  
  local result
  result=$(curl -s --max-time 5 -X POST "$backend_url/api/issue/release-session" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$session_id\"}" 2>/dev/null)
  
  local count
  count=$(echo "$result" | jq -r '.count // 0')
  [ "$count" -gt 0 ] && log "INFO" "Released $count issues for session $session_id"
}

release_session_issues
```

---

## 六、改造清单汇总

| 技能 | 文件 | 改造内容 | 后端 API |
|------|------|----------|----------|
| 003-4-issue-claim | `scripts/03UserPromptSubmit.sh` | list 后过滤 + claim 前原子领取 | `/api/issue/check` + `/api/issue/claim` |
| 003-5-issue-fix | `scripts/03UserPromptSubmit.sh` | 创建分支后标记 fixing | `/api/issue/status` |
| 003-6-issue-done | `scripts/03UserPromptSubmit.sh` | 标记完成后更新 ready-for-pr | `/api/issue/status` |
| 003-7-issue-pr | `scripts/03UserPromptSubmit.sh` | PR 创建后标记 pr-created | `/api/issue/status` |
| 003-8-issue-test | `scripts/03UserPromptSubmit.sh` | 开始测试时标记 testing | `/api/issue/status` |
| 003-9-issue-review | `scripts/03UserPromptSubmit.sh` | merge/reject 后更新状态 | `/api/issue/status` |
| SessionEnd hook | `hooks/29-session-end/base.sh` | 自动释放该 session 的 issue | `/api/issue/release-session` |

---

## 七、降级策略

### 7.1 后端不可用时

```bash
# _backend_available 检查失败时，所有 update_issue_status 直接 return 0
# _call_backend 返回空时，filter_claimed_issues 返回原始列表
# claim_issue_backend 失败时，提示用户但允许继续（或阻止，取决于策略）
```

### 7.2 建议的降级行为

| 场景 | 行为 |
|------|------|
| 未配置 backend.conf | 完全走原有逻辑，不调用后端 |
| 后端 health 检查失败 | 完全走原有逻辑，不调用后端 |
| `/api/issue/check` 失败 | 返回全部 issue 列表（不过滤） |
| `/api/issue/claim` 失败 | 阻止领取，提示用户（避免冲突） |
| `/api/issue/status` 失败 | 静默忽略，继续操作 GitHub |
| `/api/issue/release-session` 失败 | 静默忽略 |

---

## 八、数据流总结

```
用户执行 /003-4-issue-claim
  │
  ├─ 1. gh issue list → 获取 GitHub open issues
  ├─ 2. POST /api/issue/check → 后端过滤已领取的
  ├─ 3. 展示空闲 issues
  ├─ 4. 用户选择 #10
  ├─ 5. POST /api/issue/claim → 后端原子领取
  ├─ 6. 后端返回 success
  └─ 7. gh issue edit #10 --add-assignee @me --add-label "in-progress"

用户执行 /003-5-issue-fix #10
  │
  ├─ 1. 检查 assignee（原有逻辑）
  ├─ 2. git checkout -b fix/issue-10-xxx
  ├─ 3. POST /api/issue/status → 标记 fixing
  └─ 4. gh issue comment #10 "开始解决..."

...（后续技能类似，各标记对应状态）

SessionEnd hook 触发
  │
  └─ POST /api/issue/release-session → 自动释放该 session 的所有 issue
```

---

## 九、与现有代码的关联

### 9.1 claude-tap-plus 代理侧改造（模块 C 补充）

当前代理启动时会设置 `ANTHROPIC_BASE_URL` 环境变量。需要额外注入 `CLAUDE_SESSION_ID`：

```go
// cmd/claude-tap/main.go 中，启动子进程前

// 从请求中提取 session_id（首次 API 调用时）
// 或从 hook 传入的环境变量获取

// 在 BuildChildEnv 中增加：
childEnv = append(childEnv, fmt.Sprintf("CLAUDE_SESSION_ID=%s", sessionID))
```

**但更简单的方式**：hook 脚本直接从 `json_get '.session_id'` 获取，不需要代理注入环境变量。

### 9.2 后端服务启动

后端服务作为独立进程运行，与 claude-tap-plus 代理分离：

```bash
# 启动后端（单独命令或守护进程）
claude-tap-plus backend --port 8080 --db ./backend.db

# 或
claude-tap-plus backend --config ./backend.conf
```

### 9.3 配置统一

`.claude/backend.conf` 格式：

```
BACKEND_URL=http://localhost:8080
```

各技能脚本和 hook 都读取同一配置文件。

---

## 十、数据库表设计（复用模块 B）

Issue 技能改造不新增表，复用模块 B 的 `issue_claims` 表：

```sql
-- 见 claude-tap-plus-issue-design.md 第 10.1 节

CREATE TABLE issue_claims (
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
```

**本模块的写入时序**：

| 时机 | 操作 | 调用方 |
|------|------|--------|
| claim 时 | INSERT OR IGNORE issue_claims (idle) → UPDATE status='claimed', session_id=xxx | 003-4-issue-claim |
| fixing 时 | UPDATE status='fixing' | 003-5-issue-fix |
| done 时 | UPDATE status='ready-for-pr' | 003-6-issue-done |
| pr 时 | UPDATE status='pr-created' | 003-7-issue-pr |
| test 时 | UPDATE status='testing' | 003-8-issue-test |
| merge 时 | UPDATE status='merged' | 003-9-issue-review |
| reject 时 | UPDATE status='rejected' | 003-9-issue-review |
| SessionEnd | UPDATE status='idle', session_id=NULL (排除 merged/rejected) | SessionEnd hook |

---

## 十一、测试验证点

| 场景 | 验证内容 |
|------|----------|
| 单 Agent 正常工作 | 未配置后端时，所有技能走原有逻辑 |
| 后端可用时 claim | 两个 Agent 同时 claim 同一 issue，只有一个成功 |
| 后端不可用时 claim | 降级为原有逻辑，直接操作 GitHub |
| 状态流转 | claim → fixing → ready-for-pr → pr-created → testing → merged |
| SessionEnd 释放 | 会话关闭后，issue 变为 idle，可被其他 Agent 领取 |
| reject 后重新领取 | rejected 状态 issue 被释放后，可被重新 claim |
