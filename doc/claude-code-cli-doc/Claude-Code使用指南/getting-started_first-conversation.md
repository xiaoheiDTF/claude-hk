# Claude-Code / Getting-Started / First-Conversation

> 来源: claudecn.com

# 第一次对话

安装完成后，让我们开始第一次与 Claude Code 对话，熟悉基本的交互方式。

## 启动 Claude Code

### 在终端中启动

打开终端，进入你的项目目录，然后运行：

```bash
cd /path/to/your/project
claude
```

你会看到类似这样的界面：

```
╭─────────────────────────────────────────────────────────────╮
│ Claude Code                                                 │
│                                                             │
│ 输入 /help 查看帮助                                           │
╰─────────────────────────────────────────────────────────────╯

>
```

### 在 VS Code 中启动
如果安装了 VS Code 扩展：

- 按 Cmd+Shift+P（macOS）或 Ctrl+Shift+P（Windows/Linux）
- 输入 “Claude Code”
- 选择 “Claude Code: Open”
## 输入框使用

### 单行输入

直接在 `>` 提示符后输入内容，按 `Enter` 发送：

```
> 你好，介绍一下你自己
```

### 多行输入
如果需要输入多行内容，按 `Shift+Enter` 换行，然后继续输入：

```
> 请帮我写一个函数，要求：
  - 接收一个数组参数
  - 返回数组中的最大值
  - 处理空数组的情况
```

按 `Enter` 发送完整内容。

### 粘贴内容

可以直接粘贴代码或文本：

- macOS：Cmd+V
- Windows/Linux：Ctrl+V
粘贴的多行内容会自动保持格式。

## 简单示例

### 示例 1：自我介绍

```
> 你是什么？能做什么？
```

Claude Code 会介绍自己的能力和使用方式。

### 示例 2：写一段代码

```
> 写一个 Python 函数，计算斐波那契数列的第 n 项
```

Claude Code 会生成代码并解释其工作原理。

### 示例 3：解释代码

如果项目中有代码文件，可以让 Claude Code 解释：

```
> 解释一下 src/main.py 这个文件在做什么
```

Claude Code 会读取文件内容并给出解释。

## 理解回复格式

Claude Code 的回复可能包含几种不同的内容：

### 文字说明

普通的解释和回答，直接显示在终端中。

### 代码块

代码会以高亮格式显示：

```python
def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)
```

### 文件操作
当 Claude Code 需要创建或修改文件时，会先展示变更内容并请求确认：

```
我将创建文件 src/utils.py：

+def helper_function():
+    return "Hello"

是否确认创建？[y/n]
```

### 命令执行
当需要执行终端命令时，同样会请求确认：

```
我将执行以下命令：
> npm install lodash

是否确认执行？[y/n]
```

## 继续对话 vs 新对话

### 继续当前对话
Claude Code 会记住当前会话的上下文。你可以直接追问：

```
> 写一个排序函数
...（Claude Code 回复）

> 改成降序排序
...（Claude Code 理解上下文，修改之前的代码）
```

### 开始新对话
使用 `/clear` 命令清除当前对话历史，开始全新对话：

```
> /clear
对话已清除

> 新的问题...
```

### 继续上次会话
如果退出后想继续之前的对话：

```bash
claude -c
```

或者：

```bash
claude --continue
```

## 中断执行
如果 Claude Code 正在执行某个操作，想要中断：

- 按 Escape 键可以中断当前操作
- 按 Ctrl+C 可以强制停止
## 小技巧

### 清晰表达

**不太好的表达：**

```
> 改一下代码
```

**更好的表达：**

```
> 把 src/api.js 中的 fetch 调用改成使用 axios
```

### 提供上下文
**不太好的表达：**

```
> 修复这个 bug
```

**更好的表达：**

```
> 用户注册时如果邮箱已存在，页面会崩溃。请检查 src/auth/register.js 并修复
```

### 分步进行
对于复杂任务，可以分步进行：

```
> 首先，帮我分析一下当前项目的目录结构
...

> 接下来，在 src 目录下创建一个 utils 文件夹
...

> 然后，创建一个日期格式化的工具函数
```

## 获取帮助
随时可以使用 `/help` 命令查看帮助信息：

```
> /help
```

## 下一步
掌握了基本对话后，让我们学习更多 [基础使用](../basic-usage/) 技巧。
