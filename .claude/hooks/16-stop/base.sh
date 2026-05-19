#!/bin/bash
# 16-stop: Claude 完成响应时触发 (每轮一次)
# 退出 2 可防止 Claude 停止，继续对话

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

log "INFO" "Claude finished response"

# 示例: 防止 Claude 停止，让其继续工作
# hook_output 2 '{"decision":"block","reason":"还有未完成的任务"}'

hook_output 0 '{}'
