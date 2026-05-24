#!/bin/bash
# 05-pre-tool-use: 工具调用执行之前触发 (每个工具调用)
# 最丰富的决定控�? allow/deny/ask/defer
# 决定优先�? deny > defer > ask > allow

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

tool_name=$(json_get '.tool_name')
tool_input=$(json_get '.tool_input')

log "INFO" "tool=$tool_name"

# 示例: 阻止危险 Bash 命令
# if [ "$tool_name" = "Bash" ]; then
#   if echo "$tool_input" | grep -qi "rm -rf"; then
#     log "WARN" "Blocked dangerous command"
#     hook_output 2 '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"Dangerous command blocked"}}'
#   fi
# fi

# 示例: 自动允许只读操作
# if [ "$tool_name" = "Read" ] || [ "$tool_name" = "Grep" ] || [ "$tool_name" = "Glob" ]; then
#   hook_output 0 '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","permissionDecisionReason":"Read-only tool"}}'
# fi

# 示例: 修改工具输入
# hook_output 0 '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","updatedInput":{"file_path":"/safe/path"}}}'

# 示例: 添加上下文提�?
# hook_output 0 '{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"当前为生产环境，请谨慎操�?}}'

# A 层：工具级白名单拦截（在 dispatch 之前执行�?
source "$CLAUDE_PROJECT_DIR/\.codex/skills/enforce_boundary.sh"

# B 层：skill 按需路径级拦�?
dispatch_to_skill "05" || true

hook_output 0 '{}'
