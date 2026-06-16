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

# 注销 proxy.json 中的会话条目（对称于 01-session-start 的 trace-init 注册）。
# 钩子是 Claude 生命周期事件，正常退出时必触发；比 proxy 进程 defer 可靠（强杀时 defer 不执行会残留）。
unregister_proxy_session() {
  local proxy_url
  proxy_url="${CLAUDE_TAP_PROXY_URL:-}"
  [ -z "$proxy_url" ] && return 0

  curl -s --max-time 5 -X POST "$proxy_url/_internal/session-close" \
    -H "Content-Type: application/json" \
    -d '{}' > /dev/null 2>&1 \
    && log "INFO" "Proxy session unregistered from proxy.json via $proxy_url" \
    || log "DEBUG" "Proxy session-close unreachable at $proxy_url (proxy may have exited)"
}

unregister_proxy_session

hook_output 0 '{}'
