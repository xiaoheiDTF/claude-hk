#!/bin/bash
# 25-pre-compact: 上下文压缩之前触发
# 退出 2 可阻止压缩

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

log "INFO" "About to compress context"

# 示例: 阻止压缩
# hook_output 2 '{"decision":"block","reason":"当前上下文不应被压缩"}'

hook_output 0 '{}'
