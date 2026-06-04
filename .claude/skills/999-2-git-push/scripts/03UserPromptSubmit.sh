#!/bin/bash
# git-push on_load: skill 加载时注入动态上下文
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="999-2-git-push"
source "$PROJECT_DIR/.claude/skills/log.sh"

BRANCH=$(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null || echo "unknown")

skill_log "INFO" "skill 调用: git-push | 分支: $BRANCH"

CONTEXT=""

echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"$CONTEXT\"}}"
