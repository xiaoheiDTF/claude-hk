#!/bin/bash
# 003-4-issue-claim 巡检：检�?gh CLI
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="003-4-issue-claim"
source "$PROJECT_DIR/\.codex/skills/log.sh"

if command -v gh &>/dev/null; then
  skill_log "INFO" "[check] gh CLI 可用"
  exit 0
else
  skill_log "WARN" "[check] gh CLI 不可�?
  exit 1
fi
