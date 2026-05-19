#!/bin/bash
# 02-setup: 使用 --init-only 启动，或在 -p 模式中使用 --init 或 --maintenance 时触发

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

log "INFO" "Setup triggered"

hook_output 0 '{}'
