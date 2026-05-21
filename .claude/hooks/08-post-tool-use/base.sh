#!/bin/bash
# 08-post-tool-use: 工具调用成功后触发 (每个工具调用)
# 输出: decision (block), reason, additionalContext

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

tool_name=$(json_get '.tool_name')
tool_response=$(json_get '.tool_response')

log "INFO" "tool=$tool_name"

# 示例: 在编辑文件后提醒
# if [ "$tool_name" = "Edit" ] || [ "$tool_name" = "Write" ]; then
#   hook_output 0 '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"文件已修改"}}'
# fi

# 示例: 阻止后续操作
# hook_output 2 '{"decision":"block","reason":"不允许在此目录操作"}'

dispatch_to_skill "08" || true

hook_output 0 '{}'
