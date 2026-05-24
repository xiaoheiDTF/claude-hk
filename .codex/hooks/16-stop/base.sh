#!/bin/bash
# 16-stop: Claude 完成响应时触�?(每轮一�?
# 退�?2 可防�?Claude 停止，继续对�?

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

log "INFO" "Claude finished response"

# �?hook 输入获取 session_id
SESSION_ID=$(json_get '.session_id')
log "INFO" "session_id=$SESSION_ID"

# 自动注册新增 skills
REGISTER_RESULT=$(bash "$SCRIPT_DIR/skill-register.sh")
if [ "$REGISTER_RESULT" = "UPDATED" ]; then
  log "INFO" "registry.conf updated with new skills"
fi

# 只运行当�?session 激活的 skill �?16Stop.sh
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
ACTIVE_FILE="$PROJECT_DIR/\.codex/skills/.active"

if [ -s "$ACTIVE_FILE" ] && [ -n "$SESSION_ID" ]; then
  source "$PROJECT_DIR/\.codex/skills/active.sh"

  # 精确获取当前 session �?skill（不再遍历所有）
  SKILL_NAME=$(active_get "$SESSION_ID")
  if [ -n "$SKILL_NAME" ]; then
    STOP_SCRIPT="$PROJECT_DIR/\.codex/skills/$SKILL_NAME/scripts/16Stop.sh"
    if [ -f "$STOP_SCRIPT" ]; then
      log "INFO" "Running 16Stop.sh for $SKILL_NAME (session=$SESSION_ID)"
      bash "$STOP_SCRIPT" "$SESSION_ID" 2>/dev/null
    fi
  fi
fi

hook_output 0 '{}'
