#!/bin/bash
# otherdoc skill 加载时触发
TODAY=$(date +%Y-%m-%d)
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
DOC_DIR="$PROJECT_DIR/doc/otherDoc/$TODAY"

source "$PROJECT_DIR/.claude/scripts/ensure_dirs.sh"
ensure_skill_dirs "otherDoc"
mkdir -p "$DOC_DIR"

cat <<EOF
{"hookSpecificOutput":{"hookEventName":"InstructionsLoaded","additionalContext":"[otherdoc] 日期: $TODAY\n存储目录: $DOC_DIR/\n规则: 文件存入 $DOC_DIR/，文件名从内容提取关键词，扩展名 .md"}}
EOF
