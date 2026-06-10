#!/bin/bash
# 001-4-issue-claim 上下文注入
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="001-4-issue-claim"
source "$PROJECT_DIR/.claude/skills/log.sh"
source "$PROJECT_DIR/.claude/skills/backend.sh"

# gh 路径检测
_gh() { command -v gh &>/dev/null && gh "$@" || "C:/Program Files/GitHub CLI/gh.exe" "$@"; }

# 获取当前 repo 全名
_get_repo() {
  _gh repo view --json nameWithOwner --jq '.nameWithOwner' 2>/dev/null
}

# 从 gh issue list 结果中过滤已被领取的 issue
filter_claimed_issues() {
  local issues_json="$1"

  _load_backend_url
  [ -z "$BACKEND_URL" ] && { echo "$issues_json"; return; }

  local numbers
  numbers=$(echo "$issues_json" | jq '[.[].number]')

  local repo
  repo=$(_get_repo)
  [ -z "$repo" ] && { echo "$issues_json"; return; }

  local check_result
  check_result=$(_call_backend "/api/issue/check" "{\"repo_full_name\":\"$repo\",\"issue_numbers\":$numbers}")
  [ -z "$check_result" ] && { echo "$issues_json"; return; }

  local idle_numbers
  idle_numbers=$(echo "$check_result" | jq '[.issues[] | select(.status == "idle") | .number]')

  echo "$issues_json" | jq --argjson idle "$idle_numbers" '[.[] | select(.number | IN($idle[]))]'
}

# 原子领取 issue
claim_issue_backend() {
  local issue_num="$1"
  local issue_title="$2"

  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 0  # 未配置后端，跳过（单 agent 模式）

  local session_id
  session_id=$(_get_session_id)
  [ -z "$session_id" ] && return 0

  local repo
  repo=$(_get_repo)
  [ -z "$repo" ] && return 0

  # 后端已配置但不可达 → 阻止领取（防多 agent 冲突）
  if ! _backend_available; then
    skill_log "ERROR" "Backend unreachable, BLOCKING claim for #$issue_num"
    echo "ERROR: Backend server is unreachable. Cannot safely claim issue #$issue_num (risk of multi-agent conflict)."
    echo "Please ensure the backend is running at $BACKEND_URL and try again."
    return 1
  fi

  local result
  result=$(_call_backend "/api/issue/claim" "{
    \"repo_full_name\":\"$repo\",
    \"issue_number\":$issue_num,
    \"session_id\":\"$session_id\",
    \"issue_title\":\"$issue_title\"
  }")

  if [ -z "$result" ]; then
    skill_log "ERROR" "Backend claim call returned empty for #$issue_num"
    echo "ERROR: Backend claim call failed unexpectedly for issue #$issue_num."
    return 1
  fi

  if ! echo "$result" | jq -e '.success' > /dev/null 2>&1; then
    local claimed_by
    claimed_by=$(echo "$result" | jq -r '.claimed_by // "unknown"')
    echo "ERROR: Issue #$issue_num 已被 $claimed_by 领取"
    return 1
  fi

  return 0
}

# 从 prompt 提取 issue 编号
PROMPT="$1"
ISSUE_NUM=$(echo "$PROMPT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

echo "=== Issue 领取上下文 ==="
echo "日期: $(date +%Y-%m-%d)"

if [ -n "$ISSUE_NUM" ]; then
  if _gh --version &>/dev/null; then
    echo "Issue #$ISSUE_NUM 状态:"
    _gh issue view "$ISSUE_NUM" --json title,state,assignees,labels --jq '{title,state,assignees:[.assignees[].login],labels:[.labels[].name]}' 2>/dev/null || echo "无法获取 issue 信息"
  fi
else
  # 无参数时列出未领取的 open issues
  if _gh --version &>/dev/null; then
    echo ""
    echo "=== 可领取的 Issues ==="
    issues=$(_gh issue list --state open --json number,title,labels,assignees 2>/dev/null)
    if [ -n "$issues" ] && [ "$issues" != "[]" ]; then
      # 过滤出无 assignee 的 issues
      unclaimed=$(echo "$issues" | jq -r '[.[] | select(.assignees | length == 0)]')

      # [新增] 调后端过滤已被其他 Agent 领取的 issue
      filtered=$(filter_claimed_issues "$unclaimed")

      count=$(echo "$filtered" | jq 'length')
      echo "当前可领取的 open issues（共 $count 个）:"
      echo ""
      echo "$filtered" | jq -r '.[] | "#\(.number) [\([.labels[].name] | join(","))] \(.title)\n  状态: 可领取\n  请回复 /001-4-issue-claim #\(.number) 来领取\n"'
    else
      echo "(暂无 open issues)"
    fi
  else
    echo "gh CLI 不可用，请先安装 GitHub CLI"
    echo "请指定 issue 编号，例如: /001-4-issue-claim #5"
  fi
fi

skill_log "INFO" "[inject] issue-claim context injected for #$ISSUE_NUM"

# 补充上下文：从 03-user-prompt-submit/ 加载学习进化内容
source "$PROJECT_DIR/.claude/skills/_load_supplementary.sh"
_load_supplementary "$PROJECT_DIR/.claude/skills/${SKILL_TAG}"
if [ -n "$SUPPLEMENTARY_TEXT" ]; then
  echo ""
  echo "=== 补充上下文 ==="
  echo "$SUPPLEMENTARY_TEXT"
fi
