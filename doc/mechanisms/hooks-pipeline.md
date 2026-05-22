# Hooks 管道

## 事件注册

`settings.json` 中注册 29 个生命周期事件，每个事件指向对应的 `base.sh`：

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [{
          "type": "command",
          "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/01-session-start/base.sh\""
        }]
      }
    ]
  }
}
```

所有 matcher 为空字符串，意味着每个事件无条件触发。

## 调度器模式

每个事件目录遵循**两层架构**：

```
XX-event-name/
├── base.sh            # 入口（调度器）
└── other-script.sh    # 业务脚本（可选）
```

`base.sh` 的职责：
1. Source 公共的 `hooks/base.sh`
2. 读取 stdin JSON 数据
3. 调用同目录下的业务脚本
4. 通过 `hook_output()` 输出结果

**有业务逻辑的事件**：

| 事件 | base.sh 调度的脚本 |
|------|-------------------|
| 01-session-start | init.sh（首次）、各 skill 的 init_check.sh |
| 03-user-prompt-submit | skill-inject.sh（Skill 匹配与上下文注入） |
| 05-pre-tool-use | enforce_boundary.sh（工具白名单）+ dispatch_to_skill |
| 16-stop | skill-register.sh（自动注册）+ dispatch_to_skill |

**扩展方式**：在事件目录下新建 `.sh` 脚本，在 `base.sh` 中添加调用。不需要改 `settings.json`。

## 公共基础设施 (hooks/base.sh)

### JSON 解析：三级降级

`json_get(key)` 函数按优先级尝试三种解析方式：

1. **jq** — 最快最可靠
2. **Python + json_get.py** — Python 可用时使用
3. **sed** — 最后手段，仅支持顶层 `"key": "value"` 模式

### 结构化日志

```bash
log "INFO" "SessionStart" "message"
# 写入 .claude/hooks/logs/2026-05-22.log
# 格式: [2026-05-22 10:00:00] [INFO] [SessionStart] message
```

### 标准输出

```bash
hook_output 0 ""        # 继续
hook_output 2 '{"deny":true}'  # 阻止操作（如 PreToolUse 拒绝工具调用）
```

## 平台检测 (hooks/platform.sh)

通过 `uname -s` 检测操作系统：
- `Linux` → linux
- `Darwin` → macos
- `MINGW`/`MSYS`/`CYGWIN` → windows

Python 路径解析优先级：
1. 嵌入版 `.claude/localLanguage/python/python.exe`（Windows）
2. 系统 `python3`
3. 系统 `python`
