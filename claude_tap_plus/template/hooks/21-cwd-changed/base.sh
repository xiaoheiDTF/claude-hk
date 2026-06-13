#!/bin/bash
# 21-cwd-changed: 工作目录更改时触发 (如 Claude 执行 cd 命令)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

old_cwd=$(json_get '.old_cwd')
new_cwd=$(json_get '.new_cwd')

log "INFO" "old=$old_cwd new=$new_cwd"

dispatch_to_skill "21" || true
hook_output 0 '{}'
