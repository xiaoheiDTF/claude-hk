#!/bin/bash
# 16-stop: Claude 完成响应时触发 (每轮一次)
# 退出 2 可防止 Claude 停止，继续对话

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

log "INFO" "Claude finished response"

# 自动注册新增 skills
REGISTER_RESULT=$(bash "$SCRIPT_DIR/skill-register.sh")
if [ "$REGISTER_RESULT" = "UPDATED" ]; then
  log "INFO" "registry.conf updated with new skills"
fi

hook_output 0 '{}'
