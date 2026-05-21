#!/bin/bash
# 003-5-issue-fix 上下文注入
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
source "$PROJECT_DIR/.claude/skills/log.sh"

# 从 prompt 提取 issue 编号
PROMPT="$1"
ISSUE_NUM=$(echo "$PROMPT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

echo "=== Issue 解决上下文 ==="
echo "日期: $(date +%Y-%m-%d)"
echo "当前分支: $(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null)"

if [ -n "$ISSUE_NUM" ] && command -v gh &>/dev/null; then
  issue_info=$(gh issue view "$ISSUE_NUM" --json title,labels,assignees,state 2>/dev/null)
  if [ -n "$issue_info" ]; then
    echo "Issue #$ISSUE_NUM:"
    echo "  标题: $(echo "$issue_info" | jq -r '.title')"
    echo "  标签: $(echo "$issue_info" | jq -r '.labels[].name' | tr '\n' ',' | sed 's/,$//')"
    echo "  状态: $(echo "$issue_info" | jq -r '.state')"
  fi
fi

skill_log "INFO" "[inject] issue-fix context injected for #$ISSUE_NUM"
