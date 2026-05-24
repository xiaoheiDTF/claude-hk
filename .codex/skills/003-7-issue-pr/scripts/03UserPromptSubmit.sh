#!/bin/bash
# 003-7-issue-pr 上下文注�?
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="003-7-issue-pr"
source "$PROJECT_DIR/\.codex/skills/log.sh"

# gh 路径检�?
_gh() { command -v gh &>/dev/null && gh "$@" || "C:/Program Files/GitHub CLI/gh.exe" "$@"; }

# �?prompt 提取 issue 编号
PROMPT="$1"
ISSUE_NUM=$(echo "$PROMPT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

echo "=== Issue PR 上下�?==="
echo "日期: $(date +%Y-%m-%d)"
echo "当前分支: $(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null)"

# 检查当前分支是否有未提交的变更
if [ -n "$(git -C "$PROJECT_DIR" status --porcelain 2>/dev/null)" ]; then
  echo "注意: 有未提交的变�?
fi

if [ -n "$ISSUE_NUM" ]; then
  # 检查是否已有关�?PR
  if _gh --version &>/dev/null; then
    branch=$(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null)
    existing_pr=$(_gh pr list --head "$branch" --json number,title,state 2>/dev/null)
    if [ -n "$existing_pr" ] && [ "$existing_pr" != "[]" ]; then
      echo "当前分支已有 PR:"
      echo "$existing_pr" | jq -r '.[] | "  PR #\(.number): \(.title) (\(.state))"'
    fi
  fi
else
  # 无参数时列出 open PRs �?ready-for-pr �?issues
  if _gh --version &>/dev/null; then
    echo ""
    echo "=== PR 状�?==="
    # 列出当前分支关联�?PR
    branch=$(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null)
    prs=$(_gh pr list --state open --json number,title,headRefName 2>/dev/null)
    if [ -n "$prs" ] && [ "$prs" != "[]" ]; then
      echo "Open PRs:"
      echo "$prs" | jq -r '.[] | "  PR #\(.number): \(.title) (分支: \(.headRefName))"'
      echo ""
    fi
    # 列出 ready-for-pr �?issues
    issues=$(_gh issue list --state open --label "ready-for-pr" --assignee @me --json number,title,labels 2>/dev/null)
    if [ -n "$issues" ] && [ "$issues" != "[]" ]; then
      count=$(echo "$issues" | jq 'length')
      echo "待提 PR �?issues（共 $count 个）:"
      echo ""
      echo "$issues" | jq -r '.[] | "#\(.number) [\([.labels[].name] | join(","))] \(.title)\n  请回�?/003-7-issue-pr #\(.number) 来提�?PR\n"'
    fi
  else
    echo "gh CLI 不可用，请先安装 GitHub CLI"
    echo "请指�?issue 编号，例�? /003-7-issue-pr #5"
  fi
fi

skill_log "INFO" "[inject] issue-pr context injected for #$ISSUE_NUM"
