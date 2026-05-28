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
