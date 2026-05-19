#!/bin/bash
# 20-config-change: 配置文件在会话期间更改时触发
# 退出 2 可阻止配置更改生效 (除了 policy_settings)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

config_file=$(json_get '.config_file')
change_type=$(json_get '.change_type')

log "INFO" "file=$config_file type=$change_type"

# 示例: 阻止配置更改
# hook_output 2 '{"decision":"block","reason":"不允许在运行时更改此配置"}'

hook_output 0 '{}'
