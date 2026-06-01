#!/bin/bash
# 001-3-issue-discuss 上下文注�?
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="001-3-issue-discuss"
source "$PROJECT_DIR/\.codex/skills/log.sh"

# gh 路径检�?
_gh() { command -v gh &>/dev/null && gh "$@" || "C:/Program Files/GitHub CLI/gh.exe" "$@"; }

# �?prompt 提取 issue 编号
PROMPT="$1"
ISSUE_NUM=$(echo "$PROMPT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

if [ -z "$ISSUE_NUM" ]; then
  echo "=== Issue 讨论上下�?==="
  echo "日期: $(date +%Y-%m-%d)"
  echo ""
  # 列出所�?open issues
  if _gh --version &>/dev/null; then
    echo "当前 open issues:"
    echo ""
    issues=$(_gh issue list --state open --json number,title,labels,assignees 2>/dev/null)
    if [ -n "$issues" ] && [ "$issues" != "[]" ]; then
      echo "$issues" | jq -r '.[] | "#\(.number) [\([.labels[].name] | join(","))] \(.title)\n  状�? \(if (.assignees | length) > 0 then "已领�?(\(.assignees[].login))" else "未领�? end)\n  请回�?/001-3-issue-discuss #\(.number) 来讨论\n"'
    else
      echo "(暂无 open issues)"
    fi
  else
    echo "gh CLI 不可用，请先安装 GitHub CLI"
    echo "请指�?issue 编号，例�? /001-3-issue-discuss #5"
  fi
  exit 0
fi

echo "=== Issue #$ISSUE_NUM 讨论上下�?==="
echo "日期: $(date +%Y-%m-%d)"
echo ""

# 拉取 issue 详情
if _gh --version &>/dev/null; then
  issue_json=$(_gh issue view "$ISSUE_NUM" --json title,body,labels,state,assignees,comments 2>/dev/null)
  if [ -n "$issue_json" ]; then
    echo "Issue 详情:"
    _gh issue view "$ISSUE_NUM" 2>/dev/null
    echo ""
    echo "评论:"
    _gh api "repos/{owner}/{repo}/issues/$ISSUE_NUM/comments" --jq '.[] | "---\n作�? \(.user.login)\n时间: \(.created_at)\n\(.body)\n"' 2>/dev/null || echo "(无评�?"
  else
    echo "无法获取 issue #$ISSUE_NUM，请确认编号是否正确"
  fi
else
  echo "gh CLI 不可用，无法拉取 issue"
fi

skill_log "INFO" "[inject] issue-discuss context injected for #$ISSUE_NUM"
