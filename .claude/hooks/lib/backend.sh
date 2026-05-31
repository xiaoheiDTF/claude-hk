# .claude/hooks/lib/backend.sh
# Hooks 统一后端调用模块
# 供 hooks 脚本 source 引入，提供后端 API 调用封装
# 与 skills/backend.sh 功能相同但独立维护，避免 hooks 依赖 skills 目录

BACKEND_URL=""

# 引入统一配置读取（~/.claude-tap-plus/backend.json）
source "$CLAUDE_PROJECT_DIR/.claude/lib/config.sh"

# 加载后端 URL（委托给 config.sh 的 load_backend_config）
_load_backend_url() {
  load_backend_config
}

# 检查后端是否可用
_backend_available() {
  _load_backend_url
  [ -n "$BACKEND_URL" ] && curl -s --max-time 2 "$BACKEND_URL/health" > /dev/null 2>&1
}

# 检查后端是否必须可用（用于 claim 等安全敏感操作）
# 返回码: 0=可用, 1=已配置但不可达, 2=未配置
_require_backend() {
  _load_backend_url
  if [ -z "$BACKEND_URL" ]; then
    return 2
  fi
  if curl -s --max-time 2 "$BACKEND_URL/health" > /dev/null 2>&1; then
    return 0
  fi
  return 1
}

# 调用后端 API，失败时静默返回空
# $1 = endpoint (如 /api/session/register)
# $2 = JSON data
_call_backend() {
  local endpoint="$1"
  local data="$2"
  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 1

  local resp
  resp=$(curl -s --max-time 5 -X POST "$BACKEND_URL$endpoint" \
    -H "Content-Type: application/json" \
    -d "$data" 2>/dev/null)
  local rc=$?
  if [ $rc -ne 0 ] || [ -z "$resp" ]; then
    return 1
  fi
  echo "$resp"
}

# 获取当前 session_id
_get_session_id() {
  if [ -n "${CLAUDE_SESSION_ID:-}" ]; then
    echo "$CLAUDE_SESSION_ID"
    return
  fi
  if command -v json_get >/dev/null 2>&1; then
    json_get '.session_id' 2>/dev/null
  fi
}
