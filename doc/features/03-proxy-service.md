# 代理服务功能与配置

> 最后更新：2026-06-06

---

## 功能概述

代理服务（`claude-tap` 默认子命令）是一个本地反向代理，拦截 Claude Code CLI 与上游 AI API 之间的 HTTP 流量，实现 API 调用录制（JSONL Trace）、实时转发、Model 改写等功能。

### 核心能力

| 能力 | 说明 |
|------|------|
| **流量拦截与转发** | 拦截 CLI → 上游 API 的请求，完整记录后转发 |
| **JSONL Trace 录制** | 每个 API 调用记录为独立 JSON 行，含请求/响应/Token 统计 |
| **SSE 流式处理** | 支持 SSE（Server-Sent Events）实时流转发，同时重组完整响应用于 Trace |
| **Model 改写** | 按配置强制替换请求中的 model 字段 |
| **多 Provider 支持** | Anthropic、OpenAI、Gemini、Kimi 等路径白名单 |
| **上游故障兜底** | 主上游不可用时自动切换到备用上游 |
| **Kimi Reasoning 缓存** | 为 Kimi/Moonshot 上游缓存 reasoning_content，自动回填 thinking 块 |
| **后端自动启动** | 代理启动时自动检测并拉起后端服务 |
| **空闲自动关闭** | 后端持续空闲 10 分钟自动关闭 |

---

## 子命令

```bash
# 代理模式（默认）— 启动反向代理 + 自动拉起后端
go run ./cmd/claude-tap claude

# 后端模式 — 独立启动后端 HTTP 服务
go run ./cmd/claude-tap backend [--port 8080] [--db backend.db] [--host 127.0.0.1]

# 会话同步
go run ./cmd/claude-tap session-push     # 收集 ~/.claude/ 会话到本地存储
go run ./cmd/claude-tap session-pull     # 从本地存储恢复会话
go run ./cmd/claude-tap session-status   # 查看同步状态
```

---

## 配置说明

### 代理配置（`internal/config/`）

#### ClientConfig — AI CLI 启动配置

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `Cmd` | `"claude"` | 要定位的 CLI 二进制文件名 |
| `BaseURLEnv` | `"ANTHROPIC_BASE_URL"` | 覆盖 base URL 的环境变量名 |
| `DefaultTarget` | `"https://api.anthropic.com"` | 无覆盖时的默认上游地址 |
| `NestingEnvKeys` | `CLAUDECODE, CLAUDE_CODE_SSE_PORT` | 启动前需清空的防嵌套环境变量 |
| `InjectSettingsEnv` | `true` | 是否通过 `--settings` 注入 base URL |

#### ProfileConfig — 多上游配置（`profiles.json`）

| 字段 | 说明 |
|------|------|
| `base_url` | 上游 API 地址 |
| `api_key` | API 密钥 |
| `auth_token` | OAuth Token |
| `provider` | 供应商标识（`anthropic` / `openai` / `gemini`） |
| `model` | 强制替换的模型名 |

#### 配置优先级链

```
命令行直接指定 > profiles.json > 环境变量 > ~/.claude.json > 默认值
```

#### ClaudeConfig / ClaudeSettings

从 `~/.claude.json` 和 `~/.claude/settings.json` 读取配置子集，用于：
- 获取 fallback 上游配置
- 读取代理相关设置

### 后端配置

| 参数 | 命令行 | 默认值 | 说明 |
|------|--------|--------|------|
| Host | `--host` | `127.0.0.1` | HTTP 监听地址 |
| Port | `--port / -p` | `8080` | HTTP 监听端口 |
| DBPath | `--db / -d` | `backend.db` | SQLite 数据库路径 |

### 配置桥接文件

`~/.claude-tap-plus/backend.json`：

```json
{"host": "127.0.0.1", "port": 8080}
```

- **写入时机**：后端服务启动时
- **删除时机**：后端服务退出时
- **读取方**：Shell 侧通过 `.claude/lib/config.sh` 的 `load_backend_config()` 读取

---

## 代理路由

| 路径 | 功能 |
|------|------|
| `/` | 通配路由 — 所有请求的默认处理，白名单检查后转发上游 |
| `/_internal/trace-init` | 内部端点 — 初始化 Trace 写入器（POST） |

### 路径白名单（`internal/proxy/paths.go`）

| 前缀 | Provider |
|------|----------|
| `/v1/messages` | Anthropic API |
| `/v1/complete` | Anthropic Legacy |
| `/v1/chat/completions` | OpenAI API |
| `/v1/models` | OpenAI Models |
| `/v1beta/` | Google Gemini API |
| `/v1/` | Kimi / 其他 |
| `/api/` | IDE 连接 |

---

## Trace 录制

### Trace 文件路径

```
~/.claude-tap-plus/.traces/{machineID}/{projectSlug}/{sessionID}.jsonl
```

- `machineID` 格式：`username@hostname`
- `projectSlug`：优先 git remote origin，兜底当前目录名

### Trace 记录内容

每行一个 `AnthropicTraceRecord` JSON，包含：

| 字段 | 说明 |
|------|------|
| `request` | 完整请求（URL、方法、头部、请求体） |
| `response` | 完整响应（状态码、头部、响应体） |
| `sse_events` | SSE 事件帧（流式请求时） |
| `model` | 实际使用的模型 |
| `tokens` | Token 统计（input/output/cache） |
| `duration_ms` | 请求耗时 |

### Token 统计（`domain/TokenStats`）

| 字段 | 说明 |
|------|------|
| `APICalls` | API 调用次数 |
| `InputTokens` | 输入 Token 数 |
| `OutputTokens` | 输出 Token 数 |
| `CacheRead` | 缓存读取 Token 数 |
| `CacheCreate` | 缓存创建 Token 数 |
| `Total()` | `InputTokens + OutputTokens` |

### 退出汇总

代理退出时打印汇总：

```
API calls: 15
Input tokens: 12,340
Output tokens: 8,560
Cache read: 5,200
Cache create: 1,800
Total tokens: 20,900
```

---

## 特殊机制

### 上游故障兜底（Fallback）

当主上游返回 4xx/5xx 或连接失败时：
1. `markUnavailable()` 标记主上游不可用
2. 从 `~/.claude/settings.json` 读取备用上游配置
3. 自动切换到备用上游

### Kimi Reasoning 缓存

为 Kimi/Moonshot 上游处理 `reasoning_content`：
- **缓存**：收集 assistant 消息中的 `reasoning_content`
- **回填**：三级查找策略：
  1. full key 精确匹配
  2. toolcall key 匹配
  3. 最近缓存值兜底
- 自动注入 `thinking` 块和 `reasoning_content` 字段

### 空闲 Watchdog（后端自动关闭）

| 参数 | 值 | 说明 |
|------|-----|------|
| 检查间隔 | 30 秒 | 每 30 秒检查 `proxy.json` 中活跃会话数 |
| 空闲超时 | 10 分钟 | 持续空闲 10 分钟后自动关闭后端进程 |

### 头部处理（`internal/proxy/headers.go`）

- 过滤逐跳头部（`Connection`、`Transfer-Encoding` 等）
- 敏感头部脱敏（`authorization`、`x-api-key`、`cosy-*` 系列等）
