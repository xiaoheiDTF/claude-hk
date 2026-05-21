#!/bin/bash
# otherdoc on_load: skill 加载时注入动态上下文
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="002-otherdoc"
source "$PROJECT_DIR/.claude/skills/log.sh"

TODAY=$(date +%Y-%m-%d)
DOC_DIR="$PROJECT_DIR/doc/otherDoc/$TODAY"

source "$PROJECT_DIR/.claude/scripts/ensure_dirs.sh"
ensure_skill_dirs "otherDoc"
mkdir -p "$DOC_DIR"

skill_log "INFO" "日期: $TODAY"
skill_log "INFO" "存储目录: $DOC_DIR"

CONTEXT="[otherdoc] 日期: $TODAY\n存储目录: $DOC_DIR/\n规则: 文件存入 $DOC_DIR/，文件名从内容提取关键词，扩展名 .md"

echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"$CONTEXT\"}}"
