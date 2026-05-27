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
# 位置：.claude/skills/backend.sh，各技能 source

BACKEND_URL=""

# 读取后端 URL（带缓存，只读一次）
_load_backend_url() {
  if [ -z "$BACKEND_URL" ]; then
    BACKEND_URL=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null \
      | grep '^BACKEND_URL=' | cut -d= -f2)
  fi
}

# 检查后端是否可用（2 秒超时）
_backend_available() {
  _load_backend_url
  [ -n "$BACKEND_URL" ] && curl -s --max-time 2 "$BACKEND_URL/health" > /dev/null 2>&1
}

# 通用后端调用（5 秒超时，失败静默返回空）
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

# 获取当前 session_id（json_get 为当前推荐方案）
_get_session_id() {
  if command -v json_get >/dev/null 2>&1; then
    json_get '.session_id' 2>/dev/null
    return
  fi
  # 未来可选：代理注入环境变量
  if [ -n "$CLAUDE_SESSION_ID" ]; then
    echo "$CLAUDE_SESSION_ID"
  fi
}

# 状态→GitHub Label 映射
_status_to_label() {
  case "$1" in
    claimed)       echo "in-progress" ;;
    fixing)        echo "fixing" ;;
    ready-for-pr)  echo "ready-for-pr" ;;
    pr-created)    echo "pr-created" ;;
    testing)       echo "testing" ;;
    reviewing)     echo "reviewing" ;;
    rejected)      echo "rejected" ;;
    merged|idle)   echo "" ;;
    *)             echo "" ;;
  esac
}

# 同步 GitHub label（后端状态变更后调用）
_sync_github_label() {
  local issue_num="$1"
  local old_status="$2"
  local new_status="$3"

  local old_label=$(_status_to_label "$old_status")
  local new_label=$(_status_to_label "$new_status")

  # 移除旧 label
  [ -n "$old_label" ] && [ "$old_label" != "$new_label" ] && \
    gh issue edit "$issue_num" --remove-label "$old_label" 2>/dev/null

  # 添加新 label
  [ -n "$new_label" ] && \
    gh issue edit "$issue_num" --add-label "$new_label" 2>/dev/null

  # merged 特殊处理：关闭 issue
  [ "$new_status" = "merged" ] && gh issue close "$issue_num" 2>/dev/null
}

# 更新后端状态 + 同步 GitHub label（双写）
update_issue_status() {
  local issue_number="$1"
  local new_status="$2"
  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 0

  local session_id
  session_id=$(_get_session_id)
  local repo
  repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner' 2>/dev/null)

  # 1. 调后端更新状态
  local result
  result=$(_call_backend "/api/issue/status" "{
    \"repo_full_name\":\"$repo\",
    \"issue_number\":$issue_number,
    \"session_id\":\"$session_id\",
    \"status\":\"$new_status\"
  }")

  # 2. 从响应中获取 previous_status，同步 GitHub label
  local old_status
  old_status=$(echo "$result" | jq -r '.previous_status // empty' 2>/dev/null)
  [ -n "$old_status" ] && _sync_github_label "$issue_number" "$old_status" "$new_status"
}
```

> **注**：`json_get '.session_id'` 是当前推荐的 session_id 获取方式（hook 环境中已可用）。`CLAUDE_SESSION_ID` 环境变量作为未来可选方案，需要代理注入环境变量。

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
  # 后端领取成功，再操作 GitHub（label 由 _sync_github_label 自动同步）
  gh issue edit "$ISSUE_NUM" --add-assignee @me
  _sync_github_label "$ISSUE_NUM" "" "claimed"
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
# update_issue_status 来自 backend.sh（见 5.2 节），已包含 GitHub label 自动同步
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

> **SessionEnd hook 双重职责**：除了释放 issue，SessionEnd hook 还需调用 `/api/session/close` 注销会话（见 session-design.md 5.4 节）。建议执行顺序：先释放 issue，再注销 session。

---

## 六、改造清单汇总

| 技能 | 文件 | 改造内容 | 后端 API |
|------|------|----------|----------|
| 003-4-issue-claim | `scripts/03UserPromptSubmit.sh` | list 后过滤 + claim 前原子领取 | `/api/issue/check` + `/api/issue/claim` |
| 003-5-issue-fix | `scripts/03UserPromptSubmit.sh` | 创建分支后标记 fixing | `/api/issue/status` |
| 003-6-issue-done | `scripts/03UserPromptSubmit.sh` | 标记完成后更新 ready-for-pr | `/api/issue/status` |
| 003-7-issue-pr | `scripts/03UserPromptSubmit.sh` | PR 创建后标记 pr-created | `/api/issue/status` |
| 003-8-issue-test | `scripts/03UserPromptSubmit.sh` | 开始测试时标记 testing | `/api/issue/status` |
| 003-9-issue-review | `scripts/03UserPromptSubmit.sh` | 审核开始时标记 `reviewing`，再根据结果标记 `merged` 或 `rejected` | `/api/issue/status` |
| SessionEnd hook | `hooks/29-session-end/base.sh` | 自动释放该 session 的 issue | `/api/issue/release-session` |

---

## 七、降级策略

> 详细设计见 0527 `issue-management-reqs/degradation-strategy.md`。

### 7.1 设计原则

1. **后端是辅助，GitHub 是主链路**：后端只负责状态追踪和冲突防护，不替代 GitHub 操作
2. **失败静默**：后端调用失败不应阻塞或报错到用户层面（claim 除外）
3. **配置开关**：`.claude/backend.conf` 不存在时，完全不调用后端

### 7.2 降级行为表

| 场景 | 触发条件 | 降级行为 | 影响范围 |
|------|----------|----------|----------|
| 未配置后端 | `backend.conf` 不存在或 `BACKEND_URL` 为空 | 所有后端调用跳过，走原有逻辑 | 全部技能 |
| 后端不可用 | `GET /health` 超时或连接失败 | 所有后端调用跳过，走原有逻辑 | 全部技能 |
| `/api/issue/check` 失败 | 请求超时或返回非 JSON | 返回完整 issue 列表（不过滤） | 003-4-issue-claim |
| `/api/issue/claim` 失败 | 请求超时或返回失败 | **阻止领取**，提示用户 | 003-4-issue-claim |
| `/api/issue/status` 失败 | 请求超时或返回失败 | 静默忽略，继续操作 GitHub | 003-5 ~ 003-9 |
| `/api/issue/release-session` 失败 | 请求超时或返回失败 | 静默忽略 | SessionEnd hook |

### 7.3 claim 失败阻塞策略

claim 是唯一一个降级时**阻止**而非静默的操作。原因：没有后端锁的保护，多 Agent 可能同时领取同一 issue，造成冲突。

可选策略：如果用户确认单 Agent 环境，可以提供 `--force` 参数跳过后端直接操作 GitHub。

### 7.4 各 API 降级详细说明

| API | 正常流程 | 降级流程 |
|-----|----------|----------|
| check | gh issue list → check API 过滤 → 展示空闲 issue | gh issue list → 跳过过滤 → 展示全部 open issue |
| claim | 用户选择 → claim API 原子领取 → 成功 → gh issue edit | 用户选择 → claim API 超时 → 提示"后端不可用" → 阻止操作 |
| status | update_issue_status() → 后端 API → 同步 GitHub label | update_issue_status() → 超时 → GitHub label 不变 → 继续主流程 |
| release-session | SessionEnd → release-session API → 释放 issue | SessionEnd → release-session API 超时 → 静默跳过 |

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
  └─ 7. gh issue edit #10 --add-assignee @me（label 由 _sync_github_label 自动同步）

用户执行 /003-5-issue-fix #10
  │
  ├─ 1. 检查 assignee（原有逻辑）
  ├─ 2. git checkout -b fix/issue-10-xxx
  ├─ 3. POST /api/issue/status → 标记 fixing
  └─ 4. gh issue comment #10 "开始解决..."

...（后续技能类似，各标记对应状态）

SessionEnd hook 触发
  │
  ├─ POST /api/issue/release-session → 自动释放该 session 的所有 issue
  └─ POST /api/session/close → 注销会话
```

---

## 九、与现有代码的关联

### 9.1 claude-tap-plus 代理侧改造（模块 C 补充）

**当前方案**：hook 脚本直接从 `json_get '.session_id'` 获取，无需代理改造。`json_get` 在 hook 环境中已可用，不需要额外配置。

**未来可选方案**：代理注入 `CLAUDE_SESSION_ID` 环境变量（需改造 `cmd/claude-tap/main.go`）：

```go
// 在 BuildChildEnv 中增加（未来可选）：
childEnv = append(childEnv, fmt.Sprintf("CLAUDE_SESSION_ID=%s", sessionID))
```

> 此方案可减少 `json_get` 调用开销，但需要修改代理代码，当前阶段不采用。

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
| SessionEnd | UPDATE status='idle', session_id=NULL (排除 merged，rejected 会被释放为 idle) | SessionEnd hook |

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
