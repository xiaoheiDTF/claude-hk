#!/bin/bash
# 22-file-changed: 监视的文件在磁盘上更改时触发

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

file_path=$(json_get '.file_path')
change_type=$(json_get '.change_type')

log "INFO" "file=$file_path type=$change_type"

hook_output 0 '{}'
