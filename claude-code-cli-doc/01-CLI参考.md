# CLI 参考 - 启动参数与标志

> 来源: https://code.claude.com/docs/zh-CN/cli-reference

本文档列出 `claude` 命令的所有启动参数与使用技巧。

---

## 目录工作技巧

| 参数 | 用途 | 示例 |
|------|------|------|
| `--add-dir` | 为 Claude 添加额外的工作目录以读取和编辑文件。授予文件访问权限；大多数 `.claude/` 配置不会从这些目录中发现 | `claude --add-dir ../apps ../lib` |
| `--worktree`, `-w` | 在隔离的 git worktree 中启动 Claude，位于 `<repo>/.claude/worktrees/<name>`。如果未给出名称，则自动生成一个 | `claude -w feature-auth` |
| `--tmux` | 为 worktree 创建 tmux 会话。需要 `--worktree`。在可用时使用 iTerm2 原生窗格；传递 `--tmux=classic` 以使用传统 tmux | `claude -w feature-auth --tmux` |

## 代理与自定义配置

| 参数 | 用途 | 示例 |
|------|------|------|
| `--agent` | 为当前会话指定代理（覆盖 `agent` 设置） | `claude --agent my-custom-agent` |
| `--agents` | 通过 JSON 动态定义自定义 subagents | `claude --agents '{"reviewer":{"description":"Reviews code","prompt":"You are a code reviewer"}}'` |
| `--teammate-mode` | 设置 agent team 队友的显示方式：`auto`（默认）、`in-process` 或 `tmux` | `claude --teammate-mode in-process` |

## 权限模式技巧

| 参数 | 用途 | 示例 |
|------|------|------|
| `--permission-mode` | 以指定的权限模式开始。接受 `default`、`acceptEdits`、`plan`、`auto`、`dontAsk` 或 `bypassPermissions` | `claude --permission-mode plan` |
| `--allow-dangerously-skip-permissions` | 将 `bypassPermissions` 添加到 `Shift+Tab` 模式循环中而不启动它。允许您以不同的模式（如 `plan`）开始，稍后切换到 `bypassPermissions` | `claude --permission-mode plan --allow-dangerously-skip-permissions` |
| `--dangerously-skip-permissions` | 跳过权限提示。等同于 `--permission-mode bypassPermissions` | `claude --dangerously-skip-permissions` |
| `--allowedTools` | 无需提示权限即可执行的工具 | `"Bash(git log *)" "Bash(git diff *)" "Read"` |
| `--disallowedTools` | 从模型的上下文中删除的工具，无法使用 | `"Bash(git log *)" "Bash(git diff *)" "Edit"` |

## 提示与系统配置

| 参数 | 用途 | 示例 |
|------|------|------|
| `--system-prompt` | 用自定义文本替换整个系统提示 | `claude --system-prompt "You are a Python expert"` |
| `--system-prompt-file` | 从文件加载系统提示，替换默认提示 | `claude --system-prompt-file ./custom-prompt.txt` |
| `--append-system-prompt` | 将自定义文本附加到默认系统提示的末尾 | `claude --append-system-prompt "Always use TypeScript"` |
| `--append-system-prompt-file` | 从文件加载额外的系统提示文本并附加到默认提示 | `claude --append-system-prompt-file ./extra-rules.txt` |
| `--exclude-dynamic-system-prompt-sections` | 将每台机器的部分从系统提示移到第一条用户消息中。改进在运行相同任务的不同用户和机器之间的提示缓存重用 | `claude -p --exclude-dynamic-system-prompt-sections "query"` |

## 模型与输出控制

| 参数 | 用途 | 示例 |
|------|------|------|
| `--model` | 为当前会话设置模型，使用最新模型的别名（`sonnet` 或 `opus`）或模型的完整名称 | `claude --model claude-sonnet-4-6` |
| `--effort` | 为当前会话设置工作量级别。选项：`low`、`medium`、`high`、`xhigh`、`max`；可用级别取决于模型 | `claude --effort high` |
| `--fallback-model` | 当默认模型过载时启用自动回退到指定模型（仅打印模式） | `claude -p --fallback-model sonnet "query"` |
| `--betas` | 要包含在 API 请求中的 Beta 标头（仅限 API 密钥用户） | `claude --betas interleaved-thinking` |

## 打印模式（非交互式/脚本化）

| 参数 | 用途 | 示例 |
|------|------|------|
| `--print`, `-p` | 打印响应而不进入交互模式 | `claude -p "query"` |
| `--output-format` | 为打印模式指定输出格式（选项：`text`、`json`、`stream-json`） | `claude -p "query" --output-format json` |
| `--input-format` | 为打印模式指定输入格式（选项：`text`、`stream-json`） | `claude -p --output-format json --input-format stream-json` |
| `--json-schema` | 在代理完成其工作流后获得与 JSON Schema 匹配的验证 JSON 输出 | `claude -p --json-schema '{"type":"object","properties":{...}}' "query"` |
| `--max-turns` | 限制代理转数（仅打印模式）。达到限制时以错误退出 | `claude -p --max-turns 3 "query"` |
| `--max-budget-usd` | API 调用前停止的最大美元金额（仅打印模式） | `claude -p --max-budget-usd 5.00 "query"` |
| `--no-session-persistence` | 禁用会话持久化，以便会话不会保存到磁盘且无法恢复（仅打印模式） | `claude -p --no-session-persistence "query"` |

## 会话恢复与管理

| 参数 | 用途 | 示例 |
|------|------|------|
| `--resume`, `-r` | 按 ID 或名称恢复特定会话，或显示交互式选择器 | `claude --resume auth-refactor` |
| `--continue`, `-c` | 加载当前目录中最近的对话。包括使用 `/add-dir` 添加此目录的会话 | `claude --continue` |
| `--fork-session` | 恢复时，创建新的会话 ID 而不是重用原始 ID（与 `--resume` 或 `--continue` 一起使用） | `claude --resume abc123 --fork-session` |
| `--name`, `-n` | 为会话设置显示名称，显示在 `/resume` 和终端标题中 | `claude -n "my-feature-work"` |
| `--session-id` | 为对话使用特定的会话 ID（必须是有效的 UUID） | `claude --session-id "550e8400-e29b-41d4-a716-446655440000"` |
| `--from-pr` | 恢复链接到特定拉取请求的会话 | `claude --from-pr 123` |

## 远程与网络会话

| 参数 | 用途 | 示例 |
|------|------|------|
| `--remote` | 在 claude.ai 上创建新的网络会话，提供任务描述 | `claude --remote "Fix the login bug"` |
| `--teleport` | 在本地终端中恢复网络会话 | `claude --teleport` |
| `--remote-control`, `--rc` | 启动启用了 Remote Control 的交互式会话，以便您也可以从 claude.ai 或 Claude 应用控制它 | `claude --remote-control "My Project"` |
| `--remote-control-session-name-prefix` | 当未设置显式名称时，Remote Control 自动生成会话名称的前缀 | `claude remote-control --remote-control-session-name-prefix dev-box` |

## 调试与开发

| 参数 | 用途 | 示例 |
|------|------|------|
| `--debug` | 启用调试模式，可选类别过滤（例如，`"api,hooks"` 或 `"!statsig,!file"`） | `claude --debug "api,mcp"` |
| `--debug-file <path>` | 将调试日志写入特定文件路径。隐式启用调试模式 | `claude --debug-file /tmp/claude-debug.log` |
| `--verbose` | 启用详细日志记录，显示完整的逐轮输出 | `claude --verbose` |
| `--version`, `-v` | 输出版本号 | `claude -v` |

## MCP 与插件配置

| 参数 | 用途 | 示例 |
|------|------|------|
| `--mcp-config` | 从 JSON 文件或字符串加载 MCP 服务器（以空格分隔） | `claude --mcp-config ./mcp.json` |
| `--strict-mcp-config` | 仅使用来自 `--mcp-config` 的 MCP 服务器，忽略所有其他 MCP 配置 | `claude --strict-mcp-config --mcp-config ./mcp.json` |
| `--plugin-dir` | 仅为此会话从目录加载插件。每个标志采用一个路径。重复该标志以获取多个目录 | `claude --plugin-dir ./my-plugins` |
| `--channels` | MCP 服务器，其 channel 通知 Claude 应在此会话中侦听。以空格分隔的 `plugin:<name>@<marketplace>` 条目列表 | `claude --channels plugin:my-notifier@my-marketplace` |
| `--dangerously-load-development-channels` | 启用不在批准的允许列表中的 channels，用于本地开发 | `claude --dangerously-load-development-channels server:webhook` |

## 工具限制与简化模式

| 参数 | 用途 | 示例 |
|------|------|------|
| `--tools` | 限制 Claude 可以使用的内置工具。使用 `""` 禁用所有，`"default"` 表示全部，或工具名称如 `"Bash,Edit,Read"` | `claude --tools "Bash,Edit,Read"` |
| `--bare` | 最小模式：跳过 hooks、skills、plugins、MCP 服务器、自动内存和 CLAUDE.md 的自动发现，以便脚本化调用启动更快。Claude 可以访问 Bash、文件读取和文件编辑工具。设置 `CLAUDE_CODE_SIMPLE` | `claude --bare -p "query"` |
| `--disable-slash-commands` | 为此会话禁用所有 skills 和命令 | `claude --disable-slash-commands` |

## 其他实用参数

| 参数 | 用途 | 示例 |
|------|------|------|
| `--chrome` / `--no-chrome` | 启用/禁用 Chrome 浏览器集成以进行网络自动化和测试 | `claude --chrome` |
| `--ide` | 如果恰好有一个有效的 IDE 可用，则在启动时自动连接到 IDE | `claude --ide` |
| `--init` | 在会话前运行带有 `init` 匹配器的 Setup hooks（仅打印模式） | `claude -p --init "query"` |
| `--init-only` | 运行 Setup 和 `SessionStart` hooks，然后退出而不启动对话 | `claude --init-only` |
| `--maintenance` | 在会话前运行带有 `maintenance` 匹配器的 Setup hooks（仅打印模式） | `claude -p --maintenance "query"` |
| `--setting-sources` | 逗号分隔的设置源列表以加载（`user`、`project`、`local`） | `claude --setting-sources user,project` |
| `--settings` | 设置 JSON 文件的路径或 JSON 字符串以加载其他设置 | `claude --settings ./settings.json` |
| `--permission-prompt-tool` | 指定 MCP 工具以在非交互模式下处理权限提示 | `claude -p --permission-prompt-tool mcp_auth_tool "query"` |
| `--include-hook-events` | 在输出流中包含所有 hook 生命周期事件。需要 `--output-format stream-json` | `claude -p --output-format stream-json --include-hook-events "query"` |
| `--include-partial-messages` | 在输出中包含部分流事件。需要 `--print` 和 `--output-format stream-json` | `claude -p --output-format stream-json --include-partial-messages "query"` |
| `--replay-user-messages` | 从 stdin 重新发出用户消息到 stdout 以进行确认。需要 `--input-format stream-json` 和 `--output-format stream-json` | `claude -p --input-format stream-json --output-format stream-json --replay-user-messages` |
