#!/bin/bash
# 003-4-issue-claim 上下文注�?
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="003-4-issue-claim"
source "$PROJECT_DIR/\.codex/skills/log.sh"

# gh 路径检�?
_gh() { command -v gh &>/dev/null && gh "$@" || "C:/Program Files/GitHub CLI/gh.exe" "$@"; }

# �?prompt 提取 issue 编号
PROMPT="$1"
ISSUE_NUM=$(echo "$PROMPT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

echo "=== Issue 领取上下�?==="
echo "日期: $(date +%Y-%m-%d)"

if [ -n "$ISSUE_NUM" ]; then
  if _gh --version &>/dev/null; then
    echo "Issue #$ISSUE_NUM 状�?"
    _gh issue view "$ISSUE_NUM" --json title,state,assignees,labels --jq '{title,state,assignees:[.assignees[].login],labels:[.labels[].name]}' 2>/dev/null || echo "无法获取 issue 信息"
  fi
else
  # 无参数时列出未领取的 open issues
  if _gh --version &>/dev/null; then
    echo ""
    echo "=== 可领取的 Issues ==="
    issues=$(_gh issue list --state open --json number,title,labels,assignees 2>/dev/null)
    if [ -n "$issues" ] && [ "$issues" != "[]" ]; then
      # 过滤出无 assignee �?issues
      unclaimed=$(echo "$issues" | jq -r '[.[] | select(.assignees | length == 0)]')
      count=$(echo "$unclaimed" | jq 'length')
      echo "当前未领取的 open issues（共 $count 个）:"
      echo ""
      echo "$unclaimed" | jq -r '.[] | "#\(.number) [\([.labels[].name] | join(","))] \(.title)\n  状�? 未领取\n  请回�?/003-4-issue-claim #\(.number) 来领取\n"'
    else
      echo "(暂无 open issues)"
    fi
  else
    echo "gh CLI 不可用，请先安装 GitHub CLI"
    echo "请指�?issue 编号，例�? /003-4-issue-claim #5"
  fi
fi

skill_log "INFO" "[inject] issue-claim context injected for #$ISSUE_NUM"
