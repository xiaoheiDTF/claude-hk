# D1: 001-4-issue-claim 改造

> 创建时间：2026-05-27
> 模块：claude-tap-plus / 模块 D / D1
> 简述：改造 issue 领取技能，加入后端去重过滤和原子领取

---

## 目标

在 `001-4-issue-claim` 技能中：
1. list 后调后端过滤已被其他 Agent 领取的 issue
2. 用户选择后先调后端原子领取，成功再操作 GitHub

## 改造文件

| 文件 | 操作 |
|------|------|
| `.claude/skills/001-4-issue-claim/scripts/03UserPromptSubmit.sh` | **修改** |

## 依赖

- **D0**: 需要 `source backend.sh`（`_call_backend`, `_get_session_id`, `_load_backend_url`）
- **后端 API**: `/api/issue/check`, `/api/issue/claim`

## 当前流程

```
1. gh issue list → 获取 open issues
2. 展示给用户
3. 用户选择后 gh issue edit --add-assignee @me --add-label "in-progress"
```

## 改造后流程

```
1. gh issue list → 获取 open issues
2. [新增] 调后端 /api/issue/check 过滤已领取的
3. 展示过滤后的空闲 issues
4. 用户选择后
5. [新增] 调后端 /api/issue/claim 原子领取
6. 后端成功后再 gh issue edit
7. 后端失败则提示已被领取
```

## 需要新增的函数

### filter_claimed_issues — 去重过滤

```bash
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

  # 过滤出 idle 的 issue
  local idle_numbers
  idle_numbers=$(echo "$check_result" | jq '[.issues[] | select(.status == "idle") | .number]')

  echo "$issues_json" | jq --argjson idle "$idle_numbers" '[.[] | select(.number | IN($idle[]))]'
}
```

### claim_issue_backend — 原子领取

```bash
claim_issue_backend() {
  local issue_num="$1"
  local issue_title="$2"

  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 0

  local session_id
  session_id=$(_get_session_id)
  [ -z "$session_id" ] && return 0

  local repo
  repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner' 2>/dev/null)
  [ -z "$repo" ] && return 0

  local result
  result=$(_call_backend "/api/issue/claim" "{
    \"repo_full_name\":\"$repo\",
    \"issue_number\":$issue_num,
    \"session_id\":\"$session_id\",
    \"issue_title\":\"$issue_title\"
  }")

  if ! echo "$result" | jq -e '.success' > /dev/null 2>&1; then
    local claimed_by
    claimed_by=$(echo "$result" | jq -r '.claimed_by // "unknown"')
    echo "ERROR: Issue #$issue_num 已被 $claimed_by 领取"
    return 1
  fi

  return 0
}
```

### 主流程改造点

```bash
# 1. 获取 issues（原有）
issues=$(gh issue list --state open --json number,title,labels 2>/dev/null)

# 2. [新增] 过滤已领取的
filtered=$(filter_claimed_issues "$issues")

# 3. 展示给用户（改用 $filtered）
# ... 现有展示逻辑，改用 $filtered ...

# 4. 用户确认领取 #N 后
if claim_issue_backend "$ISSUE_NUM" "$ISSUE_TITLE"; then
  gh issue edit "$ISSUE_NUM" --add-assignee @me --add-label "in-progress"
  log "INFO" "Issue #$ISSUE_NUM claimed"
else
  echo "Issue #$ISSUE_NUM 无法领取，请选择其他 issue"
fi
```

## 降级行为

| 场景 | 行为 |
|------|------|
| 未配置 backend.conf | 跳过过滤，返回全部 issue |
| `/api/issue/check` 失败 | 返回全部 issue（不过滤） |
| `/api/issue/claim` 失败 | **阻止领取**，提示已被占用 |

## 验证标准

- [ ] 未配置后端时，展示所有 open issue（原有行为）
- [ ] 后端可用时，已领取的 issue 不在列表中显示
- [ ] 两个 Agent 同时 claim 同一 issue，只有一个成功
- [ ] claim 失败时提示已被谁领取
- [ ] 后端不可用时降级为直接操作 GitHub
