#!/bin/bash
# 10-post-tool-batch: 完整批次并行工具调用解决后触发 (每批次一次)
# 退出 2 可在下一个模型调用之前停止代理循环

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

log "INFO" "batch completed"

# 示例: 批次完成后阻止后续操作
# hook_output 2 '{"decision":"block","reason":"批次执行超出限制"}'

hook_output 0 '{}'
