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

# 运行当前活跃 skill 的 16Stop.sh，传入 session_id
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
ACTIVE_FILE="$PROJECT_DIR/.claude/skills/.active"

if [ -s "$ACTIVE_FILE" ]; then
  source "$PROJECT_DIR/.claude/skills/active.sh"
  ACTIVE_SKILLS=$(active_skills)
  for skill_name in $ACTIVE_SKILLS; do
    STOP_SCRIPT="$PROJECT_DIR/.claude/skills/$skill_name/scripts/16Stop.sh"
    if [ -f "$STOP_SCRIPT" ]; then
      log "INFO" "Running 16Stop.sh for $skill_name (session=$SESSION_ID)"
      bash "$STOP_SCRIPT" "$SESSION_ID" 2>/dev/null
    fi
  done
fi

hook_output 0 '{}'
