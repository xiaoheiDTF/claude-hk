#!/bin/bash
# 003-5-issue-fix 巡检：检�?git
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="003-5-issue-fix"
source "$PROJECT_DIR/\.codex/skills/log.sh"

if command -v git &>/dev/null; then
  skill_log "INFO" "[check] git 可用"
  exit 0
else
  skill_log "WARN" "[check] git 不可�?
  exit 1
fi
