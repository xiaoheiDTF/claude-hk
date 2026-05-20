#!/bin/bash
# issues skill 加载时触发
TODAY=$(date +%Y-%m-%d)
PROJECT_DIR="$CLAUDE_PROJECT_DIR"

source "$PROJECT_DIR/.claude/scripts/ensure_dirs.sh"
ensure_skill_dirs "issues"

cat <<EOF
{"hookSpecificOutput":{"hookEventName":"InstructionsLoaded","additionalContext":"[issues] 日期: $TODAY\n草稿目录: $PROJECT_DIR/doc/issues/drafts/\n模板目录: $PROJECT_DIR/doc/issues/templates/\n操作: 新建草稿、从模板创建、发布到 GitHub (gh CLI)"}}
EOF
