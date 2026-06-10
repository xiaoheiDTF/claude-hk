#!/bin/bash
# 13-subagent-stop: Subagent 完成时触发 (每次完成)
# 退出 2 可防止 subagent 停止

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

agent_type=$(json_get '.agent_type')
result=$(json_get '.result')

log "INFO" "agent_type=$agent_type"

# 示例: 阻止 subagent 停止，继续工作
# hook_output 2 '{"decision":"block","reason":"Subagent 尚未完成所有任务"}'

dispatch_to_skill "13" || true
hook_output 0 '{}'
