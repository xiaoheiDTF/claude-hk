#!/bin/bash
# 003-6-issue-pr 上下文注入
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
source "$PROJECT_DIR/.claude/skills/log.sh"

# 从 prompt 提取 issue 编号
PROMPT="$1"
ISSUE_NUM=$(echo "$PROMPT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

echo "=== Issue PR 上下文 ==="
echo "日期: $(date +%Y-%m-%d)"
echo "当前分支: $(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null)"

# 检查当前分支是否有未提交的变更
if [ -n "$(git -C "$PROJECT_DIR" status --porcelain 2>/dev/null)" ]; then
  echo "注意: 有未提交的变更"
fi

# 检查是否已有关联 PR
if [ -n "$ISSUE_NUM" ] && command -v gh &>/dev/null; then
  branch=$(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null)
  existing_pr=$(gh pr list --head "$branch" --json number,title,state 2>/dev/null)
  if [ -n "$existing_pr" ] && [ "$existing_pr" != "[]" ]; then
    echo "当前分支已有 PR:"
    echo "$existing_pr" | jq -r '.[] | "  PR #\(.number): \(.title) (\(.state))"'
  fi
fi

skill_log "INFO" "[inject] issue-pr context injected for #$ISSUE_NUM"
