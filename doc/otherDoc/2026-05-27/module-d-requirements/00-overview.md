# 模块 D 总览：Issue 技能集成后端服务

> 创建时间：2026-05-27
> 模块：claude-tap-plus / 模块 D
> 简述：改造 001-x-issue 系列技能，接入后端 Issue 全局管理服务，实现单 GitHub 账号多 Agent 协作

---

## 设计原则

1. **后端优先，GitHub 次之**：状态变更先调后端，再操作 GitHub
2. **最小侵入改造**：现有流程不变，只在关键节点插入后端调用
3. **失败静默**：后端不可用时降级为原有行为
4. **配置驱动**：通过 `.claude/backend.conf` 控制启用

## Issue 工作流调用链

```
001-2-issue (创建) → 001-3-issue-discuss (讨论)
                           ↓
              001-4-issue-claim (领取) ←──┐
                           ↓              │
              001-5-issue-fix (开发)       │
                           ↓              │ 拒绝后重新领取
              001-6-issue-done (完成)      │
                           ↓              │
              001-7-issue-pr (提PR)        │
                           ↓              │
              001-8-issue-test (测试)      │
                           ↓              │
              001-9-issue-review (审核) ───┘
                           ↓
                     merged / rejected
```

> 001-2 和 001-3 不需要后端介入（创建新 issue 无冲突，讨论只读/评论不改变状态）。

## 各技能现状与后端介入点

| 技能 | 当前 GitHub 操作 | 需要后端介入 | 介入环节 |
|------|------------------|-------------|----------|
| 001-2-issue | `gh issue create` | 否 | — |
| 001-3-issue-discuss | `gh issue view`, `gh issue comment` | 否 | — |
| 001-4-issue-claim | `gh issue list` → `gh issue edit --add-assignee` | **是** | list 后去重，edit 前原子领取 |
| 001-5-issue-fix | `gh issue view` → `git checkout -b` → `gh issue comment` | **是** | 标记 fixing 状态 |
| 001-6-issue-done | `git status` → `gh issue edit` (改 label) | **是** | 标记 ready-for-pr 状态 |
| 001-7-issue-pr | `gh pr create` | **是** | 标记 pr-created 状态 |
| 001-8-issue-test | `gh pr view` → 测试 → `gh pr edit` | **是** | 标记 testing 状态 |
| 001-9-issue-review | `gh pr merge` / `gh issue edit` | **是** | 标记 merged/rejected 状态 |

## 各技能脚本中已有的数据源

各技能脚本在运行时已可获取以下数据，无需额外参数传递：

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

## 各技能需要传入后端的数据汇总

| 技能 | repo_full_name | issue_number | issue_title | session_id | status |
|------|:-:|:-:|:-:|:-:|------|
| D1: 001-4-issue-claim | ✓ | ✓ | ✓ | ✓ | — (claim API 自处理) |
| D2: 001-5-issue-fix | ✓ | ✓ | — | ✓ | `fixing` |
| D3: 001-6-issue-done | ✓ | ✓ | — | ✓ | `ready-for-pr` |
| D4: 001-7-issue-pr | ✓ | ✓ | — | ✓ | `pr-created` |
| D5: 001-8-issue-test | ✓ | ✓ | — | ✓ | `testing` |
| D6: 001-9-issue-review | ✓ | ✓ | — | ✓ | `reviewing` → `merged` / `rejected` |
| D7: SessionEnd | — | — | — | ✓ | — (release-session API 自处理) |

## 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                     Claude Code 会话                         │
│                                                              │
│  001-4-issue-claim ──┐                                       │
│  001-5-issue-fix     │                                       │
│  001-6-issue-done    ├──→ 后端调用函数 (backend.sh)          │
│  001-7-issue-pr      │                                       │
│  001-8-issue-test    │                                       │
│  001-9-issue-review ─┘                                       │
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
│  POST /api/issue/check           → 查询 issue 状态                   │
│  POST /api/issue/claim           → 领取 issue                        │
│  POST /api/issue/status          → 更新状态                          │
│  POST /api/issue/release         → 释放 issue                        │
│  POST /api/issue/release-session → SessionEnd 释放所有 issue          │
└─────────────────────────────────────────────────────────────┘
```

## 后端状态定义

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

## 数据库表（复用模块 B）

```sql
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

CREATE INDEX idx_issue_claims_repo ON issue_claims(repo_full_name);
CREATE INDEX idx_issue_claims_session ON issue_claims(session_id);
CREATE INDEX idx_issue_claims_status ON issue_claims(status);
```

## 子需求依赖关系

```
D0 (公共基础设施 backend.sh)
 ├── D1 (001-4-issue-claim 改造) — 最复杂，涉及 check + claim
 ├── D2 (001-5-issue-fix 改造) — 状态标记 fixing
 ├── D3 (001-6-issue-done 改造) — 状态标记 ready-for-pr
 ├── D4 (001-7-issue-pr 改造) — 状态标记 pr-created
 ├── D5 (001-8-issue-test 改造) — 状态标记 testing
 ├── D6 (001-9-issue-review 改造) — 状态标记 merged/rejected
 └── D7 (SessionEnd 自动释放) — 会话结束时释放 issue
```

## 状态写入时序

| 时机 | 操作 | 调用方 |
|------|------|--------|
| claim 时 | INSERT OR IGNORE → UPDATE status='claimed' | D1: 001-4-issue-claim |
| fixing 时 | UPDATE status='fixing' | D2: 001-5-issue-fix |
| done 时 | UPDATE status='ready-for-pr' | D3: 001-6-issue-done |
| pr 时 | UPDATE status='pr-created' | D4: 001-7-issue-pr |
| test 时 | UPDATE status='testing' | D5: 001-8-issue-test |
| 开始审核时 | UPDATE status='reviewing' | D6: 001-9-issue-review |
| merge 时 | UPDATE status='merged' | D6: 001-9-issue-review |
| reject 时 | UPDATE status='rejected' | D6: 001-9-issue-review |
| SessionEnd | UPDATE status='idle', session_id=NULL, claimed_at=NULL (排除 merged/rejected) | D7: SessionEnd hook |

## 改造清单汇总

| 技能 | 改造文件 | 改造内容 | 后端 API |
|------|----------|----------|----------|
| D1: 001-4-issue-claim | `scripts/03UserPromptSubmit.sh` | list 后过滤 + claim 前原子领取 | `/api/issue/check` + `/api/issue/claim` |
| D2: 001-5-issue-fix | `scripts/03UserPromptSubmit.sh` | 创建分支后标记 fixing | `/api/issue/status` |
| D3: 001-6-issue-done | `scripts/03UserPromptSubmit.sh` | 标记完成后更新 ready-for-pr | `/api/issue/status` |
| D4: 001-7-issue-pr | `scripts/03UserPromptSubmit.sh` | PR 创建后标记 pr-created | `/api/issue/status` |
| D5: 001-8-issue-test | `scripts/03UserPromptSubmit.sh` | 开始测试时标记 testing | `/api/issue/status` |
| D6: 001-9-issue-review | `scripts/03UserPromptSubmit.sh` | 开始审核时标记 reviewing，merge/reject 后更新状态 | `/api/issue/status` |
| D7: SessionEnd | `hooks/29-session-end/base.sh` | 自动释放该 session 的 issue | `/api/issue/release-session` |

## 数据流总结

### claim 流程

```
用户执行 /001-4-issue-claim
  │
  ├─ 1. gh issue list → 获取 GitHub open issues
  ├─ 2. POST /api/issue/check → 后端过滤已领取的
  ├─ 3. 展示空闲 issues
  ├─ 4. 用户选择 #10
  ├─ 5. POST /api/issue/claim → 后端原子领取
  ├─ 6. 后端返回 success
  └─ 7. gh issue edit #10 --add-assignee @me --add-label "in-progress"
```

### fix 流程

```
用户执行 /001-5-issue-fix #10
  │
  ├─ 1. 检查 assignee（原有逻辑）
  ├─ 2. git checkout -b fix/issue-10-xxx
  ├─ 3. POST /api/issue/status → 标记 fixing
  └─ 4. gh issue comment #10 "开始解决..."
```

### SessionEnd 流程

```
SessionEnd hook 触发
  │
  └─ POST /api/issue/release-session → 自动释放该 session 的所有 issue（排除 merged/rejected）
```

## 与现有代码的关联

### claude-tap-plus 代理侧改造（模块 C 补充）

当前代理启动时会设置 `ANTHROPIC_BASE_URL` 环境变量。若需注入 `CLAUDE_SESSION_ID`：

```go
// cmd/claude-tap/main.go 中，BuildChildEnv 增加：
childEnv = append(childEnv, fmt.Sprintf("CLAUDE_SESSION_ID=%s", sessionID))
```

> **更简单的方式**：hook 脚本直接从 `json_get '.session_id'` 获取 session_id，不需要代理注入环境变量。D0 的 `backend.sh` 中 `_get_session_id()` 已兼容两种方式。

### 后端服务启动

后端服务作为独立进程运行，与 claude-tap-plus 代理分离：

```bash
claude-tap-plus backend --port 8080 --db ./backend.db
# 或
claude-tap-plus backend --config ./backend.conf
```

### 配置统一

`.claude/backend.conf` 格式：

```
BACKEND_URL=http://localhost:8080
```

各技能脚本（D1-D6）和 SessionEnd hook（D7）都读取同一配置文件。
