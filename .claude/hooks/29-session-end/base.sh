#!/bin/bash
# 29-session-end: 会话终止时触发 (每个会话一次)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

session_id=$(json_get '.session_id')
reason=$(json_get '.reason')

log "INFO" "session=$session_id reason=$reason"

# 示例: 清理临时文件或资源
# rm -rf /tmp/claude-session-"$session_id"

hook_output 0 '{}'
