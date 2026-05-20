#!/bin/bash
# 03-user-prompt-submit: 用户提交提示后、Claude 处理之前触发 (每轮一次)
# 退出 2 可阻止提示处理并从上下文中删除提示

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

prompt=$(json_get '.prompt')

log "INFO" "prompt=$prompt"

hook_output 0 '{}'
