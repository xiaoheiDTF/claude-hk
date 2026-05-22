# Claude-Code / Getting-Started / Basic-Usage

> 来源: claudecn.com

# 基础使用

本文介绍 Claude Code 日常开发中的常用操作和技巧。

## 常用启动方式

### 交互模式

最常用的方式，启动后进入对话界面：

```bash
claude
```

### 一次性任务
执行单个任务后退出，适合简单操作：

```bash
claude "统计这个项目有多少行代码"
```

### 继续上次对话
恢复之前的对话上下文：

```bash
claude -c
# 或
claude --continue
```

### 指定工作目录
在特定目录下启动：

```bash
claude --cwd /path/to/project
```

## 基本操作

### 读取文件
让 Claude Code 阅读和理解文件：

```
> 读取 src/config.js 并解释配置项的作用
```

```
> package.json 中有哪些依赖？
```

### 写入文件
创建或修改文件：

```
> 创建一个 .gitignore 文件，包含 Node.js 项目常见的忽略项
```

```
> 在 src/utils.js 中添加一个格式化日期的函数
```

Claude Code 会展示将要进行的修改，等待你确认后再执行。

### 执行命令

运行终端命令：

```
> 运行测试
```

```
> 安装 lodash 依赖
```

```
> 构建项目
```

同样会先展示命令内容，等待确认。

## 项目初始化

### 使用 /init 命令

在新项目中，使用 `/init` 让 Claude Code 了解项目并创建 CLAUDE.md 配置文件：

```
> /init
```

Claude Code 会：

- 分析项目结构
- 识别使用的技术栈
- 创建 CLAUDE.md 文件记录项目约定
CLAUDE.md 文件帮助 Claude Code 记住项目的特定规则，例如：

- 代码风格约定
- 常用命令
- 项目结构说明
- 特殊注意事项
### 手动编辑 CLAUDE.md

你也可以手动创建或编辑 CLAUDE.md：

```markdown
# 项目说明

这是一个 React + TypeScript 项目。

## 代码规范

- 使用函数组件和 Hooks
- 组件文件使用 PascalCase 命名
- 工具函数使用 camelCase 命名

## 常用命令

- `npm run dev` - 启动开发服务器
- `npm run build` - 构建生产版本
- `npm run test` - 运行测试
```

## 使用截图沟通
Claude Code 支持图片输入，你可以通过截图来沟通问题。

### 粘贴截图

- 截取屏幕内容
- 在 Claude Code 输入框中按 Ctrl+V（Windows/Linux）或 Cmd+V（macOS）
- 添加文字说明
```
> [粘贴截图]
  这个按钮的样式有问题，请帮我修复
```

这在处理 UI 问题、错误信息截图时特别有用。

## 中断操作

### 使用 Escape

如果 Claude Code 正在思考或执行任务，按 `Escape` 可以：

- 中断当前思考过程
- 取消正在进行的操作
- 返回输入状态
### 拒绝操作

当 Claude Code 请求确认时，输入 `n` 拒绝：

```
是否确认执行？[y/n] n

已取消操作
```

## 常用斜杠命令
Claude Code 提供一些内置命令，以 `/` 开头：

| 命令 | 说明 |
| --- | --- |
| `/help` | 显示帮助信息 |
| `/clear` | 清除当前对话历史 |
| `/init` | 初始化项目配置 |
| `/config` | 查看或修改配置 |
| `/cost` | 查看当前会话的 Token 使用量 |
| `/doctor` | 检查系统环境和配置 |

### /help

获取帮助信息：

```
> /help
```

### /clear
清除对话历史，开始新对话：

```
> /clear
```

### /commit
智能生成提交信息并提交代码：

```
> /commit
```

Claude Code 会：

- 分析当前的代码变更
- 生成描述性的提交信息
- 请求确认后执行提交
## Git 操作

Claude Code 对 Git 操作有很好的支持：

### 查看状态

```
> 我修改了哪些文件？
```

### 查看差异

```
> 显示 src/main.js 的修改内容
```

### 提交代码

```
> 提交当前修改，信息简洁一些
```

### 创建分支

```
> 创建一个 feature/user-auth 分支
```

## 实用技巧

### 批量操作

```
> 把所有 .js 文件重命名为 .ts
```

```
> 在所有组件文件中添加 PropTypes
```

### 代码审查

```
> 检查 src/api 目录下的代码，找出潜在问题
```

### 生成文档

```
> 为 src/utils 目录下的所有函数生成 JSDoc 注释
```

### 重构代码

```
> 将回调函数改为 async/await 风格
```

## 安全提示
Claude Code 在执行敏感操作前会请求确认：

- 创建或修改文件
- 执行终端命令
- 删除文件
请仔细阅读确认提示，确保操作符合预期。

## 下一步

掌握了基础使用后，你可以探索更多进阶功能：

- 查看 快速开始 了解更多示例
- 探索 插件系统 扩展功能
