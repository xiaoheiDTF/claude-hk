#!/bin/bash
# 28-elicitation-result: 用户响应 MCP elicitation 后、响应发送回服务器之前触发
# 退出 2 可阻止响应 (操作变为 decline)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

action=$(json_get '.action')
content=$(json_get '.content')

log "INFO" "action=$action"

# 示例: 阻止响应
# hook_output 2 '{"hookSpecificOutput":{"hookEventName":"ElicitationResult","action":"decline"}}'

hook_output 0 '{}'
