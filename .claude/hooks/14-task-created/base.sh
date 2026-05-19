#!/bin/bash
# 14-task-created: 通过 TaskCreate 创建任务时触发 (每次创建)
# 退出 2 可回滚任务创建

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

task_id=$(json_get '.task_id')
subject=$(json_get '.subject')

log "INFO" "id=$task_id subject=$subject"

# 示例: 阻止特定任务创建
# hook_output 2 '{"continue":false,"stopReason":"不允许创建此类任务"}'

hook_output 0 '{}'
