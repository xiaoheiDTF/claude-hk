#!/bin/bash
# 17-stop-failure: 回合因 API 错误结束时触发 (每次失败)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

error=$(json_get '.error')

log "ERROR" "API error=$error"

hook_output 0 '{}'
