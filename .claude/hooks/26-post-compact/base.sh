#!/bin/bash
# 26-post-compact: 上下文压缩完成后触发

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

log "INFO" "Context compressed"

hook_output 0 '{}'
