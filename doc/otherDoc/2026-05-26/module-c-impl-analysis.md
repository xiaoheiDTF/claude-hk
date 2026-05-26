# 模块 C 实现分析：本地代理改造 + 后端服务基础

> 创建时间：2026-05-26
> 模块：claude-tap-plus
> 简述：分析模块 C 如何利用 hooks SessionStart/SessionEnd 和 claude-tap-plus 代理实现会话注册、消息存储、会话注销

---

## 一、可获取的数据源

### 1.1 SessionStart Hook 提供的字段

从日志中提取的实际数据样本：

```json
{
  "session_id": "3cc83fcb-2849-4beb-9df0-6c638ab60566",
  "transcript_path": "C:\\Users\\Administrator\\.claude\\projects\\D--CodeDevelopment-CodeProject-claude-hk\\3cc83fcb-2849-4beb-9df0-6c638ab60566.jsonl",
  "cwd": "D:\\CodeDevelopment\\CodeProject\\claude-hk",
  "hook_event_name": "SessionStart",
  "source": "startup",
  "model": "GLM-5.1"
}
```

| 字段 | 说明 | 用途 |
|------|------|------|
| `session_id` | UUID，唯一标识一次会话 | 后端注册的主键 |
| `transcript_path` | 本地会话 JSONL 文件路径 | 消息溯源 |
| `cwd` | 当前项目根目录 | 项目识别 |
| `source` | 启动来源（startup/resume） | 区分首次启动和恢复 |
| `model` | 使用的模型名称 | 环境记录 |

### 1.2 SessionEnd Hook 提供的字段

```json
{
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "transcript_path": "C:\\Users\\Administrator\\.claude\\projects\\...",
  "cwd": "D:\\CodeDevelopment\\CodeProject\\claude-hk",
  "hook_event_name": "SessionEnd",
  "reason": "prompt_input_exit"
}
```

| 字段 | 说明 | 用途 |
|------|------|------|
| `session_id` | 同上 | 注销匹配 |
| `reason` | 退出原因（prompt_input_exit 等） | 判断正常退出还是异常中断 |

### 1.3 平台信息（hooks/platform.sh 已有）

| 数据 | 获取方式 | 示例值 |
|------|----------|--------|
| OS 类型 | `uname -s` | windows / linux / macos |
| 机器名 | `hostname` | DESKTOP-XXX |
| 机器 ID | 需新增生成逻辑 | 可用 hostname + username 组合或生成 UUID |

### 1.4 claude-tap-plus 代理已有的数据

代理拦截每次 API 调用，已记录：

| 数据 | 说明 |
|------|------|
| request body | 完整请求 JSON（含 messages、tools） |
| response body | 完整响应 JSON（含 content、usage） |
| session_id | 从请求元数据中提取 |
| timestamp | 每次调用时间 |
| duration_ms | 请求耗时 |
| token 统计 | input/output/cache tokens |
| model | 实际调用的模型 |

---

## 二、实现方案

### 2.1 整体架构

```
┌─────────────────────────────────────────────────┐
│                  Claude Code 进程                 │
│                                                   │
│  SessionStart hook ──→ POST /api/session/register │
│                          (注册会话)                │
│                                                   │
│  SessionEnd hook   ──→ POST /api/session/close    │
│                          (注销会话)                │
└──────────────────┬──────────────────────────────┘
                   │ API 请求
                   ▼
┌──────────────────────────────────────────────────┐
│            claude-tap-plus 本地代理                │
│                                                    │
│  拦截请求/响应 ──→ POST /api/message/store         │
│                     (转发消息到后端)                │
│                                                    │
│  代理退出 ──→ POST /api/session/heartbeat          │
│               (最后一次心跳)                        │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────┐
│              后端服务器 (待开发)                     │
│                                                    │
│  /api/session/register  → 注册会话                 │
│  /api/session/close     → 注销会话                 │
│  /api/session/heartbeat → 心跳保活                 │
│  /api/message/store     → 存储消息                 │
│  /api/issue/*           → Issue 管理接口           │
└────────────────────────────────────────────────────┘
```

### 2.2 改造点清单

#### C1: 启动时注册（SessionStart hook）

**改造位置**：`.claude/hooks/01-session-start/base.sh`

**做法**：在现有逻辑末尾新增一个函数，向后端发送注册请求。

```bash
register_session() {
  local session_id=$(json_get '.session_id')
  local cwd=$(json_get '.cwd')
  local model=$(json_get '.model')
  local source=$(json_get '.source')

  # 组装机器信息
  local machine_id=$(hostname)
  local os_type="$OS_TYPE"  # platform.sh 已解析

  # 后端地址从配置文件读取，不存在则跳过
  local backend_url
  backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)
  [ -z "$backend_url" ] && return 0

  # 发送注册请求
  curl -s -X POST "$backend_url/api/session/register" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$session_id\",\"machine_id\":\"$machine_id\",\"os\":\"$os_type\",\"cwd\":\"$cwd\",\"model\":\"$model\",\"source\":\"$source\"}" \
    > /dev/null 2>&1

  log "INFO" "Session registered to backend: $session_id"
}
```

**需要新增的配置文件**：`.claude/backend.conf`
```
BACKEND_URL=http://localhost:8080
```

**注册请求体**：

```json
{
  "session_id": "3cc83fcb-2849-4beb-9df0-6c638ab60566",
  "machine_id": "DESKTOP-XXX",
  "os": "windows",
  "cwd": "D:\\CodeDevelopment\\CodeProject\\claude-hk",
  "model": "GLM-5.1",
  "source": "startup"
}
```

#### C2: 消息拦截转发（claude-tap-plus 代理）

**改造位置**：`claude_tap_plus/internal/trace/` 或新增 `internal/upstream/`

**做法**：在 trace writer 记录 JSONL 的同时，将请求/响应异步发送到后端。

当前 trace writer 在每次 API 调用完成时写入一行 JSONL。需要在这个流程中增加一个 HTTP 上报步骤：

```
请求完成 → trace writer 写本地 JSONL → 同时 POST /api/message/store 到后端
```

**上报数据结构**：

```json
{
  "session_id": "3cc83fcb-...",
  "timestamp": "2026-05-26T22:20:24+08:00",
  "turn": 3,
  "duration_ms": 4500,
  "request": {
    "method": "POST",
    "path": "/v1/messages",
    "body": { ... }
  },
  "response": {
    "status": 200,
    "body": { ... }
  },
  "tokens": {
    "input": 1200,
    "output": 800,
    "cache_read": 500
  }
}
```

**关键点**：
- 后端地址从 `--tap-backend` 参数或配置文件读取
- 异步发送，不阻塞代理主流程
- 发送失败只记录日志，不影响代理正常工作

#### C3: 退出时注销（SessionEnd hook）

**改造位置**：`.claude/hooks/29-session-end/base.sh`

**做法**：在现有逻辑中新增向后端发送注销请求。

```bash
unregister_session() {
  local session_id=$(json_get '.session_id')
  local reason=$(json_get '.reason')

  local backend_url
  backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)
  [ -z "$backend_url" ] && return 0

  curl -s -X POST "$backend_url/api/session/close" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$session_id\",\"reason\":\"$reason\"}" \
    > /dev/null 2>&1

  log "INFO" "Session unregistered from backend: $session_id"
}
```

**注销请求体**：

```json
{
  "session_id": "3cc83fcb-2849-4beb-9df0-6c638ab60566",
  "reason": "prompt_input_exit"
}
```

---

## 三、后端 API 清单（模块 C 部分）

| 接口 | 方法 | 说明 | 调用方 |
|------|------|------|--------|
| `/api/session/register` | POST | 注册会话，记录机器/项目/模型信息 | SessionStart hook |
| `/api/session/close` | POST | 注销会话，记录退出原因 | SessionEnd hook |
| `/api/session/heartbeat` | POST | 心跳保活（可选，由代理定期发送） | claude-tap-plus |
| `/api/message/store` | POST | 存储一次 API 调用的请求/响应 | claude-tap-plus |
| `/api/message/query` | GET | 按会话/项目/时间查询历史消息 | 外部查询（可选） |

---

## 四、需要的依赖和前置条件

### 4.1 已有的（无需额外安装）

- `curl` — hooks 环境中已可用（Windows Git Bash 自带）
- `json_get()` — hooks/base.sh 已提供 JSON 解析能力
- `platform.sh` — 已有 OS 检测
- `jq` 或 Python JSON 解析 — 已有降级链

### 4.2 需要新增的

| 项目 | 说明 |
|------|------|
| `.claude/backend.conf` | 后端地址配置文件，不存在时静默跳过 |
| 机器 ID 生成逻辑 | 用 `hostname` + `whoami` 组合，或首次生成 UUID 存入 `.claude/.machine_id` |
| Go HTTP client | claude-tap-plus 内新增后端上报模块 |
| 后端服务 | 需要开发，接收注册/注销/消息存储请求 |

### 4.3 后端服务技术选型建议

后端本身不做限定，但需要满足：
- HTTP API 服务，能接收 JSON
- 持久化存储（SQLite 足够起步，后续可换 PostgreSQL）
- Issue 状态管理需要内存缓存 + 持久化

---

## 五、数据流总结

```
用户启动 claude-tap-plus claude
  │
  ├─ 代理启动，监听本地端口
  ├─ Claude Code 启动
  │    │
  │    ├─ SessionStart hook 触发
  │    │    └─ POST /api/session/register → 后端记录会话
  │    │
  │    ├─ 用户交互中...
  │    │    │
  │    │    ├─ 每次 API 请求 → 代理拦截
  │    │    │    ├─ 转发到上游 API
  │    │    │    ├─ 本地 JSONL 记录（已有）
  │    │    │    └─ POST /api/message/store → 后端存储消息
  │    │    │
  │    │    └─ Issue 技能调用 → 查询后端状态（模块 B）
  │    │
  │    └─ SessionEnd hook 触发
  │         └─ POST /api/session/close → 后端注销会话
  │
  └─ 代理退出，打印汇总
```
