#!/bin/bash
# 29-session-end: 会话终止时触发 (每个会话一次)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

session_id=$(json_get '.session_id')
reason=$(json_get '.reason')

log "INFO" "session=$session_id reason=$reason"

dispatch_to_skill "29" || true

release_session_issues() {
  local sid
  sid=$(json_get '.session_id')
  [ -z "$sid" ] && return 0

  local backend_url
  backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null \
    | grep '^BACKEND_URL=' | cut -d= -f2)
  [ -z "$backend_url" ] && return 0

  local result
  result=$(curl -s --max-time 5 -X POST "$backend_url/api/issue/release-session" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$sid\"}")

  local count
  count=$(echo "$result" | jq -r '.count // 0' 2>/dev/null)
  [ "$count" -gt 0 ] && log "INFO" "Released $count issues for session $sid"
}

release_session_issues

hook_output 0 '{}'
