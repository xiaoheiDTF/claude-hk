#!/bin/bash
# 001-2-issue 巡检：检查 gh CLI + 草稿目录
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="001-2-issue"
source "$PROJECT_DIR/.claude/skills/log.sh"

if ! command -v gh &>/dev/null; then
  skill_log "WARN" "[check] gh CLI 不可用"
  exit 1
fi

mkdir -p "$PROJECT_DIR/doc/issues/drafts" "$PROJECT_DIR/doc/issues/templates"
skill_log "INFO" "[check] gh CLI 可用，目录已就绪"
