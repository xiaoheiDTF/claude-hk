# Hooks 管线功能与配置

> 最后更新：2026-06-06

---

## 功能概述

Hooks 是 Claude Code CLI 的生命周期事件拦截层。当 Claude Code 触发事件时（如会话启动、用户提交 prompt、工具调用前后），CLI 通过 stdin JSON 将事件数据传递给注册的 Hook 脚本，脚本通过 exit code 和 stdout JSON 反馈控制指令。

### 核心能力

| 能力 | 说明 |
|------|------|
| **事件拦截** | 29 个生命周期事件全覆盖 |
| **Skill 分发** | `dispatch_to_skill()` 将事件路由到当前激活的 Skill |
| **工具边界管控** | `enforce_boundary.sh` 按 SKILL.md 白名单拦截未授权工具 |
| **JSON 解析** | `json_get()` 三级降级：jq → Python → sed |
| **平台适配** | 自动检测 OS、解析内嵌/系统 Python |
| **后端联动** | SessionStart 注册会话、SessionEnd 释放 Issue 锁 |

---

## 29 个 Hook 事件

### 会话生命周期（3 个）

| # | 事件 | 功能 |
|---|------|------|
| 01 | `SessionStart` | 首次初始化（UTF-8、Python、目录、Skill 巡检）；每次注册会话到后端、初始化 Trace 写入器 |
| 16 | `Stop` | 响应完成后：扫描注册新 Skill、执行 Skill 清理脚本、Windows 前台通知 |
| 29 | `SessionEnd` | 会话结束时：释放关联的 Issue 锁、从后端注销会话、分发到 Skill 清理 |

### 用户交互（3 个）

| # | 事件 | 功能 |
|---|------|------|
| 03 | `UserPromptSubmit` | **核心**：`skill-inject.sh` 检测 `/skill-name` → 匹配注册表 → 注入上下文 → 写入 `.active` 标记激活 |
| 04 | `UserPromptExpansion` | 命令展开前触发，exit 2 可阻止展开 |
| 27/28 | `Elicitation` / `ElicitationResult` | MCP 服务器请求用户输入时的拦截与响应 |

### 工具管控（4 个）

| # | 事件 | 功能 |
|---|------|------|
| 05 | `PreToolUse` | **双层拦截**：A层 `enforce_boundary.sh`（工具白名单）→ B层 Skill 路径级拦截；支持 allow/deny/ask/defer |
| 08 | `PostToolUse` | 工具调用成功后记录日志、分发到 Skill |
| 09 | `PostToolUseFailure` | 工具调用失败后记录 WARN 日志 |
| 10 | `PostToolBatch` | 并行工具批次完成后触发，exit 2 可阻止后续模型调用 |

### 权限与通知（3 个）

| # | 事件 | 功能 |
|---|------|------|
| 06 | `PermissionRequest` | 权限对话框出现时，Windows 下自动将终端拉到前台 |
| 07 | `PermissionDenied` | 自动模式拒绝工具调用时触发，可设置 `retry:true` 让模型重试 |
| 11 | `Notification` | Claude Code 发送通知时记录 |

### 任务与 Agent（4 个）

| # | 事件 | 功能 |
|---|------|------|
| 12/13 | `SubagentStart` / `SubagentStop` | Subagent 启停时记录 |
| 14/15 | `TaskCreated` / `TaskCompleted` | 任务创建/完成时触发，exit 2 可回滚/阻止 |
| 18 | `TeammateIdle` | 团队队友空闲时触发，exit 2 可阻止空闲让队友继续工作 |

### 其他事件（12 个）

| # | 事件 | 功能 |
|---|------|------|
| 02 | `Setup` | `--init-only` / `--maintenance` 模式触发 |
| 17 | `StopFailure` | API 错误终止时记录 ERROR |
| 19 | `InstructionsLoaded` | CLAUDE.md 加载时触发 |
| 20 | `ConfigChange` | 配置文件变更时触发，exit 2 可阻止生效 |
| 21 | `CwdChanged` | 工作目录变更 |
| 22 | `FileChanged` | 监视文件磁盘变更 |
| 23/24 | `WorktreeCreate` / `WorktreeRemove` | Worktree 创建/移除 |
| 25/26 | `PreCompact` / `PostCompact` | 上下文压缩前后，exit 2 可阻止压缩 |

---

## 配置说明

### Hook 注册（`.claude/settings.json`）

```json
{
  "hooks": {
    "SessionStart":  [{ "command": ".claude/hooks/01-session-start/base.sh", "matcher": "" }],
    "UserPromptSubmit": [{ "command": ".claude/hooks/03-user-prompt-submit/base.sh", "matcher": "" }],
    "PreToolUse": [{ "command": ".claude/hooks/05-pre-tool-use/base.sh", "matcher": "" }],
    "Stop": [{ "command": ".claude/hooks/16-stop/base.sh", "matcher": "" }],
    "SessionEnd": [{ "command": ".claude/hooks/29-session-end/base.sh", "matcher": "" }]
  }
}
```

> 所有 Hook 的 `matcher` 均为空字符串（全局匹配）。

### 目录结构

```
.claude/hooks/
├── base.sh                    # 共享基础设施（日志、JSON 解析、Skill 分发）
├── platform.sh                # 平台检测（OS_TYPE、PYTHON_CMD）
├── json_get.py                # JSON 解析 Python fallback
├── lib/
│   ├── backend.sh             # 后端 HTTP 调用模块
│   ├── config.sh              # backend.json 配置读取
│   ├── win32-foreground.sh    # Windows 前台拉起（调用 PowerShell）
│   └── win32-foreground.ps1   # Win32 API 窗口操作
├── 01-session-start/
│   ├── base.sh                # 调度器
│   ├── init.sh                # 首次初始化逻辑
│   ├── skill-inject.sh        # (03 的实际脚本，符号链接引用)
│   └── ...
├── 03-user-prompt-submit/
│   ├── base.sh
│   └── skill-inject.sh        # Skill 检测与上下文注入
├── 05-pre-tool-use/
│   ├── base.sh
│   ├── enforce_boundary.sh    # A层：工具白名单拦截
│   └── dispatch_to_skill.sh   # B层：Skill 级路径拦截
├── 16-stop/
│   ├── base.sh
│   ├── skill-register.sh      # 自动扫描注册新 Skill
│   └── task-complete-notify.sh
└── 29-session-end/
    ├── base.sh
    ├── release-session-issues.sh  # 释放 Issue 锁
    └── unregister-session.sh      # 注销会话
```

### Hook Exit Code 约定

| Exit Code | 含义 | 适用事件 |
|-----------|------|---------|
| 0 | 继续（默认行为） | 所有 |
| 2 | 阻止/拒绝 | PreToolUse（deny）、PostToolBatch（阻止后续）、PreCompact（阻止压缩）等 |
| stdout JSON | 控制指令 | PreToolUse 可返回 `{"decision":"allow\|deny\|ask"}` |

### 共享基础设施（`hooks/base.sh`）

| 函数 | 功能 |
|------|------|
| `json_get(key)` | 从 stdin JSON 提取字段，三级降级：jq → Python(json_get.py) → sed |
| `log(level, msg)` | 带事件名的格式化日志 |
| `hook_output(code, json)` | 统一退出：写日志 → 输出 JSON → 退出 |
| `dispatch_to_skill(event_num)` | 读取 `.active` 文件 → 路由到 Skill 的对应事件脚本 |

### 平台适配（`hooks/platform.sh`）

| 变量 | 来源 | 说明 |
|------|------|------|
| `OS_TYPE` | `uname -s` | `linux` / `macos` / `windows` / `unknown` |
| `PYTHON_CMD` | 三级解析 | ① 内嵌 `.claude/localLanguage/python/` ② 系统 `python3` ③ 系统 `python` |
