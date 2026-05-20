#!/bin/bash
# git-commit on_stop: skill 完成后的收尾操作
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="git-commit"
source "$PROJECT_DIR/.claude/skills/log.sh"

BRANCH=$(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null || echo "unknown")

skill_log "INFO" "[stop] skill completed | branch: $BRANCH"
skill_log "INFO" "[stop] skill lifecycle end"
