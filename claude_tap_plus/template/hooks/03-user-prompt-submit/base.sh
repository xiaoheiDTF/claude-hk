#!/bin/bash
# 03-user-prompt-submit: 用户提交提示后、Claude 处理之前触发 (每轮一次)
# 退出 2 可阻止提示处理并从上下文中删除提示

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

prompt=$(json_get '.prompt')
session_id=$(json_get '.session_id')

log "INFO" "prompt=$prompt session=$session_id"

# skill 注入：检测 /xxx 格式调用（首字符必须是 /）
OUTPUT=$(bash "$SCRIPT_DIR/skill-inject.sh" "$prompt" "$session_id" 2>/dev/null)
if [ -n "$OUTPUT" ]; then
  # 第一行：skill_name|args
  META=$(echo "$OUTPUT" | head -1)
  # 第二行起：CONTEXT（完整 hook JSON）
  CONTEXT=$(echo "$OUTPUT" | tail -n +2)

  SKILL_NAME=$(echo "$META" | cut -d'|' -f1)
  SKILL_ARGS=$(echo "$META" | cut -d'|' -f2)

  log "INFO" "Skill matched: $SKILL_NAME | session: $session_id | args: $SKILL_ARGS | context: $CONTEXT"
  hook_output 0 "$CONTEXT"
fi

hook_output 0 '{}'
