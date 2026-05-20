#!/bin/bash
# issues on_stop: skill 完成后的收尾操作
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="issues"
source "$PROJECT_DIR/.claude/skills/log.sh"

skill_log "INFO" "[stop] skill completed"
skill_log "INFO" "[stop] skill lifecycle end"
