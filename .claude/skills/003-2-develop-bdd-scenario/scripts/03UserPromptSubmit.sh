#!/bin/bash
# develop-bdd-scenario on_load: 加载 BDD 场景规范作为上下文
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="003-2-develop-bdd-scenario"
source "$PROJECT_DIR/.claude/skills/log.sh"

skill_log "INFO" "BDD 场景规范已加载"

CONTEXT=""

echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"$CONTEXT\"}}"
