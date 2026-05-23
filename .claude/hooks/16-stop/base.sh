#!/bin/bash
# 16-stop: Claude 完成响应时触发 (每轮一次)
# 退出 2 可防止 Claude 停止，继续对话

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

log "INFO" "Claude finished response"

# 从 hook 输入获取 session_id
SESSION_ID=$(json_get '.session_id')
log "INFO" "session_id=$SESSION_ID"

# 自动注册新增 skills
REGISTER_RESULT=$(bash "$SCRIPT_DIR/skill-register.sh")
if [ "$REGISTER_RESULT" = "UPDATED" ]; then
  log "INFO" "registry.conf updated with new skills"
fi

# 只运行当前 session 激活的 skill 的 16Stop.sh
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
ACTIVE_FILE="$PROJECT_DIR/.claude/skills/.active"

if [ -s "$ACTIVE_FILE" ] && [ -n "$SESSION_ID" ]; then
  source "$PROJECT_DIR/.claude/skills/active.sh"

  # 精确获取当前 session 的 skill（不再遍历所有）
  SKILL_NAME=$(active_get "$SESSION_ID")
  if [ -n "$SKILL_NAME" ]; then
    STOP_SCRIPT="$PROJECT_DIR/.claude/skills/$SKILL_NAME/scripts/16Stop.sh"
    if [ -f "$STOP_SCRIPT" ]; then
      log "INFO" "Running 16Stop.sh for $SKILL_NAME (session=$SESSION_ID)"
      bash "$STOP_SCRIPT" "$SESSION_ID" 2>/dev/null
    fi
  fi
fi

# Windows: bring terminal to foreground when Claude finishes
if [ "$OS_TYPE" = "windows" ] && [ -f "$SCRIPT_DIR/task-complete-notify.sh" ]; then
  source "$SCRIPT_DIR/task-complete-notify.sh"
fi

hook_output 0 '{}'
