#!/bin/bash
# otherdoc skill 加载时触发，注入日期和路径到上下文
TODAY=$(date +%Y-%m-%d)
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
DOC_DIR="$PROJECT_DIR/doc/otherDoc/$TODAY"

# 确保日期目录存在
mkdir -p "$DOC_DIR"

cat <<EOF
{"hookSpecificOutput":{"hookEventName":"InstructionsLoaded","additionalContext":"[otherdoc] 日期: $TODAY\n存储目录: $DOC_DIR/\n规则: 文件存入 $DOC_DIR/，文件名从内容提取关键词，扩展名 .md"}}
EOF
