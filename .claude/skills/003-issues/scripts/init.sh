#!/bin/bash
# 003-issues init: 检查 gh CLI 可用且已认证
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
source "$PROJECT_DIR/.claude/skills/log.sh"

if ! command -v gh &>/dev/null; then
  skill_log "WARN" "[init] gh CLI 未安装"
  exit 1
fi

if gh auth status &>/dev/null; then
  skill_log "INFO" "[init] gh CLI 已认证"
else
  skill_log "WARN" "[init] gh CLI 未认证，请运行 gh auth login"
  exit 1
fi
