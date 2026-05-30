# B6-1: 003-4-issue-claim 技能改造

> 创建时间：2026-05-27
> 模块：claude-tap-plus / Issue 管理 / 技能改造
> 简述：改造 issue-claim 技能，增加后端去重和原子领取

---

## 需求描述

改造 `003-4-issue-claim` 技能脚本，在现有的 `gh issue list` + `gh issue edit` 流程中插入后端 API 调用，实现：
1. 展示 issue 列表前，过滤已被其他 session 领取的 issue
2. 用户确认领取时，先调后端原子领取，成功后再操作 GitHub

## 改造内容

### 改造前流程

```
gh issue list → 展示全部 open issue → 用户选择 → gh issue edit --add-assignee @me
```

### 改造后流程

```
gh issue list → POST /api/issue/check 去重 → 展示空闲 issue → 用户选择
  → POST /api/issue/claim 原子领取 → 成功 → gh issue edit --add-assignee @me --add-label "in-progress"
                                       → 失败 → 提示已被领取
```

## 具体改动

### 1. 新增：引入后端基础设施

```bash
source "$CLAUDE_PROJECT_DIR/.claude/skills/backend.sh"
```

### 2. 新增：issue 列表去重

在展示 issue 列表前，调用 check API 过滤：

```bash
gh_issues=$(gh issue list --state open --json number,title,labels)
numbers=$(echo "$gh_issues" | jq '[.[].number]')
repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner')
_load_backend_url

if [ -n "$BACKEND_URL" ]; then
  check_result=$(curl -s -X POST "$BACKEND_URL/api/issue/check" \
    -H "Content-Type: application/json" \
    -d "{\"repo_full_name\":\"$repo\",\"issue_numbers\":$numbers}")
  idle_numbers=$(echo "$check_result" | jq '[.issues[] | select(.status == "idle") | .number]')
  gh_issues=$(echo "$gh_issues" | jq --argjson idle "$idle_numbers" '[.[] | select(.number | IN($idle[]))]')
fi
```

### 3. 新增：原子领取

用户确认领取后，先调后端，成功后同步 GitHub label（无 label → `in-progress`）：

```bash
session_id=$(json_get '.session_id')
issue_title=$(gh issue view $ISSUE_NUM --json title --jq '.title')

claim_result=$(curl -s -X POST "$BACKEND_URL/api/issue/claim" \
  -H "Content-Type: application/json" \
  -d "{
    \"repo_full_name\":\"$repo\",
    \"issue_number\":$ISSUE_NUM,
    \"session_id\":\"$session_id\",
    \"issue_title\":\"$issue_title\"
  }")

if echo "$claim_result" | jq -e '.success' > /dev/null; then
  gh issue edit $ISSUE_NUM --add-assignee @me
  # 同步 GitHub label：添加 in-progress
  _sync_github_label "$ISSUE_NUM" "" "claimed"
  log "INFO" "Issue #$ISSUE_NUM claimed successfully"
else
  echo "Issue #$ISSUE_NUM 已被其他 session 领取"
fi
```

> claim 时旧状态为 idle（无 label），`_sync_github_label` 只添加 `in-progress`，不移除旧 label。

## 降级策略

如果后端不可用（`backend_url` 为空或请求超时），回退到原有行为（不做去重），确保功能不中断。

## 验收标准

- [x] 后端可用时，只展示空闲 issue
- [x] 后端可用时，领取操作是原子的（先锁后 gh）
- [x] 后端可用时，领取成功后 GitHub 自动添加 `in-progress` label
- [x] 后端不可用时，回退到原有行为
- [x] 已被领取的 issue 不出现在列表中
