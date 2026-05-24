#!/bin/bash
# 24-worktree-remove: Worktree 被移除时触发 (会话退出或 subagent 完成)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

worktree_path=$(json_get '.worktree_path')

log "INFO" "path=$worktree_path"

dispatch_to_skill "24" || true
hook_output 0 '{}'
