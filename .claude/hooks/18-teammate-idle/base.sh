#!/bin/bash
# 18-teammate-idle: Agent team 队友即将空闲时触发
# 退出 2 可防止队友空闲 (队友继续工作)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

teammate=$(json_get '.teammate')

log "INFO" "teammate=$teammate"

# 示例: 让队友继续工作
# hook_output 2 '{"continue":false,"stopReason":"还有分配给该队友的任务"}'

dispatch_to_skill "18" || true
hook_output 0 '{}'
