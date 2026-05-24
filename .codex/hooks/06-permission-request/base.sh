#!/bin/bash
# 06-permission-request: 权限对话框出现时触发 (每个工具调用)
# 输出: decision.behavior (allow/deny)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

tool_name=$(json_get '.tool_name')

log "INFO" "tool=$tool_name"

# 示例: 自动允许特定工具
# if [ "$tool_name" = "Read" ]; then
#   hook_output 0 '{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"allow"}}}'
# fi

# 示例: 自动拒绝
# hook_output 2 '{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny"}}}'

dispatch_to_skill "06" || true
hook_output 0 '{}'
