#!/bin/bash
# 002-1-doc-otherdoc init_check: 检查输出目录存在
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="002-1-doc-otherdoc"
source "$PROJECT_DIR/.claude/skills/log.sh"

if [ -d "$PROJECT_DIR/doc/otherDoc" ]; then
  skill_log "INFO" "[check] otherDoc 目录存在"
  exit 0
else
  skill_log "WARN" "[check] otherDoc 目录缺失"
  exit 1
fi
