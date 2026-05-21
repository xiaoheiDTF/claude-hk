#!/bin/bash
# 27-elicitation: MCP 服务器在工具调用期间请求用户输入时触发
# 输出: action (accept/decline/cancel), content

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

server=$(json_get '.server')
tool=$(json_get '.tool')

log "INFO" "server=$server tool=$tool"

# 示例: 自动接受
# hook_output 0 '{"hookSpecificOutput":{"hookEventName":"Elicitation","action":"accept","content":"自动确认"}}'

# 示例: 拒绝
# hook_output 2 '{"hookSpecificOutput":{"hookEventName":"Elicitation","action":"decline"}}'

dispatch_to_skill "27" || true
hook_output 0 '{}'
