#!/bin/bash
# 07-permission-denied: 自动模式分类器拒绝工具调用时触发 (每个拒绝)
# 输出: retry: true 告诉模型它可以重试

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

tool_name=$(json_get '.tool_name')

log "INFO" "tool=$tool_name"

# 示例: 允许重试
# hook_output 0 '{"hookSpecificOutput":{"hookEventName":"PermissionDenied","retry":true}}'

hook_output 0 '{}'
