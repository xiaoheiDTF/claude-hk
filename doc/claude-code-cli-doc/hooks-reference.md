# Claude Code Hooks 完整参考手册

> 本文档基于 `src/entrypoints/sdk/coreSchemas.ts`、`src/types/hooks.ts`、`src/utils/hooks.ts`、`src/utils/hooks/hooksConfigManager.ts` 源码生成。
>
> 所有 Hook 的输入通过 **stdin** 以 JSON 传递，输出通过 **stdout** 返回（command 类型 hook）。

---

## 目录

- [公共基础字段](#公共基础字段)
- [通用 JSON 输出字段](#通用-json-输出字段)
- [01 - SessionStart](#01---sessionstart)
- [02 - Setup](#02---setup)
- [03 - UserPromptSubmit](#03---userpromptsubmit)
- [05 - PreToolUse](#05---pretooluse)
- [06 - PermissionRequest](#06---permissionrequest)
- [07 - PermissionDenied](#07---permissiondenied)
- [08 - PostToolUse](#08---posttooluse)
- [09 - PostToolUseFailure](#09---posttoolusefailure)
- [11 - Notification](#11---notification)
- [12 - SubagentStart](#12---subagentstart)
- [13 - SubagentStop](#13---subagentstop)
- [14 - TaskCreated](#14---taskcreated)
- [15 - TaskCompleted](#15---taskcompleted)
- [16 - Stop](#16---stop)
- [17 - StopFailure](#17---stopfailure)
- [18 - TeammateIdle](#18---teammateidle)
- [19 - InstructionsLoaded](#19---instructionsloaded)
- [20 - ConfigChange](#20---configchange)
- [21 - CwdChanged](#21---cwdchanged)
- [22 - FileChanged](#22---filechanged)
- [23 - WorktreeCreate](#23---worktreecreate)
- [24 - WorktreeRemove](#24---worktreeremove)
- [25 - PreCompact](#25---precompact)
- [26 - PostCompact](#26---postcompact)
- [27 - Elicitation](#27---elicitation)
- [28 - ElicitationResult](#28---elicitationresult)
- [29 - SessionEnd](#29---sessionend)
- [settings.json 配置示例](#settingsjson-配置示例)

---

## 公共基础字段

每个 Hook 事件的 stdin JSON 都包含以下基础字段（定义于 `BaseHookInputSchema`，`coreSchemas.ts:387-411`）：

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01_10-00-00.jsonl",
  "cwd": "/home/user/my-project",
  "permission_mode": "default",
  "agent_id": "agent-abc123",
  "agent_type": "general-purpose"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `session_id` | string | ✅ | 当前会话的唯一标识符 |
| `transcript_path` | string | ✅ | 会话 transcript 文件的绝对路径 |
| `cwd` | string | ✅ | 当前工作目录的绝对路径 |
| `permission_mode` | string | ❌ | 当前权限模式（如 `default`、`plan`、`auto`、`dontAsk` 等） |
| `agent_id` | string | ❌ | 子 agent 标识符。仅在子 agent 内触发时存在。主线程中不存在（即使 `--agent` 模式） |
| `agent_type` | string | ❌ | agent 类型名（如 `general-purpose`、`code-reviewer`）。子 agent 中与 `agent_id` 一起存在；`--agent` 模式主线程中单独存在 |

---

## 通用 JSON 输出字段

所有 Hook 事件在 exit 0 且 stdout 以 `{` 开头时，均可使用以下通用字段（定义于 `SyncHookJSONOutputSchema`，`coreSchemas.ts:907-935`）：

```json
{
  "continue": true,
  "suppressOutput": false,
  "stopReason": "操作已停止，原因：xxx",
  "decision": "approve",
  "reason": "批准原因说明",
  "systemMessage": "显示给用户的警告信息",
  "hookSpecificOutput": { ... }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `continue` | boolean | ❌ | 是否继续后续流程，默认 `true`。设为 `false` 会停止处理，但具体行为因事件而异 |
| `suppressOutput` | boolean | ❌ | 是否隐藏 stdout 输出，默认 `false`。设为 `true` 则 hook 的输出不会出现在对话中 |
| `stopReason` | string | ❌ | 当 `continue: false` 时显示的停止原因消息 |
| `decision` | string | ❌ | 通用决策：`"approve"`（批准）或 `"block"`（阻塞）。效果等同于 exit code 2 |
| `reason` | string | ❌ | `decision` 的解释说明 |
| `systemMessage` | string | ❌ | 警告消息，用户和 Claude 都能看到。作为 `hook_system_message` 附件注入对话 |
| `hookSpecificOutput` | object | ❌ | 事件特定的输出结构，不同事件有不同的 schema（见各事件详情） |

**最简合法 JSON 输出**：

```json
{}
```

**异步模式输出**（hook 以此作为第一行 stdout 即进入异步后台运行）：

```json
{
  "async": true,
  "asyncTimeout": 30000
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `async` | true | ✅ | 必须为字面量 `true`，标识这是异步 hook |
| `asyncTimeout` | number | ❌ | 异步超时时间（毫秒） |

---

## 01 - SessionStart

**作用**：新会话启动时触发。可用于注入初始上下文、设置环境变量、注册文件监听路径。

**触发时机**：
- `startup` — 首次启动会话
- `resume` — 恢复已有会话（`--resume`）
- `clear` — 用户执行 `/clear` 清除对话
- `compact` — 压缩完成后重新初始化

**Matcher 字段**：`source`，值：`startup` | `resume` | `clear` | `compact`

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "SessionStart",
  "source": "startup",
  "agent_type": "general-purpose",
  "model": "claude-sonnet-4-6"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `source` | string | ✅ | 会话启动来源：`startup`（首次启动）、`resume`（恢复）、`clear`（清除）、`compact`（压缩后） |
| `agent_type` | string | ❌ | agent 类型名，`--agent` 模式主线程或子 agent 中存在 |
| `model` | string | ❌ | 当前使用的模型 ID |

### 输出 (stdout JSON)

```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "你是一个资深安全审计专家。请从安全角度审视所有代码变更，特别关注 SQL 注入、XSS 和 CSRF 漏洞。",
    "initialUserMessage": "请帮我审查项目中的安全漏洞",
    "watchPaths": [
      "/home/user/my-project/src/config.json",
      "/home/user/my-project/.env"
    ]
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"SessionStart"` |
| `additionalContext` | string | ❌ | 注入给 Claude 的额外上下文。可以是角色设定、项目规范、注意事项等。Claude 会看到这段文本 |
| `initialUserMessage` | string | ❌ | 替换用户的首条消息。如果设置，会替代用户实际输入作为第一条消息发送给 Claude |
| `watchPaths` | string[] | ❌ | 注册文件监听路径（绝对路径）。当这些文件变化时触发 `FileChanged` hook |

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout 内容显示给 Claude（注入为对话上下文） |
| 2 | **阻塞被忽略**（SessionStart 的阻塞错误不会阻止会话启动） |
| 其他 | stderr 仅显示给用户 |

### 典型用例

```bash
#!/bin/bash
# 在项目启动时注入项目规范和角色设定
cat << 'EOF'
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "项目规范：\n1. 使用 React + TypeScript + Vite 技术栈\n2. 代码风格遵循 ESLint + Prettier 配置\n3. 所有 API 调用通过 src/api/client.ts 统一处理\n4. 测试框架使用 Vitest + Testing Library",
    "watchPaths": ["/project/.env", "/project/package.json"]
  }
}
EOF
exit 0
```

### settings.json 配置

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup",
        "hooks": [{ "type": "command", "command": "bash /path/to/init.sh" }]
      }
    ]
  }
}
```

---

## 02 - Setup

**作用**：仓库初始化或维护时触发。用于项目级别的初始化脚本（安装依赖、生成配置、健康检查等）。

**触发时机**：
- `init` — 首次打开项目（`/init`）
- `maintenance` — 后续维护触发

**Matcher 字段**：`trigger`，值：`init` | `maintenance`

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "Setup",
  "trigger": "init"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `trigger` | string | ✅ | 触发类型：`init`（首次初始化）或 `maintenance`（日常维护） |

### 输出 (stdout JSON)

```json
{
  "hookSpecificOutput": {
    "hookEventName": "Setup",
    "additionalContext": "项目已初始化。依赖已安装：npm install 完成。检测到 3 个过期的依赖包。"
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"Setup"` |
| `additionalContext` | string | ❌ | 注入给 Claude 的额外上下文（如初始化结果摘要） |

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout 显示给 Claude |
| 2 | **阻塞被忽略** |
| 其他 | stderr 仅显示给用户 |

### 典型用例

```bash
#!/bin/bash
# 首次初始化时检查并安装依赖
if [ ! -d "node_modules" ]; then
  npm install 2>&1
  echo "依赖安装完成"
fi
exit 0
```

### settings.json 配置

```json
{
  "hooks": {
    "Setup": [
      {
        "matcher": "init",
        "hooks": [{ "type": "command", "command": "bash scripts/setup.sh" }]
      }
    ]
  }
}
```

---

## 03 - UserPromptSubmit

**作用**：用户提交 prompt 后、发送给 LLM 之前触发。是最常用的注入点，可用于注入角色设定、动态上下文、输入验证、拦截不当内容。

**触发时机**：每次用户提交输入（按回车）时，在斜杠命令处理和附件处理之后触发。

**Matcher 字段**：无（所有配置的 hook 都会匹配）

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "permission_mode": "default",
  "hook_event_name": "UserPromptSubmit",
  "prompt": "帮我写一个用户登录页面，需要支持邮箱和手机号登录"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `prompt` | string | ✅ | 用户输入的原始文本。包含完整的用户 prompt 内容 |

### 输出 (stdout JSON)

**方式一：使用 hookSpecificOutput（推荐）**

```json
{
  "hookSpecificOutput": {
    "hookEventName": "UserPromptSubmit",
    "additionalContext": "你是一个资深前端架构师。请遵循以下规范：\n1. 使用 React 函数式组件 + TypeScript\n2. 使用 Tailwind CSS 进行样式设计\n3. 所有表单需要客户端验证\n4. API 调用使用 fetch + async/await"
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"UserPromptSubmit"` |
| `additionalContext` | string | ❌ | 注入给 Claude 的提示词/角色设定/上下文。Claude 会在用户 prompt 之后看到这段文本 |

**方式二：使用通用控制字段**

```json
{
  "decision": "block",
  "reason": "检测到敏感信息（API Key），不允许提交"
}
```

```json
{
  "continue": false,
  "stopReason": "当前处于只读模式，不允许执行修改操作"
}
```

```json
{
  "systemMessage": "注意：你正在生产环境中操作，请谨慎",
  "hookSpecificOutput": {
    "hookEventName": "UserPromptSubmit",
    "additionalContext": "当前为生产环境，所有操作需要格外谨慎。"
  }
}
```

**方式三：纯文本输出（最简单）**

```bash
#!/bin/bash
echo "你是一个资深安全审计专家。请从安全角度审视所有代码变更。"
exit 0
```

纯文本 stdout 会作为 `hook_success` 附件注入对话，Claude 可见。效果等价于 `additionalContext`。

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout 显示给 Claude（作为对话上下文注入） |
| 2 | **阻塞！擦除原始 prompt，不发送给 LLM**。用户看到错误消息和原始 prompt |
| 其他 | stderr 仅显示给用户 |

### 典型用例

**注入角色设定：**
```bash
#!/bin/bash
cat << 'EOF'
{
  "hookSpecificOutput": {
    "hookEventName": "UserPromptSubmit",
    "additionalContext": "角色：资深安全审计专家。请从安全角度审视所有代码，关注 OWASP Top 10 漏洞。"
  }
}
EOF
exit 0
```

**输入验证/拦截：**
```python
#!/usr/bin/env python3
import json, sys

data = json.loads(sys.stdin.read())
prompt = data.get("prompt", "")

# 检测敏感信息
if "password" in prompt.lower() and "=" in prompt:
    print(json.dumps({"decision": "block", "reason": "检测到可能的密码明文，不允许提交"}))
    sys.exit(0)

# 正常通过
sys.exit(0)
```

### settings.json 配置

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "matcher": "",
        "hooks": [
          { "type": "command", "command": "python3 /path/to/validate_and_inject.py" }
        ]
      }
    ]
  }
}
```

---

## 05 - PreToolUse

**作用**：工具执行之前触发。可用于权限控制（批准/拒绝/修改工具调用）、审计日志、修改工具输入参数。

**触发时机**：每次 Claude 调用工具（如 Read、Write、Edit、Bash 等）之前。

**Matcher 字段**：`tool_name`，值：所有工具名（如 `Read`、`Write`、`Edit`、`Bash`、`Glob`、`Grep` 等）

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "PreToolUse",
  "tool_name": "Bash",
  "tool_input": {
    "command": "rm -rf /tmp/test && npm run build"
  },
  "tool_use_id": "toolu_01ABCDEF"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `tool_name` | string | ✅ | 被调用的工具名称（如 `Read`、`Write`、`Edit`、`Bash`、`Glob`、`Grep` 等） |
| `tool_input` | object | ✅ | 工具调用的输入参数，结构因工具而异 |
| `tool_use_id` | string | ✅ | 工具调用的唯一标识符 |

### 输出 (stdout JSON)

**批准并修改输入：**

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow",
    "permissionDecisionReason": "只读操作，允许执行",
    "updatedInput": {
      "command": "rm -rf /tmp/test && npm run build --verbose"
    },
    "additionalContext": "已将构建命令修改为 verbose 模式"
  }
}
```

**拒绝工具调用：**

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "检测到危险的 rm -rf 命令，不允许执行"
  }
}
```

**要求用户确认：**

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "ask",
    "permissionDecisionReason": "此操作会修改生产环境配置文件，需要用户确认"
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"PreToolUse"` |
| `permissionDecision` | string | ❌ | 权限决策：`"allow"`（批准）、`"deny"`（拒绝）、`"ask"`（要求用户确认） |
| `permissionDecisionReason` | string | ❌ | 权限决策的原因说明 |
| `updatedInput` | object | ❌ | 修改后的工具输入参数。会替换原始的 `tool_input`（仅 `allow` 或 `ask` 时生效） |
| `additionalContext` | string | ❌ | 额外上下文，显示给 Claude |

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout/stderr **不显示**给用户或 Claude。但 JSON 中的 `permissionDecision` 等字段会生效 |
| 2 | 显示 stderr 给 model 并**阻止工具调用**（等效于 `deny`） |
| 其他 | stderr 仅显示给用户，但工具**继续执行**（不阻止） |

### 典型用例

**阻止危险 Bash 命令：**
```python
#!/usr/bin/env python3
import json, sys

data = json.loads(sys.stdin.read())
if data.get("tool_name") == "Bash":
    cmd = data.get("tool_input", {}).get("command", "")
    if "rm -rf /" in cmd or "DROP TABLE" in cmd.upper():
        print(json.dumps({
            "hookSpecificOutput": {
                "hookEventName": "PreToolUse",
                "permissionDecision": "deny",
                "permissionDecisionReason": f"检测到危险命令: {cmd[:50]}"
            }
        }))
        sys.exit(0)
sys.exit(0)
```

### settings.json 配置

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{ "type": "command", "command": "python3 /path/to/audit_bash.py" }]
      },
      {
        "matcher": "Write|Edit",
        "hooks": [{ "type": "command", "command": "python3 /path/to/check_write.py" }]
      }
    ]
  }
}
```

---

## 06 - PermissionRequest

**作用**：当权限对话框弹出时触发。可用于自动批准/拒绝权限请求，实现自动化的权限策略。

**触发时机**：每次需要用户确认工具调用权限时（弹出权限对话框）。

**Matcher 字段**：`tool_name`，值：所有工具名

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "PermissionRequest",
  "tool_name": "Bash",
  "tool_input": {
    "command": "npm run build"
  },
  "permission_suggestions": [
    {
      "tool": "Bash",
      "pattern": "npm run *",
      "behavior": "allow"
    }
  ]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `tool_name` | string | ✅ | 请求权限的工具名称 |
| `tool_input` | object | ✅ | 工具调用的输入参数 |
| `permission_suggestions` | array | ❌ | 系统建议的权限规则列表，每个包含 `tool`、`pattern`、`behavior` |

### 输出 (stdout JSON)

**自动批准：**

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PermissionRequest",
    "decision": {
      "behavior": "allow",
      "updatedInput": {
        "command": "npm run build --verbose"
      },
      "updatedPermissions": [
        {
          "tool": "Bash",
          "pattern": "npm run *",
          "behavior": "allow"
        }
      ]
    }
  }
}
```

**自动拒绝：**

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PermissionRequest",
    "decision": {
      "behavior": "deny",
      "message": "生产环境中不允许执行构建命令",
      "interrupt": true
    }
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"PermissionRequest"` |
| `decision` | object | ❌ | 权限决策对象 |
| `decision.behavior` | string | ✅ | `"allow"` 或 `"deny"` |
| `decision.updatedInput` | object | ❌ | 仅 `allow` 时可用。修改后的工具输入参数 |
| `decision.updatedPermissions` | array | ❌ | 仅 `allow` 时可用。持久化的权限规则，后续相同模式自动通过 |
| `decision.message` | string | ❌ | 仅 `deny` 时可用。拒绝原因 |
| `decision.interrupt` | boolean | ❌ | 仅 `deny` 时可用。是否中断当前操作流程 |

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | 使用 hook 返回的 decision（如果提供了 JSON） |
| 其他 | stderr 仅显示给用户 |

### settings.json 配置

```json
{
  "hooks": {
    "PermissionRequest": [
      {
        "matcher": "Bash",
        "hooks": [{ "type": "command", "command": "python3 /path/to/auto_approve.py" }]
      }
    ]
  }
}
```

---

## 07 - PermissionDenied

**作用**：当自动模式分类器拒绝工具调用后触发。可用于通知模型可以重试，或记录审计日志。

**触发时机**：自动模式（auto mode）下分类器拒绝工具调用后。

**Matcher 字段**：`tool_name`，值：所有工具名

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "PermissionDenied",
  "tool_name": "Bash",
  "tool_input": {
    "command": "curl https://external-api.com/data"
  },
  "tool_use_id": "toolu_01ABCDEF",
  "reason": "Command matches deny pattern: curl *"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `tool_name` | string | ✅ | 被拒绝的工具名称 |
| `tool_input` | object | ✅ | 被拒绝的工具输入参数 |
| `tool_use_id` | string | ✅ | 工具调用的唯一标识符 |
| `reason` | string | ✅ | 拒绝原因说明 |

### 输出 (stdout JSON)

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PermissionDenied",
    "retry": true
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"PermissionDenied"` |
| `retry` | boolean | ❌ | 设为 `true` 告诉模型可以重试此工具调用（可能用不同参数） |

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout 在 transcript 模式可见 |
| 其他 | stderr 仅显示给用户 |

---

## 08 - PostToolUse

**作用**：工具成功执行后触发。可用于后处理、审计日志、修改 MCP 工具输出。

**触发时机**：每次工具调用成功完成后。

**Matcher 字段**：`tool_name`，值：所有工具名

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "PostToolUse",
  "tool_name": "Read",
  "tool_input": {
    "file_path": "/home/user/my-project/src/index.ts"
  },
  "tool_response": "// file content here...\nexport function main() {\n  console.log('hello');\n}",
  "tool_use_id": "toolu_01ABCDEF"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `tool_name` | string | ✅ | 执行的工具名称 |
| `tool_input` | object | ✅ | 工具的输入参数 |
| `tool_response` | any | ✅ | 工具的返回结果（内容因工具而异） |
| `tool_use_id` | string | ✅ | 工具调用的唯一标识符 |

### 输出 (stdout JSON)

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "文件读取成功，包含 TypeScript 代码。",
    "updatedMCPToolOutput": {
      "result": "modified output for MCP tool"
    }
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"PostToolUse"` |
| `additionalContext` | string | ❌ | 注入给 Claude 的额外上下文 |
| `updatedMCPToolOutput` | any | ❌ | 替换 MCP 工具的输出。仅对 MCP 工具有效，可完全替换工具返回给 Claude 的结果 |

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout 在 transcript 模式 (ctrl+o) 可见 |
| 2 | 显示 stderr 给 model **立即**（不等到 turn 结束） |
| 其他 | stderr 仅显示给用户 |

### settings.json 配置

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{ "type": "command", "command": "python3 /path/to/log_bash.py" }]
      }
    ]
  }
}
```

---

## 09 - PostToolUseFailure

**作用**：工具执行失败后触发。可用于错误恢复建议、审计日志。

**触发时机**：每次工具调用失败后。

**Matcher 字段**：`tool_name`，值：所有工具名

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "PostToolUseFailure",
  "tool_name": "Bash",
  "tool_input": {
    "command": "npm test"
  },
  "tool_use_id": "toolu_01ABCDEF",
  "error": "Command exited with code 1: 2 tests failed",
  "is_interrupt": false
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `tool_name` | string | ✅ | 失败的工具名称 |
| `tool_input` | object | ✅ | 工具的输入参数 |
| `tool_use_id` | string | ✅ | 工具调用的唯一标识符 |
| `error` | string | ✅ | 错误消息描述 |
| `is_interrupt` | boolean | ❌ | 是否因为用户中断（Ctrl+C）导致的失败 |

### 输出 (stdout JSON)

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUseFailure",
    "additionalContext": "测试失败可能与最近的类型变更有关，建议检查 src/types.ts"
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"PostToolUseFailure"` |
| `additionalContext` | string | ❌ | 注入给 Claude 的额外上下文（如恢复建议） |

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout 在 transcript 模式可见 |
| 2 | 显示 stderr 给 model **立即** |
| 其他 | stderr 仅显示给用户 |

---

## 11 - Notification

**作用**：发送通知时触发。可用于集成外部通知系统（Slack、钉钉等）。

**触发时机**：Claude Code 发送通知时（权限提示、空闲提示、认证成功等）。

**Matcher 字段**：`notification_type`，值：`permission_prompt` | `idle_prompt` | `auth_success` | `elicitation_dialog` | `elicitation_complete` | `elicitation_response`

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "Notification",
  "message": "Tool approval required for: Bash(npm run build)",
  "title": "Permission Request",
  "notification_type": "permission_prompt"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `message` | string | ✅ | 通知消息内容 |
| `title` | string | ❌ | 通知标题 |
| `notification_type` | string | ✅ | 通知类型（见上方 matcher 值列表） |

### 输出 (stdout JSON)

```json
{
  "hookSpecificOutput": {
    "hookEventName": "Notification",
    "additionalContext": "已转发通知到 Slack #claude-alerts 频道"
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"Notification"` |
| `additionalContext` | string | ❌ | 额外上下文 |

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout/stderr **不显示** |
| 其他 | stderr 仅显示给用户 |

---

## 12 - SubagentStart

**作用**：子 agent（Agent 工具调用）启动时触发。可为子 agent 注入特定的上下文或角色设定。

**触发时机**：每次 Claude 通过 Agent 工具启动子 agent 时。

**Matcher 字段**：`agent_type`，值：agent 类型名

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "SubagentStart",
  "agent_id": "agent-abc123",
  "agent_type": "general-purpose"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `agent_id` | string | ✅ | 子 agent 的唯一标识符 |
| `agent_type` | string | ✅ | 子 agent 的类型名（如 `general-purpose`、`code-reviewer`、`Explore` 等） |

### 输出 (stdout JSON)

```json
{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStart",
    "additionalContext": "你是代码审查 agent。请关注代码质量、性能和安全性。"
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"SubagentStart"` |
| `additionalContext` | string | ❌ | 注入给子 agent 的额外上下文 |

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout 显示给子 agent |
| 2 | **阻塞被忽略** |
| 其他 | stderr 仅显示给用户 |

---

## 13 - SubagentStop

**作用**：子 agent 即将结束响应时触发。可用于验证子 agent 的工作是否完成。

**触发时机**：子 agent 完成工作、即将返回结果前。

**Matcher 字段**：`agent_type`，值：agent 类型名

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "SubagentStop",
  "stop_hook_active": true,
  "agent_id": "agent-abc123",
  "agent_transcript_path": "/home/user/.claude/projects/xxx/agents/agent-abc123.jsonl",
  "agent_type": "general-purpose",
  "last_assistant_message": "我已经完成了代码审查，发现了 3 个潜在问题..."
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `stop_hook_active` | boolean | ✅ | 是否有 stop hook 激活 |
| `agent_id` | string | ✅ | 子 agent 的唯一标识符 |
| `agent_transcript_path` | string | ✅ | 子 agent 的 transcript 文件路径（可用于读取完整对话记录） |
| `agent_type` | string | ✅ | 子 agent 的类型名 |
| `last_assistant_message` | string | ❌ | 子 agent 最后一条消息的文本内容。避免需要读取 transcript 文件 |

### 输出

**没有专属的 hookSpecificOutput 分支**。仅通用字段可用。

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout/stderr **不显示** |
| 2 | 显示 stderr 给子 agent 并**继续运行**（不让它停止） |
| 其他 | stderr 仅显示给用户 |

### 典型用例

```bash
#!/bin/bash
# 检查子 agent 是否真的完成了工作
# exit 2 可以让子 agent 继续运行
python3 -c "
import json, sys
data = json.loads(sys.stdin.read())
last_msg = data.get('last_assistant_message', '')
if 'TODO' in last_msg or '未完成' in last_msg:
    sys.stderr.write('子 agent 仍有未完成的工作，继续运行')
    sys.exit(2)
sys.exit(0)
"
```

---

## 14 - TaskCreated

**作用**：任务被创建时触发。可用于通知外部系统、验证任务内容、阻止不当任务。

**触发时机**：在团队协作中创建任务时。

**Matcher 字段**：无

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "TaskCreated",
  "task_id": "42",
  "task_subject": "实现用户登录功能",
  "task_description": "需要实现 JWT 认证的登录功能，支持邮箱和手机号登录",
  "teammate_name": "researcher",
  "team_name": "my-project"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `task_id` | string | ✅ | 任务的唯一标识符 |
| `task_subject` | string | ✅ | 任务标题 |
| `task_description` | string | ❌ | 任务的详细描述 |
| `teammate_name` | string | ❌ | 创建任务的 teammate 名称 |
| `team_name` | string | ❌ | 所属团队名称 |

### 输出

**没有专属的 hookSpecificOutput 分支**。仅通用字段可用。

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout/stderr **不显示** |
| 2 | 显示 stderr 给 model 并**阻止任务创建** |
| 其他 | stderr 仅显示给用户 |

---

## 15 - TaskCompleted

**作用**：任务被标记完成时触发。可用于验证任务是否真的完成、通知外部系统。

**触发时机**：在团队协作中标记任务完成时。

**Matcher 字段**：无

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "TaskCompleted",
  "task_id": "42",
  "task_subject": "实现用户登录功能",
  "task_description": "需要实现 JWT 认证的登录功能，支持邮箱和手机号登录",
  "teammate_name": "researcher",
  "team_name": "my-project"
}
```

字段同 TaskCreated。

### 输出

**没有专属的 hookSpecificOutput 分支**。仅通用字段可用。

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout/stderr **不显示** |
| 2 | 显示 stderr 给 model 并**阻止任务完成标记** |
| 其他 | stderr 仅显示给用户 |

---

## 16 - Stop

**作用**：Claude 即将结束当前回合响应时触发。最常用的验证 hook，可用于检查 Claude 的工作是否真正完成。

**触发时机**：Claude 完成一个回合（turn）的响应后、返回控制权给用户前。

**Matcher 字段**：无

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "Stop",
  "stop_hook_active": true,
  "last_assistant_message": "我已经完成了登录功能的实现。主要修改了以下文件：\n1. src/auth/login.ts - 新增登录逻辑\n2. src/api/client.ts - 添加认证中间件\n3. tests/auth.test.ts - 编写测试用例"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `stop_hook_active` | boolean | ✅ | 是否有 stop hook 激活（用于模型判断是否应提前输出） |
| `last_assistant_message` | string | ❌ | Claude 最后一条消息的文本内容 |

### 输出

**没有专属的 hookSpecificOutput 分支**。仅通用字段可用。

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout/stderr **不显示** |
| 2 | 显示 stderr 给 model 并**继续对话**（Claude 不会停止，会继续工作） |
| 其他 | stderr 仅显示给用户 |

### 典型用例

```bash
#!/bin/bash
# 验证 Claude 是否真的完成了工作
python3 -c "
import json, sys
data = json.loads(sys.stdin.read())
msg = data.get('last_assistant_message', '')

# 如果 Claude 声称修改了文件但未运行测试，让它继续
if '修改' in msg and '测试' not in msg and 'test' not in msg.lower():
    sys.stderr.write('请运行相关测试来验证修改是否正确')
    sys.exit(2)
sys.exit(0)
"
```

---

## 17 - StopFailure

**作用**：因 API 错误导致回合异常结束时触发。仅用于监控/日志，无法控制行为。

**触发时机**：因 API 错误（限流、认证失败、计费错误等）非正常结束时，替代 Stop 触发。

**Matcher 字段**：`error`，值：`rate_limit` | `authentication_failed` | `billing_error` | `invalid_request` | `server_error` | `max_output_tokens` | `unknown`

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "StopFailure",
  "error": "rate_limit",
  "error_details": "Rate limit exceeded: too many requests in 1 minute",
  "last_assistant_message": "我正在处理您的请求..."
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `error` | string | ✅ | 错误类型（见上方 matcher 值列表） |
| `error_details` | string | ❌ | 错误详情描述 |
| `last_assistant_message` | string | ❌ | Claude 最后一条消息的文本内容 |

### 输出

**Fire-and-forget**：hook 的所有输出和退出码**全部被忽略**。仅用于日志/监控。

### 退出码

所有退出码均被忽略。

---

## 18 - TeammateIdle

**作用**：teammate 即将进入空闲状态时触发。可用于给 teammate 分配新任务、阻止空闲。

**触发时机**：团队协作中 teammate 完成当前工作即将空闲时。

**Matcher 字段**：无

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "TeammateIdle",
  "teammate_name": "researcher",
  "team_name": "my-project"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `teammate_name` | string | ✅ | 即将空闲的 teammate 名称 |
| `team_name` | string | ✅ | 所属团队名称 |

### 输出

**没有专属的 hookSpecificOutput 分支**。仅通用字段可用。

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout/stderr **不显示** |
| 2 | 显示 stderr 给 teammate 并**阻止空闲**（teammate 继续工作） |
| 其他 | stderr 仅显示给用户 |

---

## 19 - InstructionsLoaded

**作用**：指令文件（CLAUDE.md 或 rule）被加载时触发。仅用于可观测性（监控哪些文件被加载、何时被加载）。

**触发时机**：每次加载指令文件时。

**Matcher 字段**：`load_reason`，值：`session_start` | `nested_traversal` | `path_glob_match` | `include` | `compact`

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "InstructionsLoaded",
  "file_path": "/home/user/my-project/CLAUDE.md",
  "memory_type": "Project",
  "load_reason": "session_start",
  "globs": ["src/**/*.ts"],
  "trigger_file_path": "/home/user/my-project/src/index.ts",
  "parent_file_path": "/home/user/my-project/CLAUDE.md"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `file_path` | string | ✅ | 被加载的指令文件绝对路径 |
| `memory_type` | string | ✅ | 内存类型：`User`（用户级）、`Project`（项目级）、`Local`（本地级）、`Managed`（托管级） |
| `load_reason` | string | ✅ | 加载原因：`session_start`（会话启动）、`nested_traversal`（嵌套遍历）、`path_glob_match`（路径 glob 匹配）、`include`（被 include 引用）、`compact`（压缩后重新加载） |
| `globs` | string[] | ❌ | 指令文件 `paths:` frontmatter 中匹配当前文件的 glob 模式 |
| `trigger_file_path` | string | ❌ | 触发此加载的文件路径（Claude 触碰的文件导致此指令被加载） |
| `parent_file_path` | string | ❌ | 通过 `@-include` 引用此文件的父文件路径 |

### 输出

**仅可观测**，不支持阻塞，没有 hookSpecificOutput。

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | 成功 |
| 其他 | stderr 仅显示给用户 |

---

## 20 - ConfigChange

**作用**：会话期间配置文件变更时触发。可用于审计配置变更、阻止不当配置。

**触发时机**：运行时配置文件（settings.json、skills 等）被修改时。

**Matcher 字段**：`source`，值：`user_settings` | `project_settings` | `local_settings` | `policy_settings` | `skills`

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "ConfigChange",
  "source": "project_settings",
  "file_path": "/home/user/my-project/.claude/settings.json"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `source` | string | ✅ | 配置来源：`user_settings`（用户级 `~/.claude/settings.json`）、`project_settings`（项目级 `.claude/settings.json`）、`local_settings`（本地级 `.claude/settings.local.json`）、`policy_settings`（策略配置）、`skills`（技能文件） |
| `file_path` | string | ❌ | 变更的配置文件绝对路径 |

### 输出

**没有专属的 hookSpecificOutput 分支**。仅通用字段可用。

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | 允许配置变更 |
| 2 | **阻止配置变更**应用到当前会话 |
| 其他 | stderr 仅显示给用户 |

---

## 21 - CwdChanged

**作用**：工作目录变更后触发。可用于设置环境变量、注册文件监听。

**触发时机**：通过 `cd` 命令或 `CwdChanged` 事件导致工作目录变化后。

**Matcher 字段**：无

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project-new",
  "hook_event_name": "CwdChanged",
  "old_cwd": "/home/user/my-project",
  "new_cwd": "/home/user/my-project-new"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `old_cwd` | string | ✅ | 变更前的工作目录 |
| `new_cwd` | string | ✅ | 变更后的工作目录 |

### 输出 (stdout JSON)

```json
{
  "hookSpecificOutput": {
    "hookEventName": "CwdChanged",
    "watchPaths": [
      "/home/user/my-project-new/.env",
      "/home/user/my-project-new/config.json"
    ]
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"CwdChanged"` |
| `watchPaths` | string[] | ❌ | 注册文件监听路径（绝对路径） |

> **特殊环境变量**：`CLAUDE_ENV_FILE` 可用。hook 可将 bash 环境变量定义写入此文件，后续 BashTool 命令会加载这些环境变量。

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | 成功 |
| 其他 | stderr 仅显示给用户 |

---

## 22 - FileChanged

**作用**：被监听的文件发生变更时触发。可用于自动重新加载配置、触发构建。

**触发时机**：通过 `watchPaths` 注册的文件发生变化（修改、新增、删除）时。

**Matcher 字段**：按文件名（basename）匹配

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "FileChanged",
  "file_path": "/home/user/my-project/.envrc",
  "event": "change"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `file_path` | string | ✅ | 变更文件的绝对路径 |
| `event` | string | ✅ | 变更类型：`change`（内容修改）、`add`（文件新增）、`unlink`（文件删除） |

### 输出 (stdout JSON)

```json
{
  "hookSpecificOutput": {
    "hookEventName": "FileChanged",
    "watchPaths": [
      "/home/user/my-project/.envrc",
      "/home/user/my-project/package.json"
    ]
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"FileChanged"` |
| `watchPaths` | string[] | ❌ | 动态更新监听路径列表（可新增或修改监听目标） |

> **特殊环境变量**：`CLAUDE_ENV_FILE` 可用。

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | 成功 |
| 其他 | stderr 仅显示给用户 |

---

## 23 - WorktreeCreate

**作用**：创建隔离 worktree 时触发。用于 VCS 无关的隔离机制。

**触发时机**：通过 `EnterWorktree` 或 Agent 隔离模式创建 worktree 时。

**Matcher 字段**：无

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "WorktreeCreate",
  "name": "my-feature-branch"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✅ | 建议的 worktree 名称/标识符 |

### 输出 (stdout)

**Command hook**：stdout 输出 worktree 的**绝对路径**（纯文本）。

```bash
#!/bin/bash
# 创建 worktree 并输出路径
mkdir -p /tmp/worktrees/my-feature
echo "/tmp/worktrees/my-feature"
exit 0
```

**JSON 输出**：

```json
{
  "hookSpecificOutput": {
    "hookEventName": "WorktreeCreate",
    "worktreePath": "/tmp/worktrees/my-feature"
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"WorktreeCreate"` |
| `worktreePath` | string | ✅ | 创建的 worktree 目录的**绝对路径** |

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | worktree 创建成功 |
| 其他 | 创建失败 |

---

## 24 - WorktreeRemove

**作用**：移除之前创建的 worktree 时触发。

**触发时机**：退出 worktree 或清理时。

**Matcher 字段**：无

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "WorktreeRemove",
  "worktree_path": "/tmp/worktrees/my-feature"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `worktree_path` | string | ✅ | 要移除的 worktree 目录的绝对路径 |

### 输出

**没有专属的 hookSpecificOutput 分支**。仅通用字段可用。

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | 移除成功 |
| 其他 | stderr 仅显示给用户 |

---

## 25 - PreCompact

**作用**：对话压缩（compaction）之前触发。可用于注入自定义压缩指令（控制摘要保留哪些信息）、阻止压缩。

**触发时机**：
- 用户执行 `/compact` 命令（`manual`）
- 自动压缩触发（`auto`）

**Matcher 字段**：`trigger`，值：`manual` | `auto`

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "PreCompact",
  "trigger": "manual",
  "custom_instructions": "请保留所有代码片段和错误日志"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `trigger` | string | ✅ | 触发类型：`manual`（用户手动 `/compact`）或 `auto`（自动压缩） |
| `custom_instructions` | string\|null | ✅ | 用户提供的自定义压缩指令，可能为 `null` |

### 输出 (stdout)

> ⚠️ **重要**：PreCompact **没有**专属的 `hookSpecificOutput` 分支。应使用**纯文本** stdout 输出压缩指令。

**推荐方式 — 纯文本 stdout**：

```bash
#!/bin/bash
cat << 'EOF'
请保留以下关键信息：
1. 项目使用 React + TypeScript + Vite 技术栈
2. 用户角色设定：资深安全审计专家
3. 所有 API 调用必须通过 src/api/client.ts 统一处理
4. 请保留最近的错误日志和修复历史
5. 请保留所有代码片段和函数签名
EOF
exit 0
```

stdout 文本会被收集为 `newCustomInstructions`，然后通过 `mergeHookInstructions()` 与用户指令合并，最终拼接到压缩 prompt 的 `Additional Instructions:` 部分。

合并顺序：`用户指令\n\nHook 指令`

**阻塞方式 — JSON**：

```json
{
  "decision": "block",
  "reason": "当前有未完成的调试任务，不允许压缩"
}
```

```json
{
  "continue": false,
  "stopReason": "上下文中包含重要的调试断点信息，暂不压缩"
}
```

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout → 追加为自定义压缩指令（拼到 `Additional Instructions:` 后面） |
| 2 | **阻止压缩**（整个 compaction 不执行） |
| 其他 | stderr 显示给用户，但**继续压缩**（不阻止） |

### 典型用例

**按 trigger 区分处理：**

```python
#!/usr/bin/env python3
import json, sys

data = json.loads(sys.stdin.read())
trigger = data.get("trigger", "manual")

if trigger == "auto":
    print("自动压缩：请保留最近 5 轮对话的完整内容，特别是错误信息和修复步骤")
elif trigger == "manual":
    print("手动压缩：请保留所有代码修改记录、用户反馈和待办事项")

sys.exit(0)
```

### settings.json 配置

```json
{
  "hooks": {
    "PreCompact": [
      {
        "matcher": "auto",
        "hooks": [{ "type": "command", "command": "bash /path/to/auto_compact.sh" }]
      },
      {
        "matcher": "manual",
        "hooks": [{ "type": "command", "command": "python3 /path/to/manual_compact.py" }]
      }
    ]
  }
}
```

---

## 26 - PostCompact

**作用**：对话压缩完成后触发。可用于查看压缩摘要、通知外部系统。

**触发时机**：压缩操作成功完成后。

**Matcher 字段**：`trigger`，值：`manual` | `auto`

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "PostCompact",
  "trigger": "manual",
  "compact_summary": "## 对话摘要\n\n### 主要请求和意图\n用户要求实现 JWT 认证的登录功能...\n\n### 关键技术概念\n- React + TypeScript\n- JWT Token\n- bcrypt 密码加密\n\n### 文件和代码\n- src/auth/login.ts: 登录逻辑实现\n- src/api/client.ts: 认证中间件\n\n### 待办事项\n- 添加手机号验证码登录\n- 编写集成测试"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `trigger` | string | ✅ | 触发类型：`manual` 或 `auto` |
| `compact_summary` | string | ✅ | 压缩生成的对话摘要全文 |

### 输出

**没有专属的 hookSpecificOutput 分支**。仅通用字段可用。

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | stdout 显示给用户 |
| 其他 | stderr 仅显示给用户 |

---

## 27 - Elicitation

**作用**：MCP 服务器请求用户输入时触发。可用于自动响应（批准/拒绝），跳过用户交互。

**触发时机**：MCP 服务器发起 elicitation 请求时。

**Matcher 字段**：`mcp_server_name`，值：MCP 服务器名称

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "Elicitation",
  "mcp_server_name": "my-mcp-server",
  "message": "请输入你的 API key 以继续",
  "mode": "form",
  "url": "https://example.com/auth",
  "elicitation_id": "eli-abc123",
  "requested_schema": {
    "type": "object",
    "properties": {
      "api_key": {
        "type": "string",
        "description": "Your API key"
      }
    },
    "required": ["api_key"]
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `mcp_server_name` | string | ✅ | 发起请求的 MCP 服务器名称 |
| `message` | string | ✅ | 请求消息内容（展示给用户的提示） |
| `mode` | string | ❌ | 交互模式：`form`（表单）或 `url`（URL 跳转） |
| `url` | string | ❌ | 当 mode 为 `url` 时的跳转地址 |
| `elicitation_id` | string | ❌ | elicitation 请求的唯一标识符 |
| `requested_schema` | object | ❌ | 请求的数据 schema（JSON Schema 格式） |

### 输出 (stdout JSON)

**自动接受并提供内容：**

```json
{
  "hookSpecificOutput": {
    "hookEventName": "Elicitation",
    "action": "accept",
    "content": {
      "api_key": "sk-xxxxxxxxxxxx"
    }
  }
}
```

**自动拒绝：**

```json
{
  "hookSpecificOutput": {
    "hookEventName": "Elicitation",
    "action": "decline"
  }
}
```

**取消：**

```json
{
  "hookSpecificOutput": {
    "hookEventName": "Elicitation",
    "action": "cancel"
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"Elicitation"` |
| `action` | string | ❌ | 操作：`accept`（接受）、`decline`（拒绝）、`cancel`（取消） |
| `content` | object | ❌ | 当 action 为 `accept` 时提供的数据，必须匹配 `requested_schema` |

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | 使用 hook 返回的 response（如果提供了 JSON） |
| 2 | **拒绝 elicitation** |
| 其他 | stderr 仅显示给用户 |

---

## 28 - ElicitationResult

**作用**：用户回应 MCP elicitation 后触发。可用于观察或覆盖用户响应。

**触发时机**：用户在 elicitation 对话框中选择接受/拒绝后。

**Matcher 字段**：`mcp_server_name`，值：MCP 服务器名称

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "ElicitationResult",
  "mcp_server_name": "my-mcp-server",
  "elicitation_id": "eli-abc123",
  "mode": "form",
  "action": "accept",
  "content": {
    "api_key": "sk-user-provided-key"
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `mcp_server_name` | string | ✅ | MCP 服务器名称 |
| `elicitation_id` | string | ❌ | elicitation 请求的唯一标识符 |
| `mode` | string | ❌ | 交互模式 |
| `action` | string | ✅ | 用户的响应：`accept`、`decline`、`cancel` |
| `content` | object | ❌ | 用户提交的数据（当 action 为 `accept` 时） |

### 输出 (stdout JSON)

**覆盖用户响应：**

```json
{
  "hookSpecificOutput": {
    "hookEventName": "ElicitationResult",
    "action": "accept",
    "content": {
      "api_key": "sk-sanitized-key"
    }
  }
}
```

**将接受改为拒绝：**

```json
{
  "hookSpecificOutput": {
    "hookEventName": "ElicitationResult",
    "action": "decline"
  }
}
```

| hookSpecificOutput 字段 | 类型 | 必填 | 说明 |
|------------------------|------|------|------|
| `hookEventName` | string | ✅ | 必须为 `"ElicitationResult"` |
| `action` | string | ❌ | 覆盖后的操作：`accept`、`decline`、`cancel` |
| `content` | object | ❌ | 覆盖后的数据内容 |

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | 使用 hook response（如果提供了 JSON），否则使用用户原始响应 |
| 2 | **阻止响应**（action 变为 `decline`） |
| 其他 | stderr 仅显示给用户 |

---

## 29 - SessionEnd

**作用**：会话结束时触发。用于清理资源、保存状态、发送最终通知。

**触发时机**：会话结束时（用户退出、清除、登出等）。

**Matcher 字段**：`reason`，值：`clear` | `resume` | `logout` | `prompt_input_exit` | `other` | `bypass_permissions_disabled`

### 输入 (stdin)

```json
{
  "session_id": "a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "transcript_path": "/home/user/.claude/projects/xxx/sessions/2024-01-01.jsonl",
  "cwd": "/home/user/my-project",
  "hook_event_name": "SessionEnd",
  "reason": "prompt_input_exit"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `reason` | string | ✅ | 会话结束原因：`clear`（清除对话）、`resume`（恢复其他会话）、`logout`（登出）、`prompt_input_exit`（用户在输入框退出）、`other`（其他原因）、`bypass_permissions_disabled`（bypass 权限被禁用） |

### 输出

**没有专属的 hookSpecificOutput 分支**。仅通用字段可用。

### 退出码

| 退出码 | 行为 |
|--------|------|
| 0 | 成功完成 |
| 其他 | stderr 仅显示给用户 |

> ⚠️ **超时限制**：SessionEnd hook 有严格的超时限制（默认 1500ms），可通过环境变量 `CLAUDE_CODE_SESSIONEND_HOOKS_TIMEOUT_MS` 调整。

---

## settings.json 配置示例

### 完整多 hook 配置

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup",
        "hooks": [
          {
            "type": "command",
            "command": "bash ~/.claude/hooks/on-startup.sh",
            "timeout": 30
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "python3 ~/.claude/hooks/inject_role.py"
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "python3 ~/.claude/hooks/audit_bash.py"
          }
        ]
      },
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "python3 ~/.claude/hooks/check_write.py"
          }
        ]
      }
    ],
    "PreCompact": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "bash ~/.claude/hooks/compact_instructions.sh"
          }
        ]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "python3 ~/.claude/hooks/verify_completion.py"
          }
        ]
      }
    ]
  }
}
```

### Hook 类型

```json
{
  "hooks": {
    "EventName": [
      {
        "matcher": "pattern",
        "hooks": [
          {
            "type": "command",
            "command": "bash /path/to/script.sh",
            "timeout": 30,
            "shell": "bash"
          },
          {
            "type": "command",
            "command": "powershell -File C:\\script.ps1",
            "shell": "powershell"
          }
        ]
      }
    ]
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 目前仅支持 `"command"` |
| `command` | string | 要执行的命令。支持变量替换：`${CLAUDE_PLUGIN_ROOT}`、`${CLAUDE_PLUGIN_DATA}` |
| `timeout` | number | 可选超时时间（秒），默认 600 秒（10 分钟） |
| `shell` | string | 可选 shell 类型：`"bash"`（默认）或 `"powershell"` |
| `async` | boolean | 可选异步模式，hook 在后台运行不阻塞 |
| `asyncRewake` | boolean | 可选异步重唤醒模式，hook 完成后会重新唤醒 model |
