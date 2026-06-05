#!/bin/bash
# otherdoc on_load: skill 加载时注入动态上下文
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="002-1-doc-otherdoc"
source "$PROJECT_DIR/.claude/skills/log.sh"

TODAY=$(date +%Y-%m-%d)
DOC_DIR="$PROJECT_DIR/doc/otherDoc/$TODAY"

source "$PROJECT_DIR/.claude/scripts/ensure_dirs.sh"
ensure_skill_dirs "otherDoc"
mkdir -p "$DOC_DIR"

skill_log "INFO" "日期: $TODAY"
skill_log "INFO" "存储目录: $DOC_DIR"

CONTEXT="[otherdoc] 日期: $TODAY\n存储目录: $DOC_DIR/"

# 补充上下文：从 03-user-prompt-submit/ 加载学习进化内容
source "$PROJECT_DIR/.claude/skills/_load_supplementary.sh"
_load_supplementary "$PROJECT_DIR/.claude/skills/${SKILL_TAG}"
if [ -n "$SUPPLEMENTARY_JSON" ]; then
  CONTEXT="${CONTEXT}\n\n---\n\n${SUPPLEMENTARY_JSON}"
fi

echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"$CONTEXT\"}}"
