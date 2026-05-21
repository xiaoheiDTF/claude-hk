#!/bin/bash
# 003-issues init_check: 检查 gh CLI 可用
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
source "$PROJECT_DIR/.claude/skills/log.sh"

if command -v gh &>/dev/null; then
  skill_log "INFO" "[check] gh CLI 可用"
  exit 0
else
  skill_log "WARN" "[check] gh CLI 不可用"
  exit 1
fi
