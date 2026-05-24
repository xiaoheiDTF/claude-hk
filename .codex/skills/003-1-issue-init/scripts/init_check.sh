#!/bin/bash
# 003-1-issue-init 巡检：检查标签体系完整�?
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="003-1-issue-init"
source "$PROJECT_DIR/\.codex/skills/log.sh"

if [ ! -f "$PROJECT_DIR/.github/.issue-initialized" ]; then
  skill_log "WARN" "[check] 标签体系未初始化，请运行 /003-1-issue-init"
  exit 1
fi

if command -v gh &>/dev/null; then
  skill_log "INFO" "[check] 标签体系已初始化"
  exit 0
else
  skill_log "WARN" "[check] gh CLI 不可�?
  exit 1
fi
