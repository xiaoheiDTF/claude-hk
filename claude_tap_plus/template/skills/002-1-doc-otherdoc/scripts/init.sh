#!/bin/bash
# 002-1-doc-otherdoc init: 确保输出目录存在
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="002-1-doc-otherdoc"
source "$PROJECT_DIR/.claude/skills/log.sh"

mkdir -p "$PROJECT_DIR/doc/otherDoc/$(date +%Y-%m-%d)"
skill_log "INFO" "[init] otherDoc 目录已就绪"
