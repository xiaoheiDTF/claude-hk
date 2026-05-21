#!/bin/bash
# 005-git-commit init_check: 检查 git 可用
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="005-git-commit"
source "$PROJECT_DIR/.claude/skills/log.sh"

if command -v git &>/dev/null; then
  skill_log "INFO" "[check] git 可用"
  exit 0
else
  skill_log "WARN" "[check] git 不可用"
  exit 1
fi
