# claude-tap-plus 代理模式 CLI 增强功能说明

> 创建时间：2026-05-30
> 所属模块：claude-tap-plus
> 简述：代理模式 CLI 新增功能：控制台日志控制、多配置 profiles、退出 resume 命令

---

## 一、控制台日志控制

### 问题

代理模式下，日志通过 stderr 输出，而 Claude Code 子进程共享同一 stderr，导致代理日志泄露到 Claude Code 的聊天界面中。

### 解决方案

| 标志 | 日志级别 | 写文件 | 输出到终端 | 说明 |
|------|---------|--------|-----------|------|
| （无标志） | INFO | ✅ | ❌ | 默认，只写文件，不污染终端 |
| `--tap-verbose` | DEBUG | ✅ | ❌ | 详细日志，仍然只写文件 |
| `--tap-console` | — | — | — | 单独无效，需配合 verbose |
| `--tap-verbose --tap-console` | DEBUG | ✅ | ✅ | 调试模式，文件+终端双写 |
| 后端模式（无标志） | DEBUG | ✅ | ✅ | 后端默认输出到终端 |

### 使用示例

```bash
# 正常使用：日志只写文件，终端干净
claude-tap-plus claude

# 详细日志写文件，终端仍然干净
claude-tap-plus --tap-verbose claude

# 调试模式：详细日志同时输出到终端（开发调试用）
claude-tap-plus --tap-verbose --tap-console claude
```

### 日志文件位置

```
~/.claude-tap-plus/.traces/2026-05-30.log
```

### 日志格式

```
15:04:05.123 [INFO ] proxy: listening on 127.0.0.1:50608
15:04:06.001 [DEBUG] proxy: [Turn 1] request: POST /v1/messages (15234 bytes)
15:04:08.890 [INFO ] proxy: [Turn 1] ← 200 stream done (2856ms, model=claude-sonnet-4-20250514)
```

---

## 二、多配置文件 profiles.json

### 配置文件位置

```
~/.claude-tap-plus/profiles.json
```

### 配置文件格式

```json
{
  "default": "anthropic-main",
  "profiles": {
    "anthropic-main": {
      "base_url": "https://api.anthropic.com",
      "api_key": "sk-ant-xxxxx",
      "provider": "anthropic"
    },
    "anthropic-proxy": {
      "base_url": "https://my-proxy.com",
      "api_key": "sk-ant-yyyyy",
      "provider": "anthropic"
    },
    "openai": {
      "base_url": "https://api.openai.com/v1",
      "api_key": "sk-zzzzz",
      "provider": "openai"
    },
    "gemini": {
      "base_url": "https://generativelanguage.googleapis.com/v1beta",
      "api_key": "AIza-xxxxx",
      "provider": "gemini"
    }
  }
}
```

### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `default` | string | 是 | 默认配置名，启动时未指定 `--tap-profile` 时使用 |
| `profiles` | object | 是 | 配置集合，key 为配置名 |
| `profiles[name].base_url` | string | 是 | 上游 API 地址 |
| `profiles[name].api_key` | string | 是 | API Key |
| `profiles[name].provider` | string | 否 | 供应商标识（anthropic/openai/gemini），用于日志记录 |

### 新增 CLI 参数

| 参数 | 说明 |
|------|------|
| `--tap-profile NAME` | 选择 profiles.json 中的配置 |
| `--tap-api-key KEY` | 直接传入 API Key（优先级最高） |
| `--tap-base-url URL` | 直接传入上游地址（等同 `--tap-target`，优先级最高） |

### 优先级（从高到低）

```
--tap-api-key / --tap-base-url（命令行直接指定）
        ↓ 未指定则
--tap-profile（读取 profiles.json）
        ↓ 未指定则
ANTHROPIC_API_KEY / ANTHROPIC_BASE_URL（环境变量）
        ↓ 未设置则
~/.claude.json（Claude Code 配置文件）
        ↓ 未配置则
默认值（无 API Key，base_url = https://api.anthropic.com）
```

### 使用示例

```bash
# 使用 default 配置
claude-tap-plus claude

# 使用指定配置
claude-tap-plus --tap-profile openai claude

# 命令行直接传入 API Key
claude-tap-plus --tap-api-key sk-ant-xxxxx claude

# 组合使用：profile + 覆盖 base URL
claude-tap-plus --tap-profile anthropic-main --tap-base-url https://proxy.com claude

# 完整调试模式
claude-tap-plus --tap-verbose --tap-console --tap-profile openai claude
```

---

## 三、退出摘要与 Resume 命令

### 退出时的输出

Claude Code 退出后，代理会打印完整摘要：

```
📋 Claude Code exited
   API calls:      4
   Input tokens:    1234
   Output tokens:   5678
   Cache read:      100
   Cache create:    200
   Trace: ~/.claude-tap-plus/.traces/claude-hk/2026-05-30_150924_f9a252.jsonl

📎 Resume:
   claude-tap-plus --tap-verbose claude --resume fbf7e71a-f540-4906-973b-317649cb6431
```

### Resume 命令逻辑

Resume 命令会**镜像用户原始输入**，追加 `--resume <session_id>`：

| 用户输入 | 退出时输出 |
|---------|-----------|
| `claude-tap-plus claude` | `claude-tap-plus claude --resume <id>` |
| `claude-tap-plus --tap-verbose claude` | `claude-tap-plus --tap-verbose claude --resume <id>` |
| `claude-tap-plus --tap-verbose claude --resume old-id` | `claude-tap-plus --tap-verbose claude --resume <new-id>` |
| `claude-tap-plus --tap-profile test claude` | `claude-tap-plus --tap-profile test claude --resume <id>` |

### 实现方式

1. 启动时记录 `os.Args`（原始命令行参数）
2. 去掉已有的 `--resume <old-id>` 如果存在
3. 退出时拼接 `--resume <new_session_id>` 输出

---

## 四、Claude Code `/clear` 命令与代理的关系

### `/clear` 的作用

Claude Code 的 `/clear` 命令：
- **清除当前对话上下文**（释放 context window）
- **不改变 session_id**（仍在同一会话中）
- **不清除 transcript 文件**（完整对话历史保留）
- 效果等同于 `/reset` 和 `/new`

### 对代理的影响

| 场景 | session_id | trace 文件 | resume 命令 |
|------|-----------|-----------|------------|
| 正常使用 | 不变 | 同一个 .jsonl | 同一个 --resume id |
| 输入 `/clear` | **不变** | **同一个 .jsonl**（继续追加） | 同一个 --resume id |
| 输入 `/exit` 后 resume | 新 session | 新 .jsonl | 新 --resume id |

### Resume 后的行为

```
# 第一次启动
claude-tap-plus --tap-verbose claude
→ 退出时输出: claude-tap-plus --tap-verbose claude --resume abc-123

# 用 resume 恢复
claude-tap-plus --tap-verbose claude --resume abc-123
→ Claude Code 恢复之前的对话上下文
→ session_id 不变（仍然是 abc-123）
→ trace 文件继续追加

# 如果用户在恢复的会话中输入 /clear
→ session_id 不变（仍然是 abc-123）
→ trace 文件继续追加（记录 /clear 后的新请求）
→ 退出时 resume 命令仍然是 --resume abc-123
```

### 注意事项

- `/clear` 后 transcript 文件会保留完整历史（包括 clear 前的对话）
- 代理会继续记录 `/clear` 后的所有 API 请求到同一个 trace 文件
- 如果需要完全干净的会话，应该退出后重新启动（不用 `--resume`）

---

## 五、完整使用流程示例

```bash
# 1. 启动后端服务（终端 1）
claude-tap-plus backend --port 8080

# 2. 启动代理 + Claude Code（终端 2）
claude-tap-plus --tap-verbose --tap-profile anthropic-main claude

# 3. 正常使用 Claude Code...
#    所有 API 请求被代理拦截并记录

# 4. 退出 Claude Code（输入 exit 或 Ctrl+C）
#    终端输出：
#    📋 Claude Code exited
#       API calls:      12
#       Input tokens:    50000
#       Output tokens:   8000
#       Trace: ~/.claude-tap-plus/.traces/claude-hk/2026-05-30_150924_f9a252.jsonl
#
#    📎 Resume:
#       claude-tap-plus --tap-verbose --tap-profile anthropic-main claude --resume fbf7e71a-...

# 5. 恢复会话（复制粘贴 resume 命令）
claude-tap-plus --tap-verbose --tap-profile anthropic-main claude --resume fbf7e71a-...

# 6. 查看日志文件
cat ~/.claude-tap-plus/.traces/2026-05-30.log

# 7. 查看 trace 数据
cat ~/.claude-tap-plus/.traces/claude-hk/2026-05-30_150924_f9a252.jsonl | jq .
```
