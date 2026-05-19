#!/bin/bash
# 15-task-completed: 任务被标记为已完成时触发 (每次完成)
# 退出 2 可防止任务被标记为已完成

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

task_id=$(json_get '.task_id')
subject=$(json_get '.subject')

log "INFO" "id=$task_id subject=$subject"

# 示例: 阻止过早标记完成
# hook_output 2 '{"continue":false,"stopReason":"任务验证未通过"}'

hook_output 0 '{}'
