# Hooks 使用方式

## 29 个生命周期事件

| # | 事件名 | 触发时机 | 当前逻辑 |
|---|--------|---------|---------|
| 01 | SessionStart | 每会话一次 | 初始化 + 环境巡检 |
| 02 | Setup | 设置变更时 | 纯日志 |
| 03 | UserPromptSubmit | 每轮用户输入 | Skill 检测 + 上下文注入 |
| 04 | UserPromptExpansion | prompt 展开后 | 纯日志 |
| 05 | PreToolUse | 工具调用前 | 工具边界检查 |
| 06 | PermissionRequest | 权限请求时 | 纯日志 |
| 07 | PermissionDenied | 权限被拒时 | 纯日志 |
| 08 | PostToolUse | 工具调用后 | 纯日志 |
| 09 | PostToolUseFailure | 工具调用失败后 | 纯日志 |
| 10 | PostToolBatch | 批量工具调用后 | 纯日志 |
| 11 | Notification | 通知事件 | 纯日志 |
| 12 | SubagentStart | 子 agent 启动时 | 纯日志 |
| 13 | SubagentStop | 子 agent 停止时 | 纯日志 |
| 14 | TaskCreated | 任务创建时 | 纯日志 |
| 15 | TaskCompleted | 任务完成时 | 纯日志 |
| 16 | Stop | 每轮响应完成 | Skill 清理 + 自动注册 |
| 17 | StopFailure | 响应失败时 | 纯日志 |
| 18 | TeammateIdle | 队友空闲时 | 纯日志 |
| 19 | InstructionsLoaded | 指令加载后 | 纯日志 |
| 20 | ConfigChange | 配置变更时 | 纯日志 |
| 21 | CwdChanged | 工作目录变更时 | 纯日志 |
| 22 | FileChanged | 文件变更时 | 纯日志 |
| 23 | WorktreeCreate | 创建工作树时 | 纯日志 |
| 24 | WorktreeRemove | 删除工作树时 | 纯日志 |
| 25 | PreCompact | 压缩前 | 纯日志 |
| 26 | PostCompact | 压缩后 | 纯日志 |
| 27 | Elicitation | 引导请求时 | 纯日志 |
| 28 | ElicitationResult | 引导结果时 | 纯日志 |
| 29 | SessionEnd | 会话结束时 | 纯日志 |

## 如何给事件添加自定义脚本

### 目录结构

```
.claude/hooks/
└── XX-event-name/
    ├── base.sh           # 入口（调度器）
    └── my-script.sh      # 你的自定义脚本
```

### 步骤

1. 在目标事件目录下新建 `.sh` 脚本
2. 在 `base.sh` 中添加调用

示例：给 `05-pre-tool-use` 添加自定义检查：

```bash
# .claude/hooks/05-pre-tool-use/my-check.sh
#!/bin/bash
source "$CLAUDE_PROJECT_DIR/.claude/hooks/base.sh"
# 你的逻辑...
hook_output 0 ""
```

```bash
# .claude/hooks/05-pre-tool-use/base.sh 中添加
source "$(dirname "$0")/my-check.sh"
```

### 注意事项

- 不需要修改 `settings.json`
- 脚本通过 stdin 接收 JSON 数据，使用 `json_get()` 解析
- `hook_output(0, "")` 继续，`hook_output(2, json)` 阻止/修改
