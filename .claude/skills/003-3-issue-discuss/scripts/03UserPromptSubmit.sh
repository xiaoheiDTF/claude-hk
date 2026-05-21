#!/bin/bash
# 003-3-issue-discuss 上下文注入
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
source "$PROJECT_DIR/.claude/skills/log.sh"

# 从 prompt 提取 issue 编号
PROMPT="$1"
ISSUE_NUM=$(echo "$PROMPT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

if [ -z "$ISSUE_NUM" ]; then
  echo "=== Issue 讨论上下文 ==="
  echo "请指定 issue 编号，例如: /003-3-issue-discuss #5"
  exit 0
fi

echo "=== Issue #$ISSUE_NUM 讨论上下文 ==="
echo "日期: $(date +%Y-%m-%d)"
echo ""

# 拉取 issue 详情
if command -v gh &>/dev/null; then
  issue_json=$(gh issue view "$ISSUE_NUM" --json title,body,labels,state,assignees,comments 2>/dev/null)
  if [ -n "$issue_json" ]; then
    echo "Issue 详情:"
    gh issue view "$ISSUE_NUM" 2>/dev/null
    echo ""
    echo "评论:"
    gh api "repos/{owner}/{repo}/issues/$ISSUE_NUM/comments" --jq '.[] | "---\n作者: \(.user.login)\n时间: \(.created_at)\n\(.body)\n"' 2>/dev/null || echo "(无评论)"
  else
    echo "无法获取 issue #$ISSUE_NUM，请确认编号是否正确"
  fi
else
  echo "gh CLI 不可用，无法拉取 issue"
fi

skill_log "INFO" "[inject] issue-discuss context injected for #$ISSUE_NUM"
