#!/bin/bash
# 03-user-prompt-submit: 用户提交提示后、Claude 处理之前触发 (每轮一次)
# 退出 2 可阻止提示处理并从上下文中删除提示

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

prompt=$(json_get '.prompt')

log "INFO" "prompt=$prompt"

# 示例: 阻止特定提示
# if echo "$prompt" | grep -qi "dangerous"; then
#   log "WARN" "Blocked prompt"
#   hook_output 2 '{"decision":"block","reason":"Prompt contains dangerous content"}'
# fi

hook_output 0 '{}'
