#!/bin/bash
# 19-instructions-loaded: CLAUDE.md �?\.codex/rules/*.md 加载到上下文时触�?

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

file_path=$(json_get '.file_path')

log "INFO" "file=$file_path"

dispatch_to_skill "19" || true
hook_output 0 '{}'
