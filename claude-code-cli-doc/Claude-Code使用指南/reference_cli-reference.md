# Claude-Code / Reference / Cli-Reference

> 来源: claudecn.com

# CLI 参考

整理 Claude Code 的 CLI（命令行）用法：常用命令、关键参数，以及系统提示词相关的高级开关。完整清单以你本机 `claude --help` 为准。

## 常用命令

| 命令 | 说明 | 示例 |
| --- | --- | --- |
| `claude` | 启动交互式会话（REPL） | `claude` |
| `claude "query"` | 启动会话并带初始提示 | `claude "解释这个项目的结构"` |
| `claude -p "query"` | Print/Headless 模式：输出到 stdout 后退出 | `claude -p "解释这个函数"` |
| `cat file | claude -p "query"` | 处理管道输入 | `cat logs.txt | claude -p "解释报错原因"` |
| `claude -c` | 继续当前目录下最近一次会话 | `claude -c` |
| `claude -r "" "query"` | 通过会话名/ID 恢复会话 | `claude -r "auth-refactor" "继续完成这个 PR"` |
| `claude update` | 更新到最新版本 | `claude update` |
| `claude mcp` | 配置 MCP 服务器 | `claude mcp` |

## 常用参数（选摘）

| 参数 | 说明 |
| --- | --- |
| `--print`, `-p` | Print/Headless 模式（适合脚本/CI） |
| `--output-format` | Print 模式输出格式：`text` / `json` / `stream-json` |
| `--input-format` | Print 模式输入格式：`text` / `stream-json` |
| `--continue`, `-c` | 继续最近会话 |
| `--resume`, `-r` | 恢复指定会话（或打开选择器） |
| `--fork-session` | 恢复会话时创建新 session（避免复写原会话） |
| `--session-id` | 指定会话 ID（UUID） |
| `--no-session-persistence` | Print 模式不落盘、不保存会话 |
| `--model` | 指定当前会话模型（支持别名如 `sonnet` / `opus`） |
| `--agent` | 指定当前会话 agent（覆盖 settings 中的 `agent`） |
| `--agents` | 通过 JSON 动态定义子代理 |
| `--permission-mode` | 以指定权限模式启动 |
| `--allowedTools` / `--disallowedTools` | 允许/禁止在不提示的情况下使用的工具规则 |
| `--tools` | 限制 Claude 可用的内置工具集合 |
| `--dangerously-skip-permissions` | 跳过所有权限确认（高风险） |
| `--allow-dangerously-skip-permissions` | 仅“允许出现绕过模式选项”，不自动启用（可与 `--permission-mode` 组合） |
| `--append-system-prompt` | 追加系统提示词（保留默认 Claude Code 能力） |
| `--append-system-prompt-file` | 从文件追加系统提示词（仅 Print 模式） |
| `--system-prompt` | 替换整个系统提示词（会移除默认 Claude Code 指令） |
| `--system-prompt-file` | 从文件替换系统提示词（仅 Print 模式） |
| `--verbose` | 输出更详细的 turn-by-turn 日志（便于调试） |
| `--version`, `-v` | 输出版本号 |

`--allowedTools` 的规则语法与 `settings.json` 的 permissions 一致；建议先读「配置参考」里的“权限规则语法”，避免误把 `Bash(*)` 当成“匹配所有 Bash”。

## 系统提示词相关参数（4 种）
这 4 个参数看起来相似，但语义不同：

| 参数 | 行为 | 模式 |
| --- | --- | --- |
| `--system-prompt` | **替换**默认系统提示词 | 交互 + Print |
| `--system-prompt-file` | 从文件**替换**默认系统提示词 | Print |
| `--append-system-prompt` | **追加**到默认系统提示词之后 | 交互 + Print |
| `--append-system-prompt-file` | 从文件**追加**到默认系统提示词之后 | Print |

推荐优先使用 “append” 两个参数：它们能在保留 Claude Code 默认能力的同时，叠加你的团队规则。

`--system-prompt` 与 `--system-prompt-file` 互斥；append 参数可以与替换参数组合使用。

## --agents（子代理）格式

`--agents` 接受一个 JSON 对象：键是子代理名称，值包含 `description`、`prompt` 等字段；可选字段还包括 `tools`、`model`。

```bash
claude --agents '{
  "code-reviewer": {
    "description": "做代码审查，重点关注质量/安全/最佳实践",
    "prompt": "You are a senior code reviewer. Focus on code quality, security, and best practices.",
    "tools": ["Read", "Grep", "Glob", "Bash"],
    "model": "sonnet"
  }
}'
```

## 下一步
[配置参考settings.json、权限与环境变量
](../settings/)[Amazon Bedrock第三方提供商配置示例
](../amazon-bedrock/)
