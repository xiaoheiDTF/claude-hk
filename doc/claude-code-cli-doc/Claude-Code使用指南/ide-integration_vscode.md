# Claude-Code / Ide-Integration / Vscode

> 来源: claudecn.com

# VS Code 集成

VS Code 扩展为 Claude Code 提供原生图形界面，直接集成到 IDE 中。支持内联差异预览、@提及文件、计划审查等功能。

## 前提条件

- VS Code 1.98.0 或更高版本
- Anthropic 账户（首次打开扩展时登录）
扩展**自带 CLI（命令行界面）**：在 VS Code 的集成终端里直接运行 `claude`，即可使用 CLI 的高级能力（例如 MCP 配置、更多命令等）。详见本文的「扩展 vs CLI 功能对比」。

---

## 安装扩展

### 方式一：直接安装

- VS Code 安装链接
- Cursor 安装链接
### 方式二：扩展商店

- 按 Cmd+Shift+X（Mac）或 Ctrl+Shift+X（Windows/Linux）打开扩展视图
- 搜索 “Claude Code”
- 点击 Install安装后可能需要重启 VS Code 或运行命令面板中的 “Developer: Reload Window”。

---

## 开始使用

### 打开 Claude Code 面板

**方式一：编辑器工具栏**

打开任意文件后，点击编辑器右上角的 ✱ 图标。

**方式二：命令面板**

按 `Cmd+Shift+P`（Mac）或 `Ctrl+Shift+P`（Windows/Linux），输入 “Claude Code”，选择 “Open in New Tab”。

**方式三：状态栏**

点击窗口右下角的 “✱ Claude Code”。

### 发送提示

向 Claude 提问代码相关问题，无论是解释代码、调试问题还是进行修改。

**快速引用**：Claude 会自动看到你选中的文本。你也可以按 `Option+K`（Mac）/ `Alt+K`（Windows/Linux）插入 `@file.ts#5-10` 这类引用（带文件路径与行号）到提示中，便于精确定位。

### 审查修改
当 Claude 想要编辑文件时，会展示原始内容与拟修改内容的对比（side-by-side），并请求权限。你可以接受、拒绝或告诉 Claude 改用其他方式。

---

## VS Code 命令和快捷键

| 命令 | 快捷键 | 说明 |
| --- | --- | --- |
| Focus Input | `Cmd+Esc` / `Ctrl+Esc` | 在编辑器和 Claude 之间切换焦点 |
| Open in New Tab | `Cmd+Shift+Esc` / `Ctrl+Shift+Esc` | 在新标签页打开对话 |
| New Conversation | `Cmd+N` / `Ctrl+N` | 开始新对话（Claude 聚焦时） |
| Insert @-Mention | `Alt+K` | 插入当前文件引用（选中文本会包含行号） |
| Open in Side Bar | — | 在左侧边栏打开 |
| Open in Terminal | — | 以终端模式打开 |
| Open in New Window | — | 在独立窗口打开 |
| Show Logs | — | 查看扩展调试日志 |
| Logout | — | 退出登录 |

使用 **Open in New Tab** 或 **Open in New Window** 可以同时运行多个对话。

扩展内置了两套入门引导：

- Walkthrough：命令面板运行 “Claude Code: Open Walkthrough” 了解基础流程
- 交互式清单：Claude 面板标题栏的“学位帽”图标，按步骤体验 Plan Mode、规则配置等能力

---

## 配置设置

### 扩展设置

按 `Cmd+,`（Mac）或 `Ctrl+,`（Windows/Linux）打开设置，导航到 Extensions → Claude Code。也可以在提示框输入 `/`，选择 **General Config** 快速打开相关设置。

下面是更贴近内部配置名的一份速查（以扩展版本为准）：

| 设置 | 默认值 | 说明 |
| --- | --- | --- |
| `selectedModel` | `default` | 新对话默认模型（会话内可用 `/model` 切换） |
| `useTerminal` | `false` | 用终端模式替代图形面板 |
| `initialPermissionMode` | `default` | 审批模式：`default` / `plan` / `acceptEdits` / `bypassPermissions` |
| `preferredLocation` | `panel` | 打开位置：`sidebar`（右侧）或 `panel`（新标签页） |
| `autosave` | `true` | Claude 读写文件前自动保存 |
| `useCtrlEnterToSend` | `false` | 使用 Ctrl/Cmd+Enter 发送（避免回车误触） |
| `enableNewConversationShortcut` | `true` | 启用 Cmd/Ctrl+N 新对话快捷键 |
| `hideOnboarding` | `false` | 隐藏“学位帽”入门清单 |
| `respectGitIgnore` | `true` | 文件搜索遵循 `.gitignore` |
| `environmentVariables` | `[]` | 仅给 Claude 进程注入环境变量；更建议写到 `~/.claude/settings.json` 以便扩展与 CLI 共享 |
| `disableLoginPrompt` | `false` | 跳过登录提示（常用于第三方提供商场景） |
| `allowDangerouslySkipPermissions` | `false` | 绕过所有审批提示（高风险，谨慎开启） |
| `claudeProcessWrapper` | — | 启动 Claude 进程的可执行包装器路径 |

### Claude Code 设置

`~/.claude/settings.json` 中的设置在扩展和 CLI 之间共享，包括：

- 允许的命令和目录
- 环境变量
- Hooks
- MCP 服务器
详见[配置参考](https://claudecn.com/docs/claude-code/reference/settings/)。

---

## 使用第三方提供商

如果使用 Amazon Bedrock、Google Vertex AI 或 Microsoft Foundry：

如果你主要关心的是“第三方 Provider（Anthropic 兼容接口）怎么配、怎么切换、以及扩展与 CLI 如何共用同一套配置”，可以直接看这篇实战：
[/blog/claude-code-install-and-provider-switching/](https://claudecn.com/blog/claude-code-install-and-provider-switching/)

### 步骤一：禁用登录提示
打开设置，搜索 “Claude Code login”，勾选 **Disable Login Prompt**。

### 步骤二：配置提供商

在 `~/.claude/settings.json` 中配置你的提供商。参考：

- Amazon Bedrock
- Google Vertex AI（待补充）
- Microsoft Foundry（待补充）
这些设置在扩展和 CLI 之间共享。

---

## 扩展 vs CLI 功能对比

Claude Code 同时提供 VS Code 扩展（图形面板）与 CLI（终端）。部分能力只在 CLI 提供；需要时直接在 VS Code 集成终端运行 `claude`。

| 功能 | CLI | VS Code 扩展 |
| --- | --- | --- |
| 命令与 Skills | 完整支持 | 子集（输入 `/` 查看可用命令） |
| MCP 服务器配置 | 支持 | 不支持（需用 CLI 配置，但可在扩展中使用） |
| 检查点 | 支持 | 即将支持 |
| `!` Bash 快捷方式 | 支持 | 不支持 |
| Tab 补全 | 支持 | 不支持 |

### 在 VS Code 中使用 CLI

在集成终端（`Ctrl+`` 或 `Cmd+``）中运行 `claude` 即可使用 CLI，自动集成差异查看和诊断共享。

如果使用外部终端，在 Claude Code 中运行 `/ide` 连接到 VS Code。

### 配置 MCP（通过 CLI，一次配置两端共享）

MCP（Model Context Protocol）服务器可以把 Claude 接到外部工具、数据库与 API。建议在集成终端运行 `claude mcp ...` 完成配置，然后扩展与 CLI 都可以使用。

```bash
claude mcp
```

### 切换扩展和 CLI
扩展和 CLI 共享对话历史。要在 CLI 中继续扩展对话：

```bash
claude --resume
```

这会打开交互式选择器，搜索并选择你的对话。

---

## 自定义布局

可以拖动 Claude 面板到任意位置：

- 次级侧边栏（默认）：窗口右侧
- 主侧边栏：左侧带图标的侧边栏
- 编辑器区域：作为标签页与文件并列
### 切换到终端模式

如果偏好 CLI 风格界面，在设置中勾选 **Use Terminal**。

---

## 常见问题

### 扩展无法安装

- 确认 VS Code 版本 ≥ 1.98.0
- 检查 VS Code 是否有安装扩展的权限
- 尝试从 Marketplace 网站直接安装
### 看不到 ✱ 图标

编辑器工具栏图标需要打开文件才会显示：

- 打开一个文件：仅打开文件夹不够
- 检查版本：Help → About 确认 ≥ 1.98.0
- 重启 VS Code：运行 “Developer: Reload Window”
- 禁用冲突扩展：临时禁用其他 AI 扩展
或者使用状态栏的 “✱ Claude Code” 或命令面板。

### Claude 没有响应

- 检查网络连接
- 尝试开始新对话
- 在终端运行 claude 查看详细错误
- 问题持续则提交 Issue
### CLI 无法连接到 IDE

- 确保在 VS Code 集成终端中运行（非外部终端）
- 确保 IDE 变体的 CLI 已安装：VS Code: code 命令
- Cursor: cursor 命令
- Windsurf: windsurf 命令
- VSCodium: codium 命令
- 如命令不可用，从命令面板安装：“Shell Command: Install ‘code’ command in PATH”
---

## 安全注意事项

启用自动编辑权限时，Claude Code 可能修改 VS Code 配置文件（如 `settings.json` 或 `tasks.json`），这些文件可能被 VS Code 自动执行。

处理不受信任的代码时：

- 启用 VS Code 受限模式（Workspace Trust / 工作区信任）
- 使用手动审批模式而非自动接受
- 仔细审查修改后再接受
---

## 卸载扩展

- 打开扩展视图（Cmd+Shift+X / Ctrl+Shift+X）
- 搜索 “Claude Code”
- 点击 Uninstall
同时删除扩展数据和重置设置：

```bash
rm -rf ~/.vscode/globalStorage/anthropic.claude-code
```

---

## 下一步
[工作流了解常用开发工作流
](../../workflows/)[MCP 服务器连接外部工具和数据源
](../../advanced/mcp-servers/)[Hooks 系统实现自动化工作流
](../../advanced/hooks/)
