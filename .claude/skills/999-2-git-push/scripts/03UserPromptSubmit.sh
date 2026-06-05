#!/bin/bash
# git-push on_load: skill 加载时注入动态上下文
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="999-2-git-push"
source "$PROJECT_DIR/.claude/skills/log.sh"

BRANCH=$(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null || echo "unknown")

skill_log "INFO" "skill 调用: git-push | 分支: $BRANCH"

CONTEXT=""

# 补充上下文：从 03-user-prompt-submit/ 加载学习进化内容
source "$PROJECT_DIR/.claude/skills/_load_supplementary.sh"
_load_supplementary "$PROJECT_DIR/.claude/skills/${SKILL_TAG}"
if [ -n "$SUPPLEMENTARY_JSON" ]; then
  CONTEXT="${CONTEXT}\n\n---\n\n${SUPPLEMENTARY_JSON}"
fi

echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"$CONTEXT\"}}"
