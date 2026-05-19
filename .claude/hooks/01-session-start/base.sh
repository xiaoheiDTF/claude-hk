#!/bin/bash
# 01-session-start: 会话开始或恢复时触发 (每个会话一次)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

source_type=$(json_get '.source')
log "INFO" "source=$source_type"

# 首次运行 → 触发统一初始化
[ ! -f "$CLAUDE_PROJECT_DIR/.claude/.initialized" ] && bash "$CLAUDE_PROJECT_DIR/.claude/init.sh"

# 示例: 持久化环境变量
# if [ -n "$CLAUDE_ENV_FILE" ]; then
#   echo 'export NODE_ENV=development' >> "$CLAUDE_ENV_FILE"
# fi

hook_output 0 '{}'
