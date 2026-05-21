#!/bin/bash
# 23-worktree-create: 通过 --worktree 或 isolation: "worktree" 创建 worktree 时触发
# 任何非零退出代码都会导致 worktree 创建失败

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

log "INFO" "WorktreeCreate triggered"

# 示例: 指定 worktree 路径
# hook_output 0 '{"hookSpecificOutput":{"hookEventName":"WorktreeCreate","worktreePath":"/custom/path"}}'

dispatch_to_skill "23" || true
hook_output 0 '{}'
