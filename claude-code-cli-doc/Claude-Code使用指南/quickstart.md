# Claude-Code / Quickstart

> 来源: claudecn.com

# 快速开始

欢迎使用 Claude Code！

本快速入门指南将帮助你在几分钟内开始使用 AI 驱动的编码助手。完成后，你将了解如何使用 Claude Code 完成常见开发任务。

## 开始之前

确保你已准备好：

- 打开的终端或命令提示符
- 一个代码项目
- Claude.ai（推荐）或 Claude Console 账号
## 步骤 1：安装 Claude Code

### NPM 安装

如果你已安装 Node.js 18 或更高版本：

```bash
npm install -g @anthropic-ai/claude-code
```

### 原生安装

或者尝试我们的原生安装方式，现已进入 beta 阶段。

**Homebrew (macOS, Linux):**

```bash
brew install --cask claude-code
```

**macOS, Linux, WSL:**

```bash
curl -fsSL https://claude.ai/install.sh | bash
```

**Windows PowerShell:**

```powershell
irm https://claude.ai/install.ps1 | iex
```

**Windows CMD:**

```batch
curl -fsSL https://claude.ai/install.cmd -o install.cmd && install.cmd && del install.cmd
```

**WinGet (Windows):**

```powershell
winget install Anthropic.ClaudeCode
```

## 步骤 2：登录账号
Claude Code 需要账号才能使用。当你使用 `claude` 命令启动交互会话时，需要登录：

```bash
claude
# 首次使用时会提示登录
```

```bash
/login
# 按照提示登录你的账号
```

你可以使用以下任一账号类型登录：

- Claude.ai（订阅计划 - 推荐）
- Claude Console（API 访问，需预付费）
登录后，凭据会被存储，无需再次登录。

当你首次使用 Claude Console 账号认证 Claude Code 时，会自动创建一个名为"Claude Code"的工作区。此工作区提供集中的成本跟踪和组织内所有 Claude Code 使用的管理。

你可以在同一个邮箱地址下拥有两种账号类型。如果需要再次登录或切换账号，在 Claude Code 中使用 `/login` 命令。

## 步骤 3：启动首次会话
在任何项目目录中打开终端并启动 Claude Code：

```bash
cd /path/to/your/project
claude
```

你会看到 Claude Code 欢迎屏幕，显示会话信息、最近对话和最新更新。输入 `/help` 查看可用命令，或 `/resume` 继续上次对话。

## 步骤 4：提出第一个问题

让我们从了解代码库开始。尝试这些命令：

```
> 这个项目是做什么的？
```

Claude 会分析文件并提供摘要。你也可以提出更具体的问题：

```
> 这个项目使用了哪些技术？
```

```
> 主入口点在哪里？
```

```
> 解释一下文件夹结构
```

你还可以询问 Claude 自身的功能：

```
> Claude Code 能做什么？
```

```
> 如何在 Claude Code 中使用斜杠命令？
```

```
> Claude Code 可以与 Docker 配合使用吗？
```

Claude Code 会根据需要读取文件 - 你无需手动添加上下文。Claude 还可以访问自己的文档，并能回答有关其功能和特性的问题。

## 步骤 5：进行首次代码修改
现在让 Claude Code 进行实际编码。尝试一个简单的任务：

```
> 在主文件中添加一个 hello world 函数
```

Claude Code 会：

- 找到合适的文件
- 展示建议的修改
- 请求你的批准
- 执行编辑
Claude Code 在修改文件前总是请求许可。你可以批准单个更改，或为会话启用"全部接受"模式。

## 步骤 6：使用 Git
Claude Code 让 Git 操作变得对话化：

```
> 我修改了哪些文件？
```

```
> 用描述性信息提交我的改动
```

你还可以提示更复杂的 Git 操作：

```
> 创建一个名为 feature/quickstart 的新分支
```

```
> 显示最近 5 次提交
```

```
> 帮我解决合并冲突
```

## 步骤 7：修复 bug 或添加功能
Claude 擅长调试和功能实现。

用自然语言描述你想要的：

```
> 为用户注册表单添加输入验证
```

或修复现有问题：

```
> 有个 bug，用户可以提交空表单 - 修复它
```

Claude Code 会：

- 定位相关代码
- 理解上下文
- 实现解决方案
- 运行测试（如果可用）
## 步骤 8：尝试其他常用工作流

与 Claude 合作有多种方式：

**重构代码**

```
> 将认证模块重构为使用 async/await 而不是回调
```

**编写测试**

```
> 为计算器函数编写单元测试
```

**更新文档**

```
> 更新 README，添加安装说明
```

**代码审查**

```
> 审查我的改动并提出改进建议
```

**记住**：Claude Code 是你的 AI [ 编程](#)伙伴。像与乐于助人的同事交谈一样与它对话 - 描述你想要实现的目标，它会帮助你达成。

## 基本命令
以下是日常使用最重要的命令：

| 命令 | 作用 | 示例 |
| --- | --- | --- |
| `claude` | 启动交互模式 | `claude` |
| `claude "任务"` | 运行一次性任务 | `claude "修复构建错误"` |
| `claude -p "查询"` | 运行一次性查询后退出 | `claude -p "解释这个函数"` |
| `claude -c` | 继续最近的对话 | `claude -c` |
| `claude -r` | 恢复之前的对话 | `claude -r` |
| `claude commit` | 创建 Git 提交 | `claude commit` |
| `/clear` | 清除对话历史 | `> /clear` |
| `/help` | 显示可用命令 | `> /help` |
| `exit` 或 Ctrl+C | 退出 Claude Code | `> exit` |

查看 [CLI 参考](https://docs.claude.com/en/docs/claude-code/cli-reference) 获取完整命令列表。

## 新手专业技巧
****
不要说：“修复 bug”

试试：“修复登录 bug，用户输入错误凭据后看到空白屏幕”

****
将复杂任务分解为步骤：

```
> 1. 为用户资料创建新数据库表
```

```
> 2. 创建 API 端点来获取和更新用户资料
```

```
> 3. 构建允许用户查看和编辑信息的网页
```

****在进行更改前，让 Claude 理解你的代码：

```
> 分析数据库模式
```

```
> 构建显示英国客户最常退货产品的仪表板
```

****
- 按 ? 查看所有可用键盘快捷键
- 使用 Tab 进行命令补全
- 按 ↑ 查看命令历史
- 输入 / 查看所有斜杠命令

## 下一步
现在你已经学会了基础知识，探索更多高级功能：
[常用工作流常见任务的分步指南
](https://claudecn.com/docs/claude-code/workflows/)[CLI 参考掌握所有命令和选项
](https://docs.claude.com/en/docs/claude-code/cli-reference)[配置自定义 Claude Code 工作流
](https://claudecn.com/docs/claude-code/reference/settings/)

## 获取帮助

- 在 Claude Code 中：输入 /help 或询问"如何…"
- 文档：你现在就在这里！浏览其他指南
- 社区：加入我们的 Discord 获取技巧和支持
