#!/bin/bash
# 12-subagent-start: Subagent 生成时触发 (每次生成)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

agent_type=$(json_get '.agent_type')

log "INFO" "agent_type=$agent_type"

hook_output 0 '{}'
