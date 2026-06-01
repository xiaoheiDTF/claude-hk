# B7: 公共后端调用模块与服务启动

> 创建时间：2026-05-27
> 模块：claude-tap-plus / Issue 管理
> 简述：定义共享 `backend.sh` 模块、`/health` 健康检查端点、后端服务 CLI 启动命令及配置文件格式

---

## 需求描述

各技能脚本（001-4 ~ 001-9）和 SessionEnd hook 都需要调用后端 API。将公共逻辑提取为共享模块，避免每个脚本重复实现 URL 读取、健康检查、HTTP 调用等逻辑。

## 1. 共享模块 `backend.sh`

**位置**：`.claude/skills/backend.sh`

各技能脚本在开头 `source` 此文件：

```bash
source "$CLAUDE_PROJECT_DIR/.claude/skills/backend.sh"
```

### 状态→GitHub Label 映射

```
后端状态         GitHub Label
─────────       ────────────
claimed    →    in-progress
fixing     →    fixing
ready-for-pr →  ready-for-pr
pr-created →    pr-created
testing    →    testing
reviewing  →    reviewing
merged     →    (close issue)
rejected   →    rejected
idle       →    (无 label)
```

此映射定义在 `backend.sh` 中，`update_issue_status()` 根据映射自动同步 GitHub label。

### 函数清单

```bash
# 读取后端 URL（带缓存，只读一次）
_load_backend_url() {
  if [ -z "$BACKEND_URL" ]; then
    BACKEND_URL=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null \
      | grep '^BACKEND_URL=' | cut -d= -f2)
  fi
}

# 检查后端是否可用（2 秒超时）
_backend_available() {
  _load_backend_url
  [ -n "$BACKEND_URL" ] && curl -s --max-time 2 "$BACKEND_URL/health" > /dev/null 2>&1
}

# 通用后端调用（5 秒超时，失败静默返回空）
_call_backend() {
  local endpoint="$1"
  local data="$2"
  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 1

  local resp
  resp=$(curl -s --max-time 5 -X POST "$BACKEND_URL$endpoint" \
    -H "Content-Type: application/json" \
    -d "$data" 2>/dev/null)
  [ -n "$resp" ] && echo "$resp"
}

# 获取当前 session_id
_get_session_id() {
  if [ -n "$CLAUDE_SESSION_ID" ]; then
    echo "$CLAUDE_SESSION_ID"
    return
  fi
  if command -v json_get >/dev/null 2>&1; then
    json_get '.session_id' 2>/dev/null
  fi
}

# 状态→GitHub Label 映射
_status_to_label() {
  case "$1" in
    claimed)       echo "in-progress" ;;
    fixing)        echo "fixing" ;;
    ready-for-pr)  echo "ready-for-pr" ;;
    pr-created)    echo "pr-created" ;;
    testing)       echo "testing" ;;
    reviewing)     echo "reviewing" ;;
    rejected)      echo "rejected" ;;
    merged|idle)   echo "" ;;
    *)             echo "" ;;
  esac
}

# 同步 GitHub label（后端状态变更后调用）
_sync_github_label() {
  local issue_num="$1"
  local old_status="$2"
  local new_status="$3"

  local old_label=$(_status_to_label "$old_status")
  local new_label=$(_status_to_label "$new_status")

  # 移除旧 label
  [ -n "$old_label" ] && [ "$old_label" != "$new_label" ] && \
    gh issue edit "$issue_num" --remove-label "$old_label" 2>/dev/null

  # 添加新 label
  [ -n "$new_label" ] && \
    gh issue edit "$issue_num" --add-label "$new_label" 2>/dev/null

  # merged 特殊处理：关闭 issue
  [ "$new_status" = "merged" ] && gh issue close "$issue_num" 2>/dev/null
}

# 更新后端状态 + 同步 GitHub label（双写）
update_issue_status() {
  local issue_number="$1"
  local new_status="$2"
  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 0

  local session_id
  session_id=$(_get_session_id)
  local repo
  repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner' 2>/dev/null)

  # 1. 调后端更新状态
  local result
  result=$(_call_backend "/api/issue/status" "{
    \"repo_full_name\":\"$repo\",
    \"issue_number\":$issue_number,
    \"session_id\":\"$session_id\",
    \"status\":\"$new_status\"
  }")

  # 2. 从响应中获取 previous_status，同步 GitHub label
  local old_status
  old_status=$(echo "$result" | jq -r '.previous_status // empty' 2>/dev/null)
  [ -n "$old_status" ] && _sync_github_label "$issue_number" "$old_status" "$new_status"
}
```

### 变量

| 变量 | 类型 | 说明 |
|------|------|------|
| `BACKEND_URL` | 全局缓存 | 首次调用 `_load_backend_url` 后填充，后续复用 |

### 依赖

- `curl` 命令可用
- `json_get` 函数或 `jq` 可用（用于 `_get_session_id`）
- `.claude/backend.conf` 文件（可选，不存在时所有后端调用静默跳过）

## 2. `/health` 健康检查端点

**端点**: `GET /health`

### 响应

```json
{"status": "ok", "uptime": 3600}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| status | string | 固定 `"ok"` |
| uptime | int | 服务运行秒数（可选） |

### 用途

`_backend_available()` 调用此端点判断后端是否可用。2 秒内无响应视为不可用。

### 验收标准

- [x] 后端运行时返回 `{"status": "ok"}`
- [x] 后端未启动时 curl 超时（连接失败）

## 3. 后端服务启动命令

后端服务作为 `claude-tap-plus` 的子命令运行，与代理进程独立：

```bash
# 启动后端（默认端口 8080）
claude-tap-plus backend

# 指定端口和数据库路径
claude-tap-plus backend --port 8080 --db ./backend.db

# 指定配置文件
claude-tap-plus backend --config ./backend.conf
```

### CLI 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--port` | `8080` | 监听端口 |
| `--db` | `./backend.db` | SQLite 数据库路径 |
| `--config` | 无 | 配置文件路径（可选） |

### 验收标准

- [x] `claude-tap-plus backend` 启动后监听指定端口
- [x] `--port` 和 `--db` 参数生效
- [x] 启动时自动创建数据库表（如不存在）

## 4. 配置文件格式

**位置**：`.claude/backend.conf`

```
BACKEND_URL=http://localhost:8080
```

| 字段 | 必填 | 说明 |
|------|------|------|
| BACKEND_URL | 是 | 后端服务地址，含协议和端口 |

### 读取方式

所有技能脚本和 hook 通过 `backend.sh` 的 `_load_backend_url()` 统一读取。文件不存在时 `BACKEND_URL` 为空，所有后端调用自动跳过。

## 5. `CLAUDE_SESSION_ID` 环境变量

`_get_session_id()` 优先读取环境变量 `CLAUDE_SESSION_ID`，fallback 到 `json_get '.session_id'`。

**当前决策**：不要求模块 C（代理侧）注入此环境变量，技能脚本统一使用 `json_get '.session_id'` 获取。如后续需要模块 C 注入，`backend.sh` 无需修改（已支持环境变量优先）。

## 验收标准

- [x] `backend.sh` 可被各技能脚本正确 source
- [x] 未配置 `backend.conf` 时，所有后端调用静默跳过
- [x] 后端不可用时，`_backend_available` 在 2 秒内返回失败
- [x] `_call_backend` 超时 5 秒后返回空，不阻塞技能主流程
- [x] `/health` 端点返回正确响应
