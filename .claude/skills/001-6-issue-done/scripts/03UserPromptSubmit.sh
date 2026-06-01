#!/bin/bash
# 001-6-issue-done 上下文注入
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="001-6-issue-done"
source "$PROJECT_DIR/.claude/skills/log.sh"
source "$PROJECT_DIR/.claude/skills/backend.sh"

# 从 prompt 提取 issue 编号
PROMPT="$1"
ISSUE_NUM=$(echo "$PROMPT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

echo "=== Issue Done 上下文 ==="
echo "日期: $(date +%Y-%m-%d)"
echo "当前分支: $(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null)"

# 检查未提交变更
if [ -n "$(git -C "$PROJECT_DIR" status --porcelain 2>/dev/null)" ]; then
  echo "注意: 有未提交的变更，请先提交"
fi

if [ -n "$ISSUE_NUM" ]; then
  if command -v gh &>/dev/null || [ -x "C:/Program Files/GitHub CLI/gh.exe" ]; then
    _gh() { command -v gh &>/dev/null && gh "$@" || "C:/Program Files/GitHub CLI/gh.exe" "$@"; }
    issue_info=$(_gh issue view "$ISSUE_NUM" --json title,labels,state 2>/dev/null)
    if [ -n "$issue_info" ]; then
      echo "Issue #$ISSUE_NUM:"
      echo "  标题: $(echo "$issue_info" | jq -r '.title')"
      echo "  标签: $(echo "$issue_info" | jq -r '.labels[].name' | tr '\n' ',' | sed 's/,$//')"
    fi
  fi

  # 向后端标记 ready-for-pr 状态（降级静默）
  update_issue_status "$ISSUE_NUM" "ready-for-pr"
fi

skill_log "INFO" "[inject] issue-done context injected for #$ISSUE_NUM"
