# Claude-Code / Reference / Statusline

> 来源: claudecn.com

# Statusline（状态栏）

Statusline 用来在终端底部实时展示 Claude Code 会话信息（例如当前模型、目录、上下文占用等）。你可以在 Claude Code 里运行 `/statusline` 按提示配置。

## Statusline 输入（stdin JSON）

你的 statusline 命令会通过 stdin 收到一段 JSON（示例字段会随版本演进）：

- model.display_name：当前模型展示名
- workspace.current_dir：当前工作目录
- context_window.used_percentage / context_window.remaining_percentage：上下文窗口使用/剩余百分比（0–100）
- context_window.current_usage：最近一次请求的 token 使用（可能为 null）
## 最小示例：显示模型 + 上下文占用

```bash
#!/usr/bin/env bash
set -euo pipefail

input="$(cat)"
model="$(echo "$input" | jq -r '.model.display_name // "unknown"')"
used="$(echo "$input" | jq -r '.context_window.used_percentage // 0')"

echo "[$model] Context: ${used}%"
```

优先使用 `used_percentage` / `remaining_percentage`：它们是预计算值，逻辑更稳定；只有在你需要自定义口径时，再从 `current_usage` 自己算。

## 下一步
[速查表常用命令与快捷键
](../quick-reference)[Advent of Claude高频技巧合集
](../../workflows/advent-of-claude)
