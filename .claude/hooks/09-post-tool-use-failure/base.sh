#!/bin/bash
# 09-post-tool-use-failure: 工具调用失败后触发 (每次失败)
# 输出: decision (block), reason, additionalContext

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

tool_name=$(json_get '.tool_name')
error=$(json_get '.error')

log "WARN" "tool=$tool_name error=$error"

# 示例: 记录失败并提供建议
# hook_output 0 '{"hookSpecificOutput":{"hookEventName":"PostToolUseFailure","additionalContext":"工具调用失败，请检查参数"}}'

hook_output 0 '{}'
