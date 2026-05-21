#!/bin/bash
# 004-git-push init_check: 检查 git 可用
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="004-git-push"
source "$PROJECT_DIR/.claude/skills/log.sh"

if command -v git &>/dev/null; then
  skill_log "INFO" "[check] git 可用"
  exit 0
else
  skill_log "WARN" "[check] git 不可用"
  exit 1
fi
