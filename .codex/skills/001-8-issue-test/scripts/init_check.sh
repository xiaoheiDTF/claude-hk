#!/bin/bash
# 001-8-issue-test 巡检：检�?gh + git
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="001-8-issue-test"
source "$PROJECT_DIR/\.codex/skills/log.sh"

if command -v gh &>/dev/null && command -v git &>/dev/null; then
  skill_log "INFO" "[check] gh + git 可用"
  exit 0
else
  skill_log "WARN" "[check] gh �?git 不可�?
  exit 1
fi
