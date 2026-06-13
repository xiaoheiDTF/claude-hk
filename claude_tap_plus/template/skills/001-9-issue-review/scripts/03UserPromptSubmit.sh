#!/bin/bash
# 001-9-issue-review 上下文注入
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="001-9-issue-review"
source "$PROJECT_DIR/.claude/skills/log.sh"
source "$PROJECT_DIR/.claude/skills/backend.sh"

# gh 路径检测
_gh() { command -v gh &>/dev/null && gh "$@" || "C:/Program Files/GitHub CLI/gh.exe" "$@"; }

# 从 prompt 提取 issue 编号和子命令
PROMPT="$1"
ISSUE_NUM=$(echo "$PROMPT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

# 检测子命令：merge 或 reject
REVIEW_ACTION=""
if echo "$PROMPT" | grep -qi "merge"; then
  REVIEW_ACTION="merge"
elif echo "$PROMPT" | grep -qi "reject"; then
  REVIEW_ACTION="reject"
fi

echo "=== Issue Review 上下文 ==="
echo "日期: $(date +%Y-%m-%d)"

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

  # 向后端标记状态（降级静默）
  if [ "$REVIEW_ACTION" = "merge" ]; then
    # merge 时先标记 reviewing，最终由 Claude 执行完 merge 后调 merged
    update_issue_status "$ISSUE_NUM" "reviewing"
    update_issue_status "$ISSUE_NUM" "merged"
  elif [ "$REVIEW_ACTION" = "reject" ]; then
    update_issue_status "$ISSUE_NUM" "reviewing"
    update_issue_status "$ISSUE_NUM" "rejected"
  else
    # 默认：仅标记 reviewing
    update_issue_status "$ISSUE_NUM" "reviewing"
  fi
else
  if _gh --version &>/dev/null; then
    echo ""
    echo "=== 可审核的 PRs ==="
    prs=$(_gh pr list --state open --json number,title,headRefName 2>/dev/null)
    if [ -n "$prs" ] && [ "$prs" != "[]" ]; then
      echo "$prs" | jq -r '.[] | "  PR #\(.number): \(.title) (分支: \(.headRefName))"'
      echo ""
      echo "用法:"
      echo "  /001-9-issue-review merge #<N>   合并 PR"
      echo "  /001-9-issue-review reject #<N>  打回 PR"
    else
      echo "(暂无 open PRs)"
    fi
  fi
fi

skill_log "INFO" "[inject] issue-review context injected for #$ISSUE_NUM action=$REVIEW_ACTION"

# 补充上下文：从 03-user-prompt-submit/ 加载学习进化内容
source "$PROJECT_DIR/.claude/skills/_load_supplementary.sh"
_load_supplementary "$PROJECT_DIR/.claude/skills/${SKILL_TAG}"
if [ -n "$SUPPLEMENTARY_TEXT" ]; then
  echo ""
  echo "=== 补充上下文 ==="
  echo "$SUPPLEMENTARY_TEXT"
fi
