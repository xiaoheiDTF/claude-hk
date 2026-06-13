#!/bin/bash
# 001-5-issue-fix 上下文注入
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="001-5-issue-fix"
source "$PROJECT_DIR/.claude/skills/log.sh"
source "$PROJECT_DIR/.claude/skills/backend.sh"

# gh 路径检测
_gh() { command -v gh &>/dev/null && gh "$@" || "C:/Program Files/GitHub CLI/gh.exe" "$@"; }

# 从 prompt 提取 issue 编号
PROMPT="$1"
ISSUE_NUM=$(echo "$PROMPT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

echo "=== Issue 解决上下文 ==="
echo "日期: $(date +%Y-%m-%d)"
echo "当前分支: $(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null)"

if [ -n "$ISSUE_NUM" ]; then
  if _gh --version &>/dev/null; then
    issue_info=$(_gh issue view "$ISSUE_NUM" --json title,labels,assignees,state 2>/dev/null)
    if [ -n "$issue_info" ]; then
      echo "Issue #$ISSUE_NUM:"
      echo "  标题: $(echo "$issue_info" | jq -r '.title')"
      echo "  标签: $(echo "$issue_info" | jq -r '.labels[].name' | tr '\n' ',' | sed 's/,$//')"
      echo "  状态: $(echo "$issue_info" | jq -r '.state')"
    fi
  fi

  # 向后端标记 fixing 状态（降级静默）
  update_issue_status "$ISSUE_NUM" "fixing"
else
  # 无参数时列出已领取（in-progress）的 issues
  if _gh --version &>/dev/null; then
    echo ""
    echo "=== 可解决的 Issues ==="
    issues=$(_gh issue list --state open --label "in-progress" --assignee @me --json number,title,labels 2>/dev/null)
    if [ -n "$issues" ] && [ "$issues" != "[]" ]; then
      count=$(echo "$issues" | jq 'length')
      echo "当前已领取的 open issues（共 $count 个）:"
      echo ""
      echo "$issues" | jq -r '.[] | "#\(.number) [\([.labels[].name] | join(","))] \(.title)\n  请回复 /001-5-issue-fix #\(.number) 来开始解决\n"'
    else
      echo "(暂无已领取的 issues，请先使用 /001-4-issue-claim 领取)"
    fi
  else
    echo "gh CLI 不可用，请先安装 GitHub CLI"
    echo "请指定 issue 编号，例如: /001-5-issue-fix #5"
  fi
fi

skill_log "INFO" "[inject] issue-fix context injected for #$ISSUE_NUM"

# 补充上下文：从 03-user-prompt-submit/ 加载学习进化内容
source "$PROJECT_DIR/.claude/skills/_load_supplementary.sh"
_load_supplementary "$PROJECT_DIR/.claude/skills/${SKILL_TAG}"
if [ -n "$SUPPLEMENTARY_TEXT" ]; then
  echo ""
  echo "=== 补充上下文 ==="
  echo "$SUPPLEMENTARY_TEXT"
fi
