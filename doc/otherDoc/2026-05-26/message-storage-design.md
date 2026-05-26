# 消息存储设计：项目标识与 trace 关联

> 创建时间：2026-05-26
> 模块：claude-tap-plus
> 简述：复用现有 trace JSONL 存储，重点解决项目唯一标识和 trace 文件关联

---

## 一、现有 trace 存储不变

代理已经在 `buildRecord()` 后写入 JSONL trace。

当前路径规则：
```
.traces/{project}/{date}_{time}_{shortID}.jsonl
```

**需要改为**：跟 Claude Code 的 transcript_path 一致，一个 session 一个文件：
```
.claude-tap-plus/.traces/{machine_id}/{project_slug}/{session_id}.jsonl
```

示例：
```
.claude-tap-plus/.traces/Administrator@DESKTOP-ABC123/D--CodeDevelopment-CodeProject-claude-hk/bf15cac4-7235-48ce-8853-5d4598547f31.jsonl
```

四层目录：
1. `.claude-tap-plus/` — 工具根目录
2. `.traces/` — trace 存储目录
3. `{machine_id}/` — 机器标识（`whoami@hostname`）
4. `{project_slug}/` — 项目标识（从 transcript_path 提取）

改造点：`trace/writer.go` 的 `NewTracePath()` 中加入 machine_id 层级，session_id 从代理拦截的请求里 `extractSessionID()` 获取，首次拦截到时创建文件，后续同 session 追加写入。

---

## 二、核心问题：怎么识别"哪个项目、哪台机器"

### 2.1 SessionStart 给了什么

```json
{
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "transcript_path": "C:\\Users\\Administrator\\.claude\\projects\\D--CodeDevelopment-CodeProject-claude-hk\\bf15cac4-7235-48ce-8853-5d4598547f31.jsonl",
  "cwd": "D:\\CodeDevelopment\\CodeProject\\claude-hk",
  "hook_event_name": "SessionStart",
  "source": "startup",
  "model": "GLM-5.1"
}
```

### 2.2 transcript_path 已经包含所有标识

拆解这条路径：

```
C:\Users\Administrator\.claude\projects\D--CodeDevelopment-CodeProject-claude-hk\bf15cac4-7235-48ce-8853-5d4598547f31.jsonl
│                                          │                                              │
│                                          │                                              └─ session_id（已知）
│                                          │
│                                          └─ project_slug（Claude Code 内置的项目标识）
│                                             由 cwd 的路径分隔符替换为 - 得到
│                                             D:\CodeDevelopment\CodeProject\claude-hk
│                                             → D--CodeDevelopment-CodeProject-claude-hk
│
└─ 用户主目录 → 机器/用户标识
   Windows: C:\Users\Administrator
   Linux:   /home/username
   macOS:   /Users/username
```

**一条 transcript_path 就能拆出三个关键信息**：

| 信息 | 从哪取 | 值 |
|------|--------|-----|
| 用户/机器 | `.claude/` 前的路径 | `C:\Users\Administrator` |
| 项目标识 | `projects/` 后、session_id 前 | `D--CodeDevelopment-CodeProject-claude-hk` |
| session_id | 文件名去掉 `.jsonl` | `bf15cac4-7235-48ce-8853-5d4598547f31` |

### 2.3 解析方式

```bash
transcript_path=$(json_get '.transcript_path')

# 提取 project_slug
# 路径格式: {home}/.claude/projects/{project_slug}/{session_id}.jsonl
project_slug=$(echo "$transcript_path" | sed -E 's|.*/projects/([^/]+)/.*|\1|')
# → D--CodeDevelopment-CodeProject-claude-hk

# 提取用户主目录（机器/用户标识）
home_projects=$(echo "$transcript_path" | sed -E 's|(.*/.claude)/projects/.*|\1|')
# → C:\Users\Administrator\.claude
# 去掉 .claude 就是用户目录
user_home=$(echo "$home_projects" | sed 's|/\.claude$||')
# → C:\Users\Administrator
```

### 2.4 最终标识组合

```
project_id = "{hostname}/{project_slug}"
```

| 示例 | 含义 |
|------|------|
| `DESKTOP-ABC123/D--CodeDevelopment-CodeProject-claude-hk` | DESKTOP-ABC123 上的 claude-hk 项目 |
| `server-01/home-user-projects-my-app` | server-01 上的 my-app 项目 |

**hostname 加上去的原因**：project_slug 只在同一台机器内唯一，跨机器可能重复。

---

## 三、后端只存一个关联表

trace 文件在本地，后端不存消息内容。后端只需要知道：

> 哪个 session → 哪台机器 → 哪个项目 → trace 文件在哪

### 3.1 数据结构

```json
{
  "bf15cac4-7235-48ce-8853-5d4598547f31": {
    "machine_id": "Administrator@DESKTOP-ABC123",
    "os": "windows",
    "project_slug": "D--CodeDevelopment-CodeProject-claude-hk",
    "project_cwd": "D:\\CodeDevelopment\\CodeProject\\claude-hk",
    "trace_path": ".claude-tap-plus/.traces/Administrator@DESKTOP-ABC123/D--CodeDevelopment-CodeProject-claude-hk/bf15cac4-7235-48ce-8853-5d4598547f31.jsonl",
    "model": "GLM-5.1",
    "registered_at": "2026-05-26T21:06:16Z",
    "closed_at": null,
    "close_reason": null
  }
}
```

### 3.2 trace_path 怎么来

trace 文件路径现在跟 Claude Code 一致，格式为 `.claude-tap-plus/.traces/{machine_id}/{project_slug}/{session_id}.jsonl`。

代理拦截到第一个请求时，从 `extractSessionID()` 拿到 session_id，创建对应的 trace 文件。后续同 session 的请求追加写入同一个文件。

SessionStart hook 中也能拼出这个路径：

```bash
# 从 transcript_path 提取 project_slug
project_slug=$(echo "$transcript_path" | sed -E 's|.*/projects/([^/]+)/.*|\1|')
session_id=$(json_get '.session_id')
machine_id="$(whoami)@$(hostname)"

# 拼出 trace 路径
trace_path=".claude-tap-plus/.traces/${machine_id}/${project_slug}/${session_id}.jsonl"
```

或者代理通过环境变量传入：

```bash
TRACE_PATH="${CLAUDE_TAP_TRACE_PATH:-}"
```

### 3.3 写入时机

| 时机 | 动作 | 写什么 |
|------|------|--------|
| SessionStart | 注册 | session_id、machine_id、os、project_slug、project_cwd、trace_path、model |
| SessionEnd | 注销 | 关闭时间、退出原因 |

---

## 四、SessionStart hook 改造

在 `01-session-start/base.sh` 末尾加一段：

```bash
# ---- 会话注册到后端 ----
register_to_backend() {
  local session_id=$(json_get '.session_id')
  local transcript_path=$(json_get '.transcript_path')
  local cwd=$(json_get '.cwd')
  local model=$(json_get '.model')

  local machine_id="$(whoami)@$(hostname)"
  local trace_path="${CLAUDE_TAP_TRACE_PATH:-}"

  # 从 transcript_path 提取 project_slug
  local project_slug=""
  project_slug=$(echo "$transcript_path" | sed -E 's|.*/projects/([^/]+)/.*|\1|')

  # 后端地址，不存在则跳过
  local backend_url
  backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null \
    | grep '^BACKEND_URL=' | cut -d= -f2)
  [ -z "$backend_url" ] && return 0

  curl -s -X POST "$backend_url/api/session/register" \
    -H "Content-Type: application/json" \
    -d "{
      \"session_id\":\"$session_id\",
      \"machine_id\":\"$machine_id\",
      \"os\":\"$OS_TYPE\",
      \"project_slug\":\"$project_slug\",
      \"project_cwd\":\"$cwd\",
      \"trace_path\":\"$trace_path\",
      \"model\":\"$model\"
    }" > /dev/null 2>&1

  log "INFO" "Backend registered: project=$project_slug machine=$machine_id"
}

register_to_backend
```

---

## 五、查询示例

拿到一个 session_id，查后端：

```
session_id: bf15cac4-7235-48ce-8853-5d4598547f31
  → machine:  Administrator@DESKTOP-ABC123 (windows)
  → project:  D--CodeDevelopment-CodeProject-claude-hk
  → cwd:      D:\CodeDevelopment\CodeProject\claude-hk
  → trace:    .claude-tap-plus/.traces/Administrator@DESKTOP-ABC123/D--CodeDevelopment-CodeProject-claude-hk/bf15cac4-7235-48ce-8853-5d4598547f31.jsonl
```

按项目查所有会话：

```
project_slug: D--CodeDevelopment-CodeProject-claude-hk
  → session bf15cac4... (已结束, 15次调用)
  → session 3cc83fcb... (活跃, 28次调用)
  → session 6b6f69d3... (活跃, 3次调用)
```

需要看消息内容时，去对应机器的 trace JSONL 文件里找。
