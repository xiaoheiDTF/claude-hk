#!/bin/bash
# 003-1-issue-init 上下文注�?
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="003-1-issue-init"
source "$PROJECT_DIR/\.codex/skills/log.sh"

echo "=== Issue Init 上下�?==="
echo "日期: $(date +%Y-%m-%d)"
echo "分支: $(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null)"
echo "初始化状�? $([ -f "$PROJECT_DIR/.github/.issue-initialized" ] && echo '已初始化' || echo '未初始化')"

# GitHub remote
remote_url=$(git -C "$PROJECT_DIR" remote get-url origin 2>/dev/null)
echo "Remote: $remote_url"

skill_log "INFO" "[inject] issue-init context injected"
