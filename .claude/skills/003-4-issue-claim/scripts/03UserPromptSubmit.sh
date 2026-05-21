#!/bin/bash
# 003-4-issue-claim 上下文注入
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
source "$PROJECT_DIR/.claude/skills/log.sh"

# 从 prompt 提取 issue 编号
PROMPT="$1"
ISSUE_NUM=$(echo "$PROMPT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

echo "=== Issue 领取上下文 ==="
echo "日期: $(date +%Y-%m-%d)"

if [ -n "$ISSUE_NUM" ] && command -v gh &>/dev/null; then
  echo "Issue #$ISSUE_NUM 状态:"
  gh issue view "$ISSUE_NUM" --json title,state,assignees,labels --jq '{title,state,assignees:[.assignees[].login],labels:[.labels[].name]}' 2>/dev/null || echo "无法获取 issue 信息"
fi

skill_log "INFO" "[inject] issue-claim context injected for #$ISSUE_NUM"
