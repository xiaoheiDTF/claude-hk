#!/bin/bash
# 29-session-end: 会话终止时触发 (每个会话一次)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"
source "$HOOKS_DIR/lib/backend.sh"

session_id=$(json_get '.session_id')
reason=$(json_get '.reason')

log "INFO" "session=$session_id reason=$reason"

dispatch_to_skill "29" || true

release_session_issues() {
  local sid
  sid=$(json_get '.session_id')
  [ -z "$sid" ] && return 0

  if ! _backend_available; then
    log "INFO" "Backend unreachable, skipping session release for $sid"
    return 0
  fi

  local result
  result=$(_call_backend "/api/issue/release-session" "{\"session_id\":\"$sid\"}") || return 0

  local count
  count=$(echo "$result" | jq -r '.count // 0' 2>/dev/null)
  [ "$count" -gt 0 ] && log "INFO" "Released $count issues for session $sid"
}

release_session_issues

# SR-3: 注销会话
unregister_session() {
  local sid reason
  sid=$(json_get '.session_id')
  reason=$(json_get '.reason')
  [ -z "$sid" ] && return 0

  if ! _backend_available; then
    log "DEBUG" "Backend unreachable, skipping session close for $sid"
    return 0
  fi

  _call_backend "/api/session/close" "{\"session_id\":\"$sid\",\"reason\":\"$reason\"}" > /dev/null 2>&1
  log "INFO" "Session unregistered from backend: $sid"
}

unregister_session

hook_output 0 '{}'
