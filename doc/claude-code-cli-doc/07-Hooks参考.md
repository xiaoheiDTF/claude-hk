# Hooks 参考

> 来源: https://code.claude.com/docs/zh-CN/hooks

Hooks 是用户定义的 shell 命令、HTTP 端点或 LLM 提示，在 Claude Code 生命周期中的特定点自动执行。

---

## Hook 生命周期事件

| 事件 | 触发时机 | 频率 |
|------|---------|------|
| `SessionStart` | 会话开始或恢复时 | 每个会话一次 |
| `Setup` | 使用 `--init-only` 启动，或在 `-p` 模式中使用 `--init` 或 `--maintenance` 时 | 按标志触发 |
| `UserPromptSubmit` | 用户提交提示，Claude 处理之前 | 每轮一次 |
| `UserPromptExpansion` | 用户输入的命令展开为提示之前 | 每轮一次 |
| `PreToolUse` | 工具调用执行之前 | 每个工具调用 |
| `PermissionRequest` | 权限对话框出现时 | 每个工具调用 |
| `PermissionDenied` | 自动模式分类器拒绝工具调用时 | 每个拒绝 |
| `PostToolUse` | 工具调用成功后 | 每个工具调用 |
| `PostToolUseFailure` | 工具调用失败后 | 每个失败 |
| `PostToolBatch` | 完整批次并行工具调用解决后 | 每批次一次 |
| `Notification` | Claude Code 发送通知时 | 每次通知 |
| `SubagentStart` | Subagent 生成时 | 每次生成 |
| `SubagentStop` | Subagent 完成时 | 每次完成 |
| `TaskCreated` | 通过 TaskCreate 创建任务时 | 每次创建 |
| `TaskCompleted` | 任务被标记为已完成时 | 每次完成 |
| `Stop` | Claude 完成响应时 | 每轮一次 |
| `StopFailure` | 回合因 API 错误结束时 | 每次失败 |
| `TeammateIdle` | Agent team 队友即将空闲时 | 每次 |
| `InstructionsLoaded` | CLAUDE.md 或 `.claude/rules/*.md` 加载到上下文时 | 按需 |
| `ConfigChange` | 配置文件在会话期间更改时 | 每次更改 |
| `CwdChanged` | 工作目录更改时（如 Claude 执行 cd 命令） | 每次更改 |
| `FileChanged` | 监视的文件在磁盘上更改时 | 每次更改 |
| `WorktreeCreate` | 通过 `--worktree` 或 `isolation: "worktree"` 创建 worktree 时 | 每次创建 |
| `WorktreeRemove` | Worktree 被移除时（会话退出或 subagent 完成） | 每次移除 |
| `PreCompact` | 上下文压缩之前 | 每次压缩 |
| `PostCompact` | 上下文压缩完成后 | 每次压缩 |
| `Elicitation` | MCP 服务器在工具调用期间请求用户输入时 | 每次请求 |
| `ElicitationResult` | 用户响应 MCP elicitation 后，响应发送回服务器之前 | 每次响应 |
| `SessionEnd` | 会话终止时 | 每个会话一次 |

---

## Hook 配置位置

| 位置 | 范围 | 可共享 |
|------|------|--------|
| `~/.claude/settings.json` | 您的所有项目 | 否，本地于您的计算机 |
| `.claude/settings.json` | 单个项目 | 是，可以提交到仓库 |
| `.claude/settings.local.json` | 单个项目 | 否，gitignored |
| 托管策略设置 | 组织范围 | 是，管理员控制 |
| Plugin hooks/hooks.json | 启用插件时 | 是，与插件捆绑 |
| Skill 或代理 frontmatter | 组件活跃时 | 是，在组件文件中定义 |

---

## 匹配器模式技巧

匹配器过滤 hooks 何时触发。评估方式取决于包含的字符：

| 匹配器值 | 评估为 | 示例 |
|---------|--------|------|
| `*`、`""` 或省略 | 匹配所有 | 在事件的每次出现时触发 |
| 仅字母、数字、`_` 和 `\|` | 精确字符串或 `\|` 分隔的精确字符串列表 | `Bash` 仅匹配 Bash 工具；`Edit\|Write` 精确匹配任一工具 |
| 包含任何其他字符 | JavaScript 正则表达式 | `^Notebook` 匹配任何以 Notebook 开头的工具；`mcp__memory__.*` 匹配来自 memory 服务器的每个工具 |

### MCP 工具匹配技巧

MCP 工具遵循命名模式 `mcp__<server>__<tool>`：

- `mcp__memory__.*` 匹配来自 memory 服务器的所有工具
- `mcp__.*__write.*` 匹配来自任何服务器的任何名称以 write 开头的工具

> 注意：`.*` 是必需的。像 `mcp__memory` 这样的匹配器仅包含字母和下划线，因此它作为精确字符串进行比较，不匹配任何工具。

---

## Hook 处理程序类型

有五种 hook 类型：

### 1. 命令 Hooks (`type: "command"`)

运行 shell 命令。脚本在 stdin 上接收事件的 JSON 输入，并通过退出代码和 stdout 传回结果。

**特殊字段**：
- `command`（必需）：要执行的 shell 命令
- `async`：如果为 `true`，在后台运行而不阻止
- `asyncRewake`：如果为 `true`，在后台运行并在退出代码 2 时唤醒 Claude
- `shell`：用于此 hook 的 shell（`bash` 或 `powershell`）

### 2. HTTP Hooks (`type: "http"`)

将事件的 JSON 输入作为 HTTP POST 请求发送到 URL。

**特殊字段**：
- `url`（必需）：发送 POST 请求的 URL
- `headers`：其他 HTTP 标头作为键值对
- `allowedEnvVars`：可能被插值到标头值中的环境变量名称列表

> HTTP hooks 使用 HTTP 状态代码和响应体。非 2xx 响应、连接失败和超时都会产生非阻止错误。要阻止工具调用，返回 2xx 响应，其 JSON 体包含适当的决定字段。

### 3. MCP 工具 Hooks (`type: "mcp_tool"`)

在已连接的 MCP 服务器上调用工具。

**特殊字段**：
- `server`（必需）：已配置的 MCP 服务器的名称
- `tool`（必需）：该服务器上要调用的工具的名称
- `input`：传递给工具的参数。支持 `${path}` 替换

### 4. 提示 Hooks (`type: "prompt"`)

向 Claude 模型发送提示以进行单轮评估。模型返回 yes/no 决定作为 JSON。

**特殊字段**：
- `prompt`（必需）：要发送给模型的提示文本。使用 `$ARGUMENTS` 作为 hook 输入 JSON 的占位符
- `model`：用于评估的模型（默认为快速模型）

### 5. 代理 Hooks (`type: "agent"`)

生成一个可以使用 Read、Grep 和 Glob 等工具来验证条件的 subagent，然后返回决定。

---

## 退出代码技巧

### 命令 Hooks 的退出代码

| 退出代码 | 含义 |
|---------|------|
| `0` | 成功。Claude Code 解析 stdout 以获取 JSON 输出字段 |
| `2` | 阻止错误。Claude Code 忽略 stdout。stderr 文本被反馈给 Claude 作为错误消息 |
| 其他 | 大多数 hook 事件的非阻止错误。执行继续 |

### 退出代码 2 的行为（按事件）

| Hook 事件 | 退出 2 的效果 |
|-----------|-------------|
| `PreToolUse` | 阻止工具调用 |
| `PermissionRequest` | 拒绝权限 |
| `UserPromptSubmit` | 阻止提示处理并从上下文中删除提示 |
| `UserPromptExpansion` | 阻止扩展 |
| `Stop` | 防止 Claude 停止，继续对话 |
| `SubagentStop` | 防止 subagent 停止 |
| `TeammateIdle` | 防止队友空闲（队友继续工作） |
| `TaskCreated` | 回滚任务创建 |
| `TaskCompleted` | 防止任务被标记为已完成 |
| `ConfigChange` | 阻止配置更改生效（除了 policy_settings） |
| `PreCompact` | 阻止压缩 |
| `WorktreeCreate` | 任何非零退出代码都会导致 worktree 创建失败 |
| `PostToolBatch` | 在下一个模型调用之前停止代理循环 |
| `Elicitation` | 拒绝 elicitation |
| `ElicitationResult` | 阻止响应（操作变为 decline） |

> **重要**：对于旨在强制执行策略的 hook，请使用 `exit 2`。`exit 1` 被视为非阻止错误。

---

## JSON 输出技巧

### 通用字段（所有事件）

| 字段 | 默认 | 描述 |
|------|------|------|
| `continue` | `true` | 如果为 `false`，Claude 在 hook 运行后完全停止处理 |
| `stopReason` | 无 | hook 运行后 `continue` 为 `false` 时向用户显示的消息 |
| `suppressOutput` | `false` | 如果为 `true`，从调试日志中隐藏 stdout |
| `systemMessage` | 无 | 向用户显示的警告消息 |

### 为 Claude 添加上下文

`additionalContext` 字段将来自您的 hook 的字符串传递到 Claude 的上下文窗口中：

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "This file is generated. Edit src/schema.ts and run `bun generate` instead."
  }
}
```

> 提醒出现的位置取决于事件。当多个 hooks 为同一事件返回 `additionalContext` 时，Claude 接收所有值。如果值超过 10,000 个字符，完整文本写入文件。

### 决定控制模式

| 事件 | 决定模式 | 关键字段 |
|------|---------|---------|
| `UserPromptSubmit`、`UserPromptExpansion`、`PostToolUse`、`PostToolUseFailure`、`PostToolBatch`、`Stop`、`SubagentStop`、`ConfigChange`、`PreCompact` | 顶级 `decision` | `decision: "block"`、`reason` |
| `TeammateIdle`、`TaskCreated`、`TaskCompleted` | 退出代码或 `continue: false` | 退出代码 2 或 `{"continue": false, "stopReason": "..."}` |
| `PreToolUse` | `hookSpecificOutput` | `permissionDecision`（allow/deny/ask/defer）、`permissionDecisionReason` |
| `PermissionRequest` | `hookSpecificOutput` | `decision.behavior`（allow/deny） |
| `PermissionDenied` | `hookSpecificOutput` | `retry: true` 告诉模型它可能重试 |
| `WorktreeCreate` | 路径返回 | stdout 上打印路径或 `hookSpecificOutput.worktreePath` |
| `Elicitation` | `hookSpecificOutput` | `action`（accept/decline/cancel）、`content` |

### PreToolUse 决定控制（最丰富）

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow",
    "permissionDecisionReason": "My reason here",
    "updatedInput": {
      "field_to_modify": "new value"
    },
    "additionalContext": "Current environment: production. Proceed with caution."
  }
}
```

**决定优先级**：`deny` > `defer` > `ask` > `allow`

**四个结果**：
- `allow`：绕过权限提示
- `deny`：防止工具调用
- `ask`：提示用户确认
- `defer`：优雅地退出，以便工具稍后可以恢复（仅非交互模式）

---

## 路径引用技巧

使用环境变量按项目或插件根目录引用 hook 脚本：

| 变量 | 说明 |
|------|------|
| `$CLAUDE_PROJECT_DIR` | 项目根目录。用引号包装以处理包含空格的路径 |
| `${CLAUDE_PLUGIN_ROOT}` | 插件的安装目录，用于与插件捆绑的脚本 |
| `${CLAUDE_PLUGIN_DATA}` | 插件的持久数据目录，用于应该在插件更新后保留的依赖项和状态 |

---

## 禁用或移除 Hooks

- 要移除 hook，请从设置 JSON 文件中删除其条目
- 要临时禁用所有 hooks，在设置文件中设置 `"disableAllHooks": true`
- 没有办法在保持 hook 在配置中的同时禁用单个 hook
- `disableAllHooks` 遵守托管设置层次结构

---

## SessionStart Hook 持久化环境变量技巧

SessionStart hooks 可以访问 `CLAUDE_ENV_FILE` 环境变量，该变量提供一个文件路径，您可以在其中为后续 Bash 命令持久化环境变量：

```bash
#!/bin/bash
if [ -n "$CLAUDE_ENV_FILE" ]; then
  echo 'export NODE_ENV=production' >> "$CLAUDE_ENV_FILE"
  echo 'export DEBUG_LOG=true' >> "$CLAUDE_ENV_FILE"
  echo 'export PATH="$PATH:./node_modules/.bin"' >> "$CLAUDE_ENV_FILE"
fi
exit 0
```

### 捕获设置命令中的所有环境更改

```bash
#!/bin/bash
ENV_BEFORE=$(export -p | sort)
source ~/.nvm/nvm.sh
nvm use 20
if [ -n "$CLAUDE_ENV_FILE" ]; then
  ENV_AFTER=$(export -p | sort)
  comm -13 <(echo "$ENV_BEFORE") <(echo "$ENV_AFTER") >> "$CLAUDE_ENV_FILE"
fi
exit 0
```

---

## `/hooks` 菜单

在 Claude Code 中键入 `/hooks` 以打开您配置的 hooks 的只读浏览器。

### 菜单显示

- 每个 hook 事件及其配置的 hooks 计数
- 匹配器详情
- 每个 hook 处理程序的完整详细信息
- Hook 类型前缀：`[command]`、`[prompt]`、`[agent]`、`[http]`、`[mcp_tool]`
- 源标识：`[User]`、`[Project]`、`[Local]`、`[Plugin]`、`[Session]`、`[Built-in]`

> 菜单是只读的：要添加、修改或移除 hooks，请直接编辑设置 JSON 或要求 Claude 进行更改。
