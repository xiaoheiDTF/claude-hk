# Claude-Code / Advanced / Hooks

> 来源: claudecn.com

# Hooks 系统

Hooks 允许在 Claude Code 特定事件发生时自动执行[ 脚本](#)或 LLM 评估，实现自动化工作流。

如果你想直接参考一组“可迁移的 Hooks 配方”（tmux 提醒、git push 停顿、格式化/类型检查、console.log 审计等），见：[Hooks 配方：把经验变成自动化护栏](https://claudecn.com/docs/claude-code/advanced/hooks-recipes/)。

## Hook 事件类型

| 事件 | 触发时机 | 匹配器 |
| --- | --- | --- |
| `PreToolUse` | 工具执行前 | 工具名称 |
| `PostToolUse` | 工具执行后 | 工具名称 |
| `PermissionRequest` | 权限对话框显示时 | 工具名称 |
| `Notification` | 发送通知时 | 通知类型 |
| `UserPromptSubmit` | 用户提交提示时 | 无 |
| `Stop` | 主代理完成响应时 | 无 |
| `SubagentStop` | Subagent 完成时 | 代理名称 |
| `PreCompact` | 压缩操作前 | `manual` / `auto` |
| `Setup` | 运行仓库初始化/维护（`--init`/`--init-only`/`--maintenance`）时 | `init` / `maintenance` |
| `SessionStart` | 会话开始或恢复时 | `startup` / `resume` / `clear` / `compact` |
| `SessionEnd` | 会话结束时 | 退出原因 |

---

## 配置位置

Hooks 可在以下位置配置：

- ~/.claude/settings.json - 用户设置
- .claude/settings.json - 项目设置
- .claude/settings.local.json - 本地项目设置（不提交）
- 插件的 hooks/hooks.json
- Skills、Subagents 的 frontmatter 中
---

## 配置语法

### 基本结构

```json
{
  "hooks": {
    "EventName": [
      {
        "matcher": "ToolPattern",
        "hooks": [
          {
            "type": "command",
            "command": "your-command-here"
          }
        ]
      }
    ]
  }
}
```

### 字段说明

- matcher：匹配模式（区分大小写）精确匹配：Write 只匹配 Write 工具
- 支持正则：Edit|Write 或 Notebook.*
- 匹配所有：* 或留空
- hooks：匹配时执行的钩子数组type："command"（bash 命令）或 "prompt"（LLM 评估）
- command：要执行的 bash 命令
- prompt：发送给 LLM 的评估提示
- timeout：超时时间（秒）
### 无需匹配器的事件
`UserPromptSubmit`、`Stop`、`SubagentStop`、`Setup` 等事件可以省略 `matcher` 字段：

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/prompt-validator.py"
          }
        ]
      }
    ]
  }
}
```

### 使用项目目录变量
`$CLAUDE_PROJECT_DIR` 引用项目根目录：

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/check-style.sh"
          }
        ]
      }
    ]
  }
}
```

---

## 常用匹配器

### PreToolUse / PostToolUse

| 匹配器 | 说明 |
| --- | --- |
| `Task` | Subagent 任务 |
| `Bash` | Shell 命令 |
| `Glob` | 文件模式匹配 |
| `Grep` | 内容搜索 |
| `Read` | 文件读取 |
| `Edit` | 文件编辑 |
| `Write` | 文件写入 |
| `WebFetch`, `WebSearch` | Web 操作 |

### Notification

| 匹配器 | 说明 |
| --- | --- |
| `permission_prompt` | 权限请求 |
| `idle_prompt` | 空闲超过 60 秒 |
| `auth_success` | 认证成功 |
| `elicitation_dialog` | MCP 工具输入对话框 |

---

## 两种 Hook 类型

### 命令式 Hook（type: command）
执行 bash 命令，通过退出码和输出控制行为：

```json
{
  "type": "command",
  "command": "./scripts/validate-command.sh",
  "timeout": 30
}
```

### 提示式 Hook（type: prompt）
使用 LLM（Haiku）评估，适合需要上下文理解的决策：

```json
{
  "type": "prompt",
  "prompt": "评估 Claude 是否应该停止：$ARGUMENTS。检查所有任务是否完成。",
  "timeout": 30
}
```

LLM 必须返回 JSON：

```json
{
  "ok": true,
  "reason": "决策解释（ok 为 false 时必填）"
}
```

---

## 退出码行为

| 退出码 | 行为 |
| --- | --- |
| 0 | 成功。`stdout` 在详细模式显示 |
| 2 | 软/硬阻塞错误。对部分事件会阻止继续执行；对部分事件仅向用户显示 `stderr` |
| 其他 | 非阻塞错误。`stderr` 在详细模式显示，继续执行 |

### 退出码 2 的具体影响

| Hook 事件 | 行为 |
| --- | --- |
| `PreToolUse` | 阻止工具调用，向 Claude 显示 stderr |
| `PermissionRequest` | 拒绝权限，向 Claude 显示 stderr |
| `PostToolUse` | 向 Claude 显示 stderr（工具已执行） |
| `UserPromptSubmit` | 阻止提示处理，清除提示，仅向用户显示 stderr |
| `Stop` | 阻止停止，向 Claude 显示 stderr |
| `SubagentStop` | 阻止停止，向 Subagent 显示 stderr |
| `Setup` | 不阻止执行，仅向用户显示 stderr |
| `PreCompact` | 不阻止执行，仅向用户显示 stderr |
| `SessionStart` | 不阻止执行，仅向用户显示 stderr |
| `SessionEnd` | 不阻止执行，仅向用户显示 stderr |

---

## Hook 输入
Hooks 通过 stdin 接收 JSON 数据：

### 通用字段

```json
{
  "session_id": "abc123",
  "transcript_path": "/Users/.../.claude/projects/.../session.jsonl",
  "cwd": "/Users/...",
  "permission_mode": "default",
  "hook_event_name": "PreToolUse"
}
```

### Setup 输入示例
当使用 `--init` / `--init-only` / `--maintenance` 启动时，会触发 `Setup`，并包含 `trigger` 字段：

```json
{
  "session_id": "abc123",
  "transcript_path": "~/.claude/projects/.../00893aaf-19fa-41d2-8238-13269b9b3ca0.jsonl",
  "cwd": "/Users/...",
  "permission_mode": "default",
  "hook_event_name": "Setup",
  "trigger": "init"
}
```

`trigger` 取值：

- init：来自 --init 或 --init-only
- maintenance：来自 --maintenance
### PreToolUse 输入示例（Bash 工具）

```json
{
  "hook_event_name": "PreToolUse",
  "tool_name": "Bash",
  "tool_input": {
    "command": "psql -c 'SELECT * FROM users'",
    "description": "查询用户表",
    "timeout": 120000
  },
  "tool_use_id": "toolu_01ABC123..."
}
```

### PostToolUse 输入示例

```json
{
  "hook_event_name": "PostToolUse",
  "tool_name": "Write",
  "tool_input": {
    "file_path": "/path/to/file.txt",
    "content": "文件内容"
  },
  "tool_response": {
    "filePath": "/path/to/file.txt",
    "success": true
  }
}
```

---

## 实用示例

### 文件编辑后自动格式化

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "prettier --write \"$TOOL_INPUT_file_path\" 2>/dev/null || true"
          }
        ]
      }
    ]
  }
}
```

### 阻止危险命令
创建验证[ 脚本](#) `.claude/hooks/validate-command.sh`：

```bash
#!/bin/bash

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

# 检查危险命令
if echo "$COMMAND" | grep -qE "(rm -rf /|sudo rm|chmod 777|> /dev/sda)"; then
  echo "已阻止危险命令：$COMMAND" >&2
  exit 2
fi

exit 0
```

配置：

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/validate-command.sh"
          }
        ]
      }
    ]
  }
}
```

### 会话启动时设置环境
`SessionStart` 可使用 `CLAUDE_ENV_FILE` 持久化环境变量：

```bash
#!/bin/bash
# .claude/hooks/setup-env.sh

if [ -n "$CLAUDE_ENV_FILE" ]; then
  echo 'export NODE_ENV=development' >> "$CLAUDE_ENV_FILE"
  echo 'export API_KEY=your-api-key' >> "$CLAUDE_ENV_FILE"
  echo 'export PATH="$PATH:./node_modules/.bin"' >> "$CLAUDE_ENV_FILE"
fi

exit 0
```

配置：

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/setup-env.sh"
          }
        ]
      }
    ]
  }
}
```

`SessionStart` 会在每次会话开始/恢复时触发，建议保持足够轻量；如果是“偶发/一次性”的操作（依赖安装、迁移、周期性清理等），更适合放到 `Setup`（需要用 `--init` / `--init-only` / `--maintenance` 显式触发）。

### 不同通知类型的处理

```json
{
  "hooks": {
    "Notification": [
      {
        "matcher": "permission_prompt",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/permission-alert.sh"
          }
        ]
      },
      {
        "matcher": "idle_prompt",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/idle-notification.sh"
          }
        ]
      }
    ]
  }
}
```

### 智能停止判断
使用 LLM 评估是否应该停止：

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "prompt",
            "prompt": "你正在评估 Claude 是否应该停止工作。上下文：$ARGUMENTS\n\n分析对话并判断：\n1. 所有用户请求的任务是否完成\n2. 是否有错误需要处理\n3. 是否需要后续工作\n\n返回 JSON：{\"ok\": true} 允许停止，或 {\"ok\": false, \"reason\": \"你的解释\"} 继续工作。",
            "timeout": 30
          }
        ]
      }
    ]
  }
}
```

---

## 高级 JSON 输出
Hooks 可返回结构化 JSON 进行更精细控制（仅退出码 0 时处理）：

### 通用字段

```json
{
  "continue": true,           // 是否继续处理
  "stopReason": "string",     // continue 为 false 时显示的原因
  "suppressOutput": true,     // 隐藏 stdout 输出
  "systemMessage": "string"   // 向用户显示的警告信息
}
```

### 提供额外上下文（additionalContext）
Hooks 可以通过 `hookSpecificOutput.additionalContext` 在工具执行前给 Claude 增加一段上下文（多个 hooks 的 `additionalContext` 会被拼接）：

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "additionalContext": "Current environment: production. Proceed with caution."
  }
}
```

### PreToolUse 决策控制

```json
{
  "permissionDecision": "allow",  // "allow" | "deny" | "ask"
  "permissionDecisionReason": "原因说明"
}
```

- "allow"：绕过权限系统，直接执行
- "deny"：阻止工具调用
- "ask"：显示权限对话框
### PermissionRequest 决策控制

```json
{
  "decision": "allow"  // "allow" | "deny" | "pass"
}
```

---

## 插件 Hooks
插件可在 `hooks/hooks.json` 中定义 Hooks：

```json
{
  "description": "自动代码格式化",
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/scripts/format.sh",
            "timeout": 30
          }
        ]
      }
    ]
  }
}
```

插件 Hooks 与用户和项目 Hooks 合并执行。

---

## 在 Skills 和 Subagents 中使用

Skills 和 Subagents 可在 frontmatter 中定义作用域内的 Hooks：

```yaml
---
name: secure-operations
description: 执行带安全检查的操作
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "./scripts/security-check.sh"
          once: true
---
```

`once: true` 表示只运行一次。这些 Hooks 仅在该组件激活时运行，完成后自动清理。
（提示：目前 `once` 仅 Skills 支持，不适用于 agents/subagents。）

---

## 调试技巧

### 添加日志

```bash
#!/bin/bash
echo "Hook triggered: $(date)" >> /tmp/claude-hooks.log
echo "Event: $HOOK_EVENT_NAME" >> /tmp/claude-hooks.log
cat >> /tmp/claude-hooks.log  # 将 stdin 追加到日志
```

### 查看日志

```bash
tail -f /tmp/claude-hooks.log
```

### 使用调试模式

```bash
claude --debug
```

---

## 安全最佳实践

- 验证输入 - 始终验证和清理 stdin 中的 JSON 输入
- 避免硬编码密钥 - 使用环境变量存储敏感信息
- 限制文件访问 - 脚本应只访问必要的文件和目录
- 定期审查 - 定期检查 Hook 配置和脚本
- 使用 exit 2 - 阻止危险操作时使用退出码 2
---

## 下一步
[Agent Skills创建可复用的知识模块
](../skills/)[Subagents创建专用子代理处理任务
](../subagents/)[MCP 服务器连接外部工具和数据源
](../mcp-servers/)
