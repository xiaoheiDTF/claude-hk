# Claude-Code / Reference / Quick-Reference

> 来源: claudecn.com

# Claude Code 速查表

用于“抬手就查”：把常用快捷键、命令/Skills 与 CLI 参数浓缩成一份表格。不同版本可能略有差异，以你本机 `claude` 的 `/help` 与官方文档为准。

## 键盘快捷键

| 快捷键 | 用途 |
| --- | --- |
| `!` | 直接执行 Bash 命令并把输出注入上下文 |
| `Esc` | 中断当前思考/执行 |
| `Esc Esc` | 回到更早的检查点（rewind） |
| `Ctrl+G` | 在默认文本编辑器中编辑当前输入（如版本支持） |
| `Ctrl+R` | 反向搜索历史提示词 |
| `Ctrl+S` | 暂存当前提示词草稿 |
| `Shift+Tab` | 循环切换权限模式（含 Plan Mode） |
| `Tab` / `Enter` | 接受提示词建议（如版本支持） |

## 常用命令（内置 + Skills）

提示：在 Claude Code 里输入 `/` 可以看到完整命令列表，并支持按字母过滤；你自己写的 Skills 也会以 `/skill-name` 的形式出现在列表中。

| 命令 | 用途 | 延伸阅读 |
| --- | --- | --- |
| `/help` | 查看内置命令与帮助 | — |
| `/init` | 生成/更新 `CLAUDE.md`，让 Claude 了解项目 | [上下文管理](https://claudecn.com/docs/claude-code/workflows/context-management/) |
| `/clear` | 清理当前对话上下文 | [上下文管理](https://claudecn.com/docs/claude-code/workflows/context-management/) |
| `/compact` | 压缩/总结当前上下文（如版本支持） | [Hooks 系统](https://claudecn.com/docs/claude-code/advanced/hooks/) |
| `/config` | 查看/调整配置（部分配置也可在 `.claude/settings.json`） | [配置参考](https://claudecn.com/docs/claude-code/reference/settings/) |
| `/permissions` | 查看/调整权限（如版本支持） | [配置参考](https://claudecn.com/docs/claude-code/reference/settings/) |
| `/mcp` | 管理 MCP 连接与认证（如版本支持） | [MCP 服务器](https://claudecn.com/docs/claude-code/advanced/mcp-servers/) |
| `/model` | 切换模型（如版本支持） | — |
| `/sandbox` | 设置/启用沙箱与权限边界 | [安全指南](https://claudecn.com/docs/claude-code/reference/security/) |
| `/hooks` | 配置 Hooks（如版本支持） | [Hooks 系统](https://claudecn.com/docs/claude-code/advanced/hooks/) |
| `/commit` | 智能生成提交信息并提交 | [Git 集成](https://claudecn.com/docs/claude-code/workflows/git-integration/) |
| `/resume` | 恢复历史会话（如版本支持） | [基础使用](https://claudecn.com/docs/claude-code/getting-started/basic-usage/) |
| `/rename` | 给当前会话命名（如版本支持） | — |
| `/export` | 导出会话为 Markdown（如版本支持） | — |
| `/vim` | Vim 编辑模式（如版本支持） | — |
| `/context` | 查看上下文/Token 占用（如版本支持） | — |
| `/stats` | 查看使用统计（如版本支持） | — |
| `/usage` | 查看额度/限额（如版本支持） | — |
| `/statusline` | 配置终端状态栏（如版本支持） | [Statusline](https://claudecn.com/docs/claude-code/reference/statusline/) |
| `/tasks` | 查看/管理后台任务（如版本支持） | — |
| `/todos` | 列出 TODO 项（如版本支持） | — |
| `/theme` | 切换主题（如版本支持） | — |

## 常用 CLI 参数

| 参数 | 用途 |
| --- | --- |
| `claude ""` | 一次性执行任务 |
| `claude -c` / `claude --continue` | 继续上次会话 |
| `claude --resume` | 选择并恢复历史会话 |
| `claude --permission-mode ` | 指定权限模式（如 `plan`、`acceptEdits`） |
| `claude -p ""` | Headless/Print 模式：把输出写到 stdout，适合脚本与 CI |
| `claude -p --append-system-prompt-file  ""` | 从文件追加系统提示词（适合版本化团队规则） |
| `claude --dangerously-skip-permissions` | 跳过权限确认（高风险） |
| `claude --teleport` | 把 claude.ai 的远程会话拉回本地（如版本支持） |

## 常用组合（复制即用）

```bash
# 总结当前改动
git diff | claude -p "总结这次改动，并指出潜在风险"

# 只读分析 + 产出计划
claude --permission-mode plan -p "分析当前项目的认证模块并给出重构计划"
```

## 相关页面

- 31 个高频技巧（Advent of Claude）
- 配置参考
- CLI 参考
- 安全指南
- Headless 模式
