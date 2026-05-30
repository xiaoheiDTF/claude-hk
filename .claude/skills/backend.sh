# .claude/skills/backend.sh
# 公共后端调用函数，各技能脚本 source 引入

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
  [ -n "${SKILL_TAG:-}" ] && skill_log "ERROR" "Backend configured but unreachable at $BACKEND_URL"
  return 1
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
  local rc=$?
  if [ $rc -ne 0 ] || [ -z "$resp" ]; then
    [ -n "${SKILL_TAG:-}" ] && skill_log "DEBUG" "Backend call failed: $endpoint (curl rc=$rc)"
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

# 通用状态更新函数（D2-D6 使用）— 静默降级
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

  # 1. 调后端更新状态
  local result
  result=$(_call_backend "/api/issue/status" "{
    \"repo_full_name\":\"$repo\",
    \"issue_number\":$issue_num,
    \"session_id\":\"$session_id\",
    \"status\":\"$new_status\"
  }")

  # 2. 从响应中获取 previous_status，同步 GitHub label
  local old_status
  old_status=$(echo "$result" | jq -r '.previous_status // empty' 2>/dev/null)
  [ -n "$old_status" ] && _sync_github_label "$issue_num" "$old_status" "$new_status"
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

  local old_label
  old_label=$(_status_to_label "$old_status")
  local new_label
  new_label=$(_status_to_label "$new_status")

  # 移除旧 label
  [ -n "$old_label" ] && [ "$old_label" != "$new_label" ] && \
    gh issue edit "$issue_num" --remove-label "$old_label" 2>/dev/null

  # 添加新 label
  [ -n "$new_label" ] && \
    gh issue edit "$issue_num" --add-label "$new_label" 2>/dev/null

  # merged 特殊处理：关闭 issue
  [ "$new_status" = "merged" ] && gh issue close "$issue_num" 2>/dev/null
}
