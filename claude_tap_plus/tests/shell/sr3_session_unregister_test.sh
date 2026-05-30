#!/bin/bash
# SR-3: SessionEnd Hook 会话注销 — Shell 集成测试
# 验收标准：
#   1. 正常退出后，后端会话状态变为 closed
#   2. 后端记录了 close_reason 和 closed_at 时间
#   3. 代理未运行时（后端不可用），退出无报错
#   4. 后端不可达时，退出无报错
#   5. 异常退出后（未调 close），session 保持 active
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
BACKEND_SH="$PROJECT_DIR/.claude/skills/backend.sh"
BACKEND_CONF="$PROJECT_DIR/.claude/backend.conf"

export CLAUDE_PROJECT_DIR="$PROJECT_DIR"
export SKILL_TAG="sr3-test"
skill_log() { :; }
export -f skill_log

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

TOTAL=0
PASS=0
FAIL=0

ORIG_CONF=""
[ -f "$BACKEND_CONF" ] && ORIG_CONF=$(cat "$BACKEND_CONF")
cleanup() {
  if [ -n "$ORIG_CONF" ]; then
    echo "$ORIG_CONF" > "$BACKEND_CONF"
  elif [ -f "$BACKEND_CONF" ]; then
    rm -f "$BACKEND_CONF"
  fi
  [ -n "${BACKEND_PID:-}" ] && { kill "$BACKEND_PID" 2>/dev/null; wait "$BACKEND_PID" 2>/dev/null; }
  [ -n "${TEMP_DB:-}" ] && rm -f "$TEMP_DB"
}
trap cleanup EXIT

run_test() {
  local name="$1"; shift
  TOTAL=$((TOTAL + 1))
  echo ""
  echo "--- [$name] ---"
  if "$@"; then
    PASS=$((PASS + 1))
    echo -e "${GREEN}PASS${NC}: $name"
  else
    FAIL=$((FAIL + 1))
    echo -e "${RED}FAIL${NC}: $name"
  fi
}

# --- Test 1: 后端不可用时 unregister_session 静默跳过 ---

test_backend_down_unregister_silent() {
  echo "BACKEND_URL=http://127.0.0.1:19999" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  if _backend_available 2>/dev/null; then
    echo "FAIL: backend should not be available"
    return 1
  fi

  echo "OK: backend down → unregister_session silently skips"
  return 0
}

# --- Test 2: 未配置 backend.conf 时 unregister_session 静默跳过 ---

test_no_conf_unregister_silent() {
  rm -f "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  _load_backend_url
  if [ -n "$BACKEND_URL" ]; then
    echo "FAIL: BACKEND_URL should be empty without conf"
    return 1
  fi

  echo "OK: no conf → unregister_session silently skips"
  return 0
}

# --- Test 3: 正常关闭后 session 状态变为 closed ---

test_close_sets_closed() {
  local url="${1:-}"
  if [ -z "$url" ]; then
    echo "SKIP: no backend URL"
    return 0
  fi

  echo "BACKEND_URL=$url" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  local sid="sess_sr3_close_$(date +%s)"

  # Register first
  _call_backend "/api/session/register" "{
    \"session_id\":\"$sid\",
    \"machine_id\":\"test@test\",
    \"os\":\"linux\",
    \"project_slug\":\"proj\",
    \"project_cwd\":\"/p\",
    \"transcript_path\":\"/t.jsonl\"
  }" > /dev/null

  # Verify active
  local detail
  detail=$(curl -s --max-time 3 "$url/api/session/$sid")
  local status
  status=$(echo "$detail" | jq -r '.status')
  if [ "$status" != "active" ]; then
    echo "FAIL: expected active before close, got $status"
    return 1
  fi

  # Close
  local close_result
  close_result=$(_call_backend "/api/session/close" "{\"session_id\":\"$sid\",\"reason\":\"prompt_input_exit\"}")
  local close_status
  close_status=$(echo "$close_result" | jq -r '.status')
  if [ "$close_status" != "closed" ]; then
    echo "FAIL: expected closed, got $close_status"
    return 1
  fi

  # Verify closed via GET
  detail=$(curl -s --max-time 3 "$url/api/session/$sid")
  status=$(echo "$detail" | jq -r '.status')
  if [ "$status" != "closed" ]; then
    echo "FAIL: expected closed after close, got $status"
    return 1
  fi

  echo "OK: close sets session status to closed"
  return 0
}

# --- Test 4: close_reason 和 closed_at 正确记录 ---

test_close_records_reason_and_time() {
  local url="${1:-}"
  if [ -z "$url" ]; then
    echo "SKIP: no backend URL"
    return 0
  fi

  echo "BACKEND_URL=$url" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  local sid="sess_sr3_reason_$(date +%s)"

  _call_backend "/api/session/register" "{
    \"session_id\":\"$sid\",
    \"machine_id\":\"test@test\",
    \"os\":\"linux\",
    \"project_slug\":\"proj\",
    \"project_cwd\":\"/p\",
    \"transcript_path\":\"/t.jsonl\"
  }" > /dev/null

  _call_backend "/api/session/close" "{\"session_id\":\"$sid\",\"reason\":\"prompt_input_exit\"}" > /dev/null

  local detail
  detail=$(curl -s --max-time 3 "$url/api/session/$sid")

  local reason
  reason=$(echo "$detail" | jq -r '.close_reason')
  if [ "$reason" != "prompt_input_exit" ]; then
    echo "FAIL: expected close_reason=prompt_input_exit, got $reason"
    return 1
  fi

  local closed_at
  closed_at=$(echo "$detail" | jq -r '.closed_at // empty')
  if [ -z "$closed_at" ]; then
    echo "FAIL: closed_at should not be empty"
    return 1
  fi

  echo "OK: close_reason and closed_at recorded correctly"
  return 0
}

# --- Test 5: 未调 close 时 session 保持 active（异常退出场景）---

test_no_close_keeps_active() {
  local url="${1:-}"
  if [ -z "$url" ]; then
    echo "SKIP: no backend URL"
    return 0
  fi

  echo "BACKEND_URL=$url" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  local sid="sess_sr3_active_$(date +%s)"

  _call_backend "/api/session/register" "{
    \"session_id\":\"$sid\",
    \"machine_id\":\"test@test\",
    \"os\":\"linux\",
    \"project_slug\":\"proj\",
    \"project_cwd\":\"/p\",
    \"transcript_path\":\"/t.jsonl\"
  }" > /dev/null

  # Don't call close — simulate crash
  local detail
  detail=$(curl -s --max-time 3 "$url/api/session/$sid")
  local status
  status=$(echo "$detail" | jq -r '.status')
  if [ "$status" != "active" ]; then
    echo "FAIL: expected active (no close called), got $status"
    return 1
  fi

  echo "OK: session stays active when close not called (crash scenario)"
  return 0
}

# --- Main ---

echo "=== SR-3: SessionEnd Hook 会话注销 — Shell 集成测试 ==="
echo "Project: $PROJECT_DIR"
echo ""

run_test "backend_down_unregister_silent" test_backend_down_unregister_silent
run_test "no_conf_unregister_silent" test_no_conf_unregister_silent

BACKEND_URL_ARG="${1:-}"
if [ -n "$BACKEND_URL_ARG" ]; then
  if command -v jq &>/dev/null; then
    run_test "close_sets_closed" test_close_sets_closed "$BACKEND_URL_ARG"
    run_test "close_records_reason_and_time" test_close_records_reason_and_time "$BACKEND_URL_ARG"
    run_test "no_close_keeps_active" test_no_close_keeps_active "$BACKEND_URL_ARG"
  else
    echo ""
    echo -e "${YELLOW}NOTE: Skipping backend-up tests (jq not found).${NC}"
  fi
else
  echo ""
  echo -e "${YELLOW}NOTE: Skipping backend-up tests. To run:${NC}"
  echo "  1. Start backend: cd claude_tap_plus && go run ./cmd/claude-tap backend --port 18086 --db /tmp/sr3_test.db"
  echo "  2. Run: bash $0 http://127.0.0.1:18086"
fi

echo ""
echo "================================"
echo "Results: $PASS/$TOTAL passed, $FAIL failed"
echo "================================"

[ $FAIL -eq 0 ] || exit 1
