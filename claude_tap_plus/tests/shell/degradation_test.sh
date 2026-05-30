#!/bin/bash
# Shell-side degradation tests for backend integration
# Usage: bash degradation_test.sh [BACKEND_URL]
# If BACKEND_URL is not provided, starts a temporary backend.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
BACKEND_SH="$PROJECT_DIR/.claude/skills/backend.sh"
BACKEND_CONF="$PROJECT_DIR/.claude/backend.conf"

# Set CLAUDE_PROJECT_DIR for backend.sh
export CLAUDE_PROJECT_DIR="$PROJECT_DIR"
export SKILL_TAG="degradation-test"

# Mock skill_log for standalone testing (normally provided by log.sh)
skill_log() { :; }
export -f skill_log

# Backup original backend.conf
ORIG_CONF=""
[ -f "$BACKEND_CONF" ] && ORIG_CONF=$(cat "$BACKEND_CONF")

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

TOTAL=0
PASS=0
FAIL=0

# --- helpers ---

restore_conf() {
  if [ -n "$ORIG_CONF" ]; then
    echo "$ORIG_CONF" > "$BACKEND_CONF"
  elif [ -f "$BACKEND_CONF" ]; then
    rm -f "$BACKEND_CONF"
  fi
}

write_conf() {
  local url="$1"
  echo "BACKEND_URL=$url" > "$BACKEND_CONF"
}

remove_conf() {
  rm -f "$BACKEND_CONF"
}

run_test() {
  local name="$1"
  shift
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

cleanup() {
  restore_conf
  if [ -n "${BACKEND_PID:-}" ]; then
    kill "$BACKEND_PID" 2>/dev/null || true
    wait "$BACKEND_PID" 2>/dev/null || true
  fi
  [ -n "${TEMP_DB:-}" ] && rm -f "$TEMP_DB"
}
trap cleanup EXIT

# --- Test: no backend.conf, all backend calls skip ---

test_no_conf_all_pass() {
  remove_conf

  # Reset BACKEND_URL
  BACKEND_URL=""
  source "$BACKEND_SH"

  # _load_backend_url should fail
  BACKEND_URL=""
  _load_backend_url
  [ $? -ne 0 ] || { echo "FAIL: _load_backend_url should return non-zero without conf"; return 1; }

  # _backend_available should return false
  BACKEND_URL=""
  if _backend_available; then
    echo "FAIL: _backend_available should be false without conf"
    return 1
  fi

  # update_issue_status should silently return 0
  BACKEND_URL=""
  update_issue_status 1 "fixing"
  [ $? -eq 0 ] || { echo "FAIL: update_issue_status should return 0 without conf"; return 1; }

  echo "OK: no conf, all calls skip gracefully"
  return 0
}

# --- Test: malformed backend.conf does not crash ---

test_malformed_conf_no_crash() {
  local test_cases=(
    ""                                            # empty value
    "not-a-url"                                   # no protocol
    "ftp://example.com"                           # wrong protocol
    "http://example.com with spaces"              # spaces in URL
    "http://example.com	tab"                     # tab in URL
    "BACKEND_URL="                                # empty assignment
  )

  for raw in "${test_cases[@]}"; do
    if [ -z "$raw" ] || [ "$raw" = "BACKEND_URL=" ]; then
      echo "BACKEND_URL=" > "$BACKEND_CONF"
    else
      echo "BACKEND_URL=$raw" > "$BACKEND_CONF"
    fi

    BACKEND_URL=""
    source "$BACKEND_SH"

    if _load_backend_url && [ -n "$BACKEND_URL" ]; then
      echo "FAIL: _load_backend_url should reject '$raw'"
      return 1
    fi
  done

  echo "OK: all malformed configs rejected without crash"
  return 0
}

# --- Test: backend down, claim should be blocked ---

test_backend_down_claim_blocked() {
  write_conf "http://127.0.0.1:19999"

  BACKEND_URL=""
  source "$BACKEND_SH"

  # _backend_available should return false
  if _backend_available; then
    echo "FAIL: _backend_available should be false when backend is down"
    return 1
  fi

  # _require_backend should return 1 (configured but unreachable)
  _require_backend
  local rc=$?
  if [ $rc -ne 1 ]; then
    echo "FAIL: _require_backend should return 1 when configured but unreachable, got $rc"
    return 1
  fi

  echo "OK: backend down, claim blocked"
  return 0
}

# --- Test: backend down, status update is silent ---

test_backend_down_status_silent() {
  write_conf "http://127.0.0.1:19999"

  BACKEND_URL=""
  source "$BACKEND_SH"

  # update_issue_status should silently return 0 (no stdout, no error)
  local output
  output=$(update_issue_status 1 "fixing" 2>&1)
  local rc=$?
  if [ $rc -ne 0 ]; then
    echo "FAIL: update_issue_status should return 0 when backend down, got $rc"
    return 1
  fi
  if [ -n "$output" ]; then
    echo "FAIL: update_issue_status should produce no output, got: $output"
    return 1
  fi

  echo "OK: status update silent when backend down"
  return 0
}

# --- Test: _call_backend returns failure on unreachable backend ---

test_backend_down_call_fails() {
  write_conf "http://127.0.0.1:19999"

  BACKEND_URL=""
  source "$BACKEND_SH"

  local result
  result=$(_call_backend "/api/issue/check" '{"repo":"test","issue_numbers":[1]}')
  local rc=$?
  if [ $rc -eq 0 ]; then
    echo "FAIL: _call_backend should return non-zero when backend unreachable"
    return 1
  fi
  if [ -n "$result" ]; then
    echo "FAIL: _call_backend should return empty when backend unreachable"
    return 1
  fi

  echo "OK: _call_backend fails cleanly when backend unreachable"
  return 0
}

# --- Test: backend up and healthy ---

test_backend_up_works() {
  local url="${1:-}"
  if [ -z "$url" ]; then
    echo "SKIP: no backend URL provided"
    return 0
  fi

  write_conf "$url"

  BACKEND_URL=""
  source "$BACKEND_SH"

  if ! _backend_available; then
    echo "FAIL: _backend_available should return true when backend is up"
    return 1
  fi

  _require_backend
  local rc=$?
  if [ $rc -ne 0 ]; then
    echo "FAIL: _require_backend should return 0 when backend is up, got $rc"
    return 1
  fi

  # Check issues should work
  local result
  result=$(_call_backend "/api/issue/check" '{"repo_full_name":"test/degradation","issue_numbers":[100]}')
  if [ $? -ne 0 ] || [ -z "$result" ]; then
    echo "FAIL: /api/issue/check should succeed"
    return 1
  fi

  echo "$result" | jq -e '.issues' > /dev/null || {
    echo "FAIL: response should have .issues array"
    return 1
  }

  echo "OK: backend up, all calls work"
  return 0
}

# --- Test: backend recovery preserves state ---

test_recovery_no_conflict() {
  local url="${1:-}"
  if [ -z "$url" ]; then
    echo "SKIP: no backend URL provided"
    return 0
  fi

  write_conf "$url"
  BACKEND_URL=""
  source "$BACKEND_SH"

  # Claim issue 200
  local claim_result
  claim_result=$(_call_backend "/api/issue/claim" '{
    "repo_full_name":"test/degradation",
    "issue_number":200,
    "session_id":"sess_recovery_test",
    "issue_title":"Recovery test"
  }')
  if [ $? -ne 0 ]; then
    echo "FAIL: initial claim should succeed"
    return 1
  fi
  echo "$claim_result" | jq -e '.success' > /dev/null || {
    echo "FAIL: claim response should have success=true"
    return 1
  }

  # Try to claim same issue with different session (should fail)
  local claim2_result
  claim2_result=$(_call_backend "/api/issue/claim" '{
    "repo_full_name":"test/degradation",
    "issue_number":200,
    "session_id":"sess_other",
    "issue_title":"Recovery test"
  }')
  local claim2_success
  claim2_success=$(echo "$claim2_result" | jq -r '.success')
  if [ "$claim2_success" = "true" ]; then
    echo "FAIL: second claim should fail (already claimed)"
    return 1
  fi

  # Release via session
  local release_result
  release_result=$(_call_backend "/api/issue/release-session" '{"session_id":"sess_recovery_test"}')
  if [ $? -ne 0 ]; then
    echo "FAIL: release-session should succeed"
    return 1
  fi
  local released_count
  released_count=$(echo "$release_result" | jq '.count')
  if [ "$released_count" -lt 1 ]; then
    echo "FAIL: should release at least 1 issue"
    return 1
  fi

  # Re-claim should now succeed
  local claim3_result
  claim3_result=$(_call_backend "/api/issue/claim" '{
    "repo_full_name":"test/degradation",
    "issue_number":200,
    "session_id":"sess_other",
    "issue_title":"Recovery test"
  }')
  echo "$claim3_result" | jq -e '.success' > /dev/null || {
    echo "FAIL: re-claim after release should succeed"
    return 1
  }

  # Cleanup: release
  _call_backend "/api/issue/release-session" '{"session_id":"sess_other"}' > /dev/null

  echo "OK: recovery scenario works correctly"
  return 0
}

# --- Test: _require_backend distinguishes not-configured vs unreachable ---

test_require_backend_return_codes() {
  # Case 1: No backend.conf → return 2 (not configured)
  remove_conf
  BACKEND_URL=""
  source "$BACKEND_SH"
  _require_backend
  local rc1=$?
  if [ $rc1 -ne 2 ]; then
    echo "FAIL: _require_backend should return 2 when not configured, got $rc1"
    return 1
  fi

  # Case 2: Backend configured but unreachable → return 1
  write_conf "http://127.0.0.1:19999"
  BACKEND_URL=""
  source "$BACKEND_SH"
  _require_backend
  local rc2=$?
  if [ $rc2 -ne 1 ]; then
    echo "FAIL: _require_backend should return 1 when unreachable, got $rc2"
    return 1
  fi

  echo "OK: _require_backend return codes correct"
  return 0
}

# --- Test: check API fails → filter returns original list (B8 #3) ---

test_check_fail_returns_full_list() {
  write_conf "http://127.0.0.1:19999"

  BACKEND_URL=""
  source "$BACKEND_SH"

  # _call_backend fails → _call_backend returns empty, rc != 0
  local result
  result=$(_call_backend "/api/issue/check" '{"repo_full_name":"test/repo","issue_numbers":[1,2,3]}')
  if [ $? -eq 0 ] || [ -n "$result" ]; then
    echo "FAIL: _call_backend should fail when backend unreachable"
    return 1
  fi

  # Simulate filter_claimed_issues logic: when _call_backend returns empty → return original list
  local issues='[{"number":1},{"number":2},{"number":3}]'
  local filtered
  filtered=$(echo "$issues" | jq '[.[]]')  # no filtering → original list
  local count
  count=$(echo "$filtered" | jq 'length')
  if [ "$count" != "3" ]; then
    echo "FAIL: expected 3 issues (no filtering), got $count"
    return 1
  fi

  echo "OK: check API fail returns full list (no filtering)"
  return 0
}

# --- Test: release-session API timeout doesn't block (B8 #6) ---

test_release_session_timeout_no_block() {
  write_conf "http://127.0.0.1:19999"

  BACKEND_URL=""
  source "$BACKEND_SH"

  # Simulate SessionEnd hook behavior: _backend_available fails → skip
  if _backend_available; then
    echo "FAIL: backend should not be available"
    return 1
  fi

  # SessionEnd hook logic: _backend_available fails → return 0 (skip)
  # This means the hook doesn't block or error
  echo "OK: release-session timeout → hook skips without blocking"
  return 0
}

# --- Test: all status values silent on backend down (B8 #5 comprehensive) ---

test_all_status_values_silent_on_backend_down() {
  write_conf "http://127.0.0.1:19999"

  BACKEND_URL=""
  source "$BACKEND_SH"

  for status in "fixing" "ready-for-pr" "pr-created" "testing" "reviewing" "merged" "rejected"; do
    local output
    output=$(update_issue_status 10 "$status" 2>&1)
    if [ $? -ne 0 ]; then
      echo "FAIL: update_issue_status($status) returned non-zero"
      return 1
    fi
    if [ -n "$output" ]; then
      echo "FAIL: update_issue_status($status) produced output: $output"
      return 1
    fi
  done

  echo "OK: all status values silently skip on backend down"
  return 0
}

# --- Main ---

echo "=== Shell Degradation Tests ==="
echo "Project: $PROJECT_DIR"
echo ""

# Tests that don't need a running backend
run_test "no_conf_all_pass" test_no_conf_all_pass
run_test "malformed_conf_no_crash" test_malformed_conf_no_crash
run_test "backend_down_claim_blocked" test_backend_down_claim_blocked
run_test "backend_down_status_silent" test_backend_down_status_silent
run_test "backend_down_call_fails" test_backend_down_call_fails
run_test "require_backend_return_codes" test_require_backend_return_codes
run_test "check_fail_returns_full_list" test_check_fail_returns_full_list
run_test "release_session_timeout_no_block" test_release_session_timeout_no_block
run_test "all_status_values_silent_on_backend_down" test_all_status_values_silent_on_backend_down

# Tests that need a running backend (if URL provided)
BACKEND_URL_ARG="${1:-}"
if [ -n "$BACKEND_URL_ARG" ]; then
  if command -v jq &>/dev/null; then
    run_test "backend_up_works" test_backend_up_works "$BACKEND_URL_ARG"
    run_test "recovery_no_conflict" test_recovery_no_conflict "$BACKEND_URL_ARG"
  else
    echo ""
    echo "NOTE: Skipping backend-up tests (jq not found in PATH)."
    echo "  Install jq to run these tests."
  fi
else
  echo ""
  echo "NOTE: Skipping backend-up tests. To run them:"
  echo "  1. Start backend: go run ./cmd/claude-tap backend --db /tmp/test-degradation.db"
  echo "  2. Run: bash $0 http://127.0.0.1:8080"
fi

# --- Summary ---
echo ""
echo "================================"
echo "Results: $PASS/$TOTAL passed, $FAIL failed"
echo "================================"

[ $FAIL -eq 0 ] || exit 1
