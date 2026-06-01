#!/bin/bash
# B6-1: 001-4-issue-claim 技能改造 — Shell 集成测试
# 验收标准：
#   1. 后端可用时，只展示空闲 issue（check API 过滤非 idle）
#   2. 后端可用时，领取操作是原子的（先锁后 gh）
#   3. 后端可用时，领取成功后 GitHub 自动添加 in-progress label
#   4. 后端不可用时，回退到原有行为
#   5. 已被领取的 issue 不出现在列表中
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
BACKEND_SH="$PROJECT_DIR/.claude/skills/backend.sh"
BACKEND_CONF="$PROJECT_DIR/.claude/backend.conf"

export CLAUDE_PROJECT_DIR="$PROJECT_DIR"
export SKILL_TAG="b6_1-test"
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

# --- Test 1: _call_backend 后端不可用时静默失败 ---

test_backend_down_silent_fail() {
  echo "BACKEND_URL=http://127.0.0.1:19999" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  local output
  output=$(_call_backend "/api/issue/check" '{"repo_full_name":"test/repo","issue_numbers":[1]}' 2>&1)
  local rc=$?
  if [ $rc -eq 0 ]; then
    echo "FAIL: _call_backend should fail when backend down, got rc=0"
    return 1
  fi

  echo "OK: _call_backend returns non-zero when backend down"
  return 0
}

# --- Test 2: 未配置 backend.conf 时 _load_backend_url 返回失败 ---

test_no_conf_load_fails() {
  rm -f "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  _load_backend_url
  if [ $? -eq 0 ]; then
    echo "FAIL: _load_backend_url should fail without conf"
    return 1
  fi
  if [ -n "$BACKEND_URL" ]; then
    echo "FAIL: BACKEND_URL should be empty, got '$BACKEND_URL'"
    return 1
  fi

  echo "OK: _load_backend_url fails without conf"
  return 0
}

# --- Test 3: _backend_available 后端不可用时返回 1 ---

test_backend_available_returns_false() {
  echo "BACKEND_URL=http://127.0.0.1:19999" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  if _backend_available 2>/dev/null; then
    echo "FAIL: _backend_available should return false for unreachable backend"
    return 1
  fi

  echo "OK: _backend_available returns false when backend unreachable"
  return 0
}

# --- Test 4: _status_to_label("claimed") 返回 "in-progress" ---

test_claimed_label_mapping() {
  source "$BACKEND_SH"

  local label
  label=$(_status_to_label "claimed")
  if [ "$label" != "in-progress" ]; then
    echo "FAIL: _status_to_label(claimed)='$label', expected 'in-progress'"
    return 1
  fi

  echo "OK: _status_to_label(claimed) = in-progress"
  return 0
}

# --- Test 5: _require_backend 后端不可用时返回 1 ---

test_require_backend_unreachable() {
  echo "BACKEND_URL=http://127.0.0.1:19999" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  _require_backend
  local rc=$?
  if [ $rc -ne 1 ]; then
    echo "FAIL: _require_backend should return 1 when configured but unreachable, got $rc"
    return 1
  fi

  echo "OK: _require_backend returns 1 when backend unreachable"
  return 0
}

# --- Test 6: _require_backend 未配置时返回 2 ---

test_require_backend_not_configured() {
  rm -f "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  _require_backend
  local rc=$?
  if [ $rc -ne 2 ]; then
    echo "FAIL: _require_backend should return 2 when not configured, got $rc"
    return 1
  fi

  echo "OK: _require_backend returns 2 when not configured"
  return 0
}

# --- Test 7: 后端可用时 check API 过滤非 idle issue ---

test_check_filters_non_idle() {
  local url="${1:-}"
  if [ -z "$url" ]; then
    echo "SKIP: no backend URL"
    return 0
  fi

  echo "BACKEND_URL=$url" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  local repo="test/b6_1"
  local sid="sess_b6_1_check"

  # 预置：#600 idle, #601 claimed
  _call_backend "/api/issue/check" "{\"repo_full_name\":\"$repo\",\"issue_numbers\":[600,601]}" > /dev/null
  _call_backend "/api/issue/claim" "{\"repo_full_name\":\"$repo\",\"issue_number\":601,\"session_id\":\"$sid\"}" > /dev/null

  # Check → 只有 #600 是 idle
  local check
  check=$(_call_backend "/api/issue/check" "{\"repo_full_name\":\"$repo\",\"issue_numbers\":[600,601]}")

  local s600 s601
  s600=$(echo "$check" | jq -r '.issues[] | select(.number==600) | .status')
  s601=$(echo "$check" | jq -r '.issues[] | select(.number==601) | .status')

  if [ "$s600" != "idle" ]; then
    echo "FAIL: #600 expected idle, got $s600"
    return 1
  fi
  if [ "$s601" != "claimed" ]; then
    echo "FAIL: #601 expected claimed, got $s601"
    return 1
  fi

  echo "OK: check API correctly returns idle/claimed statuses"
  return 0
}

# --- Test 8: 后端可用时原子领取 ---

test_claim_atomic() {
  local url="${1:-}"
  if [ -z "$url" ]; then
    echo "SKIP: no backend URL"
    return 0
  fi

  echo "BACKEND_URL=$url" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  local repo="test/b6_1"
  local sid_a="sess_b6_1_a"
  local sid_b="sess_b6_1_b"

  # 确保 #700 idle
  _call_backend "/api/issue/check" "{\"repo_full_name\":\"$repo\",\"issue_numbers\":[700]}" > /dev/null

  # Session A claim → success
  local result_a
  result_a=$(_call_backend "/api/issue/claim" "{\"repo_full_name\":\"$repo\",\"issue_number\":700,\"session_id\":\"$sid_a\"}")
  local success_a
  success_a=$(echo "$result_a" | jq -r '.success')
  if [ "$success_a" != "true" ]; then
    echo "FAIL: session A claim failed: $result_a"
    return 1
  fi

  # Session B claim → fail (already claimed)
  local result_b
  result_b=$(_call_backend "/api/issue/claim" "{\"repo_full_name\":\"$repo\",\"issue_number\":700,\"session_id\":\"$sid_b\"}")
  local success_b
  success_b=$(echo "$result_b" | jq -r '.success')
  if [ "$success_b" != "false" ]; then
    echo "FAIL: session B should not claim: $result_b"
    return 1
  fi

  # 验证 claimed_by = sid_a
  local claimed_by
  claimed_by=$(echo "$result_b" | jq -r '.claimed_by // empty')
  if [ "$claimed_by" != "$sid_a" ]; then
    echo "FAIL: expected claimed_by=$sid_a, got $claimed_by"
    return 1
  fi

  # Session A 幂等 → success
  local result_idem
  result_idem=$(_call_backend "/api/issue/claim" "{\"repo_full_name\":\"$repo\",\"issue_number\":700,\"session_id\":\"$sid_a\"}")
  local success_idem
  success_idem=$(echo "$result_idem" | jq -r '.success')
  if [ "$success_idem" != "true" ]; then
    echo "FAIL: idempotent claim should succeed: $result_idem"
    return 1
  fi

  echo "OK: atomic claim works (A succeeds, B blocked, A idempotent)"
  return 0
}

# --- Test 9: claim 后 status 为 claimed ---

test_claim_sets_claimed_status() {
  local url="${1:-}"
  if [ -z "$url" ]; then
    echo "SKIP: no backend URL"
    return 0
  fi

  echo "BACKEND_URL=$url" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  local repo="test/b6_1"
  local sid="sess_b6_1_status"

  _call_backend "/api/issue/check" "{\"repo_full_name\":\"$repo\",\"issue_numbers\":[800]}" > /dev/null
  _call_backend "/api/issue/claim" "{\"repo_full_name\":\"$repo\",\"issue_number\":800,\"session_id\":\"$sid\"}" > /dev/null

  local check
  check=$(_call_backend "/api/issue/check" "{\"repo_full_name\":\"$repo\",\"issue_numbers\":[800]}")
  local status
  status=$(echo "$check" | jq -r '.issues[0].status')
  if [ "$status" != "claimed" ]; then
    echo "FAIL: expected claimed, got $status"
    return 1
  fi

  echo "OK: claim sets status to claimed"
  return 0
}

# --- Main ---

echo "=== B6-1: 001-4-issue-claim 技能改造 — Shell 集成测试 ==="
echo "Project: $PROJECT_DIR"
echo ""

run_test "backend_down_silent_fail" test_backend_down_silent_fail
run_test "no_conf_load_fails" test_no_conf_load_fails
run_test "backend_available_returns_false" test_backend_available_returns_false
run_test "claimed_label_mapping" test_claimed_label_mapping
run_test "require_backend_unreachable" test_require_backend_unreachable
run_test "require_backend_not_configured" test_require_backend_not_configured
BACKEND_URL_ARG="${1:-}"
if [ -n "$BACKEND_URL_ARG" ]; then
  if command -v jq &>/dev/null; then
    run_test "check_filters_non_idle" test_check_filters_non_idle "$BACKEND_URL_ARG"
    run_test "claim_atomic" test_claim_atomic "$BACKEND_URL_ARG"
    run_test "claim_sets_claimed_status" test_claim_sets_claimed_status "$BACKEND_URL_ARG"
  else
    echo ""
    echo -e "${YELLOW}NOTE: Skipping backend-up tests (jq not found).${NC}"
  fi
else
  echo ""
  echo -e "${YELLOW}NOTE: Skipping backend-up tests. To run:${NC}"
  echo "  1. Start backend: cd claude_tap_plus && go run ./cmd/claude-tap backend --port 18083 --db /tmp/b6_1_test.db"
  echo "  2. Run: bash $0 http://127.0.0.1:18083"
fi

echo ""
echo "================================"
echo "Results: $PASS/$TOTAL passed, $FAIL failed"
echo "================================"

[ $FAIL -eq 0 ] || exit 1
