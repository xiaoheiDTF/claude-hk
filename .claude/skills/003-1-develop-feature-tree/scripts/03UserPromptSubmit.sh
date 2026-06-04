#!/bin/bash
# develop-feature-tree on_load: 加载功能点与功能树规范作为上下文
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="003-1-develop-feature-tree"
source "$PROJECT_DIR/.claude/skills/log.sh"

skill_log "INFO" "功能点与功能树规范已加载"

CONTEXT=""

echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"$CONTEXT\"}}"
