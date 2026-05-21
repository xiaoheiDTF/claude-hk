#!/bin/bash
# 003-6-issue-pr 巡检：检查 gh + git
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
source "$PROJECT_DIR/.claude/skills/log.sh"

if command -v gh &>/dev/null && command -v git &>/dev/null; then
  skill_log "INFO" "[check] gh + git 可用"
  exit 0
else
  skill_log "WARN" "[check] gh 或 git 不可用"
  exit 1
fi
