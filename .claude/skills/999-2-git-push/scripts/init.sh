#!/bin/bash
# 999-2-git-push init: 检查 git 配置
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="999-2-git-push"
source "$PROJECT_DIR/.claude/skills/log.sh"

if ! command -v git &>/dev/null; then
  skill_log "WARN" "[init] git 未安装"
  exit 1
fi

user_name=$(git config user.name 2>/dev/null)
user_email=$(git config user.email 2>/dev/null)

if [ -z "$user_name" ] || [ -z "$user_email" ]; then
  skill_log "WARN" "[init] git user.name 或 user.email 未配置"
  exit 1
fi

skill_log "INFO" "[init] git 配置正常: $user_name <$user_email>"
