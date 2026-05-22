#!/bin/bash
# 003-8-issue-test 上下文注入
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="003-8-issue-test"
source "$PROJECT_DIR/.claude/skills/log.sh"

# gh 路径检测
_gh() { command -v gh &>/dev/null && gh "$@" || "C:/Program Files/GitHub CLI/gh.exe" "$@"; }

# 从 prompt 提取 issue 编号
PROMPT="$1"
ISSUE_NUM=$(echo "$PROMPT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

echo "=== Issue Test 上下文 ==="
echo "日期: $(date +%Y-%m-%d)"
echo "当前分支: $(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null)"

if [ -n "$ISSUE_NUM" ]; then
  if _gh --version &>/dev/null; then
    # 找关联 PR
    pr_info=$(_gh pr list --state open --json number,title,body --jq ".[] | select(.body | test(\"Closes #$ISSUE_NUM\")) | {number,title}" 2>/dev/null)
    if [ -n "$pr_info" ]; then
      echo "关联 PR:"
      echo "$pr_info" | jq -r '"  PR #\(.number): \(.title)"'
    else
      echo "未找到关联 issue #$ISSUE_NUM 的 PR"
    fi
  fi
else
  # 无参数时列出有 PR 的 issues
  if _gh --version &>/dev/null; then
    echo ""
    echo "=== 可测试的 PRs ==="
    prs=$(_gh pr list --state open --json number,title,headRefName 2>/dev/null)
    if [ -n "$prs" ] && [ "$prs" != "[]" ]; then
      echo "$prs" | jq -r '.[] | "  PR #\(.number): \(.title) (分支: \(.headRefName))"'
    else
      echo "(暂无 open PRs)"
    fi
  fi
fi

skill_log "INFO" "[inject] issue-test context injected for #$ISSUE_NUM"
