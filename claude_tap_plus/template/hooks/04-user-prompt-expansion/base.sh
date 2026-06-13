#!/bin/bash
# 04-user-prompt-expansion: 用户输入的命令展开为提示之前触发 (每轮一次)
# 退出 2 可阻止展开

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

command_val=$(json_get '.command')
expanded_prompt=$(json_get '.expanded_prompt')

log "INFO" "command=$command_val"

# 示例: 阻止特定命令展开
# if [ "$command_val" = "/dangerous" ]; then
#   log "WARN" "Blocked command expansion: $command_val"
#   hook_output 2 '{"decision":"block","reason":"Command expansion blocked"}'
# fi

dispatch_to_skill "04" || true
hook_output 0 '{}'
