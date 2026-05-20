#!/bin/bash
# issues on_load: skill 加载时注入动态上下文
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="issues"
source "$PROJECT_DIR/.claude/skills/log.sh"

TODAY=$(date +%Y-%m-%d)

source "$PROJECT_DIR/.claude/scripts/ensure_dirs.sh"
ensure_skill_dirs "issues"

skill_log "INFO" "日期: $TODAY"
skill_log "INFO" "草稿目录: $PROJECT_DIR/doc/issues/drafts/"
skill_log "INFO" "模板目录: $PROJECT_DIR/doc/issues/templates/"

CONTEXT="[issues] 日期: $TODAY\n草稿目录: $PROJECT_DIR/doc/issues/drafts/\n模板目录: $PROJECT_DIR/doc/issues/templates/\n操作: 新建草稿、从模板创建、发布到 GitHub (gh CLI)"

echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"$CONTEXT\"}}"
