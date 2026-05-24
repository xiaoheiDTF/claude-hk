#!/bin/bash
# 003-2-issue 巡检：检�?gh CLI + 草稿目录
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="003-2-issue"
source "$PROJECT_DIR/\.codex/skills/log.sh"

if ! command -v gh &>/dev/null; then
  skill_log "WARN" "[check] gh CLI 不可�?
  exit 1
fi

mkdir -p "$PROJECT_DIR/doc/issues/drafts" "$PROJECT_DIR/doc/issues/templates"
skill_log "INFO" "[check] gh CLI 可用，目录已就绪"
