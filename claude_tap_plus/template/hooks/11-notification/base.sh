#!/bin/bash
# 11-notification: Claude Code 发送通知时触发 (每次通知)
# 字段: message, notification_type

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

message=$(json_get '.message')
notification_type=$(json_get '.notification_type')

log "INFO" "type=$notification_type message=$message"

dispatch_to_skill "11" || true
hook_output 0 '{}'
