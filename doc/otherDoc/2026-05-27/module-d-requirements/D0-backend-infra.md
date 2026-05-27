# D0: 公共后端调用基础设施

> 创建时间：2026-05-27
> 模块：claude-tap-plus / 模块 D / D0
> 简述：创建 `backend.sh` 共享模块，提供后端 API 调用函数，供 D1-D6 子需求依赖（D7 除外）

---

## 目标

新增 `.claude/skills/backend.sh`，所有 Issue 技能脚本通过 `source` 引入，统一后端调用逻辑。

## 产出文件

| 文件 | 操作 |
|------|------|
| `.claude/skills/backend.sh` | **新增** |
| `.claude/backend.conf` | **新增**（配置文件，部署时创建） |

## backend.sh 完整实现

```bash
# .claude/skills/backend.sh
# 公共后端调用函数，各技能脚本 source 引入

BACKEND_URL=""

_load_backend_url() {
  if [ -z "$BACKEND_URL" ]; then
    BACKEND_URL=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)
  fi
}

# 检查后端是否可用
_backend_available() {
  _load_backend_url
  [ -n "$BACKEND_URL" ] && curl -s --max-time 2 "$BACKEND_URL/health" > /dev/null 2>&1
}

# 调用后端 API，失败时静默返回空
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

# 通用状态更新函数（D2-D6 使用）
update_issue_status() {
  local issue_num="$1"
  local new_status="$2"

  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 0

  local session_id
  session_id=$(_get_session_id)
  [ -z "$session_id" ] && return 0

  local repo
  repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner' 2>/dev/null)
  [ -z "$repo" ] && return 0

  _call_backend "/api/issue/status" "{
    \"repo_full_name\":\"$repo\",
    \"issue_number\":$issue_num,
    \"session_id\":\"$session_id\",
    \"status\":\"$new_status\"
  }" > /dev/null 2>&1
}
```

## backend.conf 格式

```
BACKEND_URL=http://localhost:8080
```

## 使用方式

D1-D6 通过 `source backend.sh` 调用后端；D7（SessionEnd hook）**除外**——hook 运行在独立环境中，不 source backend.sh，直接内联 curl 调用。

在需要后端集成的技能脚本头部添加：

```bash
source "$CLAUDE_PROJECT_DIR/.claude/skills/backend.sh"
```

## 依赖的后端 API（由模块 B 提供）

| API | 方法 | 用途 |
|-----|------|------|
| `/health` | GET | 健康检查 |
| `/api/issue/check` | POST | 批量查询 issue 状态 |
| `/api/issue/claim` | POST | 原子领取 issue |
| `/api/issue/status` | POST | 更新 issue 状态 |
| `/api/issue/release` | POST | 释放 issue |
| `/api/issue/release-session` | POST | 释放 session 下所有 issue |

## 验证标准

- [ ] `backend.sh` 文件创建，所有函数可被 source 调用
- [ ] 未配置 `backend.conf` 时，所有函数静默返回（不报错）
- [ ] 后端不可用时，`_call_backend` 返回空字符串
- [ ] `update_issue_status` 可正确构造 JSON 请求体

## CLAUDE_SESSION_ID 来源说明

`_get_session_id()` 有两种获取方式（优先级从高到低）：

1. **环境变量 `CLAUDE_SESSION_ID`**：由 claude-tap-plus 代理在启动子进程时注入（模块 C 改造）
2. **`json_get '.session_id'`**：hook 脚本从 stdin JSON 中直接提取，无需代理改造

> 当前推荐使用方式 2，因为不需要改动代理代码。如果后续模块 C 实现了环境变量注入，方式 1 会自动生效。
