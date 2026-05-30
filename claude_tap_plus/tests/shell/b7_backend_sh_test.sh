#!/bin/bash
# B7: 公共后端调用模块与服务启动 — Shell 测试
# 验收标准：
#   1. backend.sh 可被各技能脚本正确 source
#   2. 未配置 backend.conf 时，所有后端调用静默跳过
#   3. 后端不可用时，_backend_available 在 2 秒内返回失败
#   4. _call_backend 超时 5 秒后返回空，不阻塞技能主流程
#   5. /health 端点返回正确响应（配合后端启动验证）
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
BACKEND_SH="$PROJECT_DIR/.claude/skills/backend.sh"
BACKEND_CONF="$PROJECT_DIR/.claude/backend.conf"

export CLAUDE_PROJECT_DIR="$PROJECT_DIR"
export SKILL_TAG="b7-test"
skill_log() { :; }
export -f skill_log

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

TOTAL=0
PASS=0
FAIL=0

# Backup/restore original config
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

# --- Test 1: backend.sh 可被正确 source ---

test_source_backend_sh() {
  BACKEND_URL=""
  # source 在当前 shell 中执行（不能用 $() 子 shell）
  source "$BACKEND_SH" 2>&1
  if [ $? -ne 0 ]; then
    echo "FAIL: source returned non-zero"
    return 1
  fi

  # 验证关键函数已定义
  for fn in _load_backend_url _backend_available _call_backend _get_session_id update_issue_status _status_to_label _sync_github_label; do
    if ! type "$fn" &>/dev/null; then
      echo "FAIL: function $fn not defined after source"
      return 1
    fi
  done

  echo "OK: source succeeded, all functions defined"
  return 0
}

# --- Test 2: 未配置 backend.conf 时静默跳过 ---

test_no_config_silent_skip() {
  rm -f "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  # _load_backend_url 应失败
  if _load_backend_url; then
    echo "FAIL: _load_backend_url should fail without conf"
    return 1
  fi

  # _backend_available 应返回失败
  if _backend_available; then
    echo "FAIL: _backend_available should be false without conf"
    return 1
  fi

  # _call_backend 应返回失败（无输出）
  local output
  output=$(_call_backend "/api/issue/check" '{"repo":"t","issue_numbers":[1]}')
  if [ $? -eq 0 ]; then
    echo "FAIL: _call_backend should return non-zero without conf"
    return 1
  fi
  if [ -n "$output" ]; then
    echo "FAIL: _call_backend produced output without conf: $output"
    return 1
  fi

  # update_issue_status 应静默返回 0
  output=$(update_issue_status 1 "fixing" 2>&1)
  if [ $? -ne 0 ]; then
    echo "FAIL: update_issue_status should return 0 without conf"
    return 1
  fi
  if [ -n "$output" ]; then
    echo "FAIL: update_issue_status produced output: $output"
    return 1
  fi

  echo "OK: all calls silently skip without backend.conf"
  return 0
}

# --- Test 3: _backend_available 在 2 秒内返回失败 ---

test_backend_available_timeout() {
  # 配置一个不可达的地址
  echo "BACKEND_URL=http://127.0.0.1:19999" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  local start end elapsed
  start=$(date +%s)
  _backend_available
  local rc=$?
  end=$(date +%s)
  elapsed=$((end - start))

  if [ $rc -eq 0 ]; then
    echo "FAIL: _backend_available should return false for unreachable backend"
    return 1
  fi

  # 验证在 3 秒内返回（curl --max-time 2 + 小余量）
  if [ $elapsed -gt 3 ]; then
    echo "FAIL: _backend_available took ${elapsed}s, expected <= 3s"
    return 1
  fi

  echo "OK: _backend_available returned in ${elapsed}s (rc=$rc)"
  return 0
}

# --- Test 4: _call_backend 在 5 秒后返回空 ---

test_call_backend_timeout() {
  echo "BACKEND_URL=http://127.0.0.1:19999" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  local start end elapsed
  start=$(date +%s)
  local output
  output=$(_call_backend "/api/issue/check" '{"repo":"t","issue_numbers":[1]}')
  local rc=$?
  end=$(date +%s)
  elapsed=$((end - start))

  if [ $rc -eq 0 ]; then
    echo "FAIL: _call_backend should return non-zero for unreachable backend"
    return 1
  fi
  if [ -n "$output" ]; then
    echo "FAIL: _call_backend should return empty, got: $output"
    return 1
  fi

  # 验证在 6 秒内返回（curl --max-time 5 + 小余量）
  if [ $elapsed -gt 6 ]; then
    echo "FAIL: _call_backend took ${elapsed}s, expected <= 6s"
    return 1
  fi

  echo "OK: _call_backend returned empty in ${elapsed}s (rc=$rc)"
  return 0
}

# --- Test 5: /health 端点返回正确响应（需后端运行） ---

test_health_with_backend() {
  local url="${1:-}"
  if [ -z "$url" ]; then
    echo "SKIP: no backend URL"
    return 0
  fi

  echo "BACKEND_URL=$url" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  if ! _backend_available; then
    echo "FAIL: backend not available at $url"
    return 1
  fi

  # 直接 curl 验证 /health 响应
  local resp
  resp=$(curl -s "$url/health")
  if [ "$resp" != '{"status":"ok"}' ]; then
    echo "FAIL: /health returned: $resp"
    return 1
  fi

  echo "OK: /health returns correct response"
  return 0
}

# --- Test 6: _status_to_label 映射正确 ---

test_status_to_label_mapping() {
  source "$BACKEND_SH"

  local cases="claimed:in-progress fixing:fixing ready-for-pr:ready-for-pr pr-created:pr-created testing:testing reviewing:reviewing rejected:rejected merged: idle: unknown:"

  for pair in $cases; do
    local status="${pair%%:*}"
    local expected="${pair##*:}"
    local got
    got=$(_status_to_label "$status")
    if [ "$got" != "$expected" ]; then
      echo "FAIL: _status_to_label($status) = '$got', expected '$expected'"
      return 1
    fi
  done

  echo "OK: all status-to-label mappings correct"
  return 0
}

# --- Test 7: _get_session_id 读取 CLAUDE_SESSION_ID ---

test_get_session_id() {
  source "$BACKEND_SH"

  CLAUDE_SESSION_ID="test-session-123"
  local got
  got=$(_get_session_id)
  if [ "$got" != "test-session-123" ]; then
    echo "FAIL: _get_session_id returned '$got', expected 'test-session-123'"
    return 1
  fi

  echo "OK: _get_session_id reads CLAUDE_SESSION_ID"
  return 0
}

# --- Main ---

echo "=== B7: 公共后端调用模块与服务启动 — Shell 测试 ==="
echo "Project: $PROJECT_DIR"
echo ""

run_test "source_backend_sh" test_source_backend_sh
run_test "no_config_silent_skip" test_no_config_silent_skip
run_test "backend_available_timeout" test_backend_available_timeout
run_test "call_backend_timeout" test_call_backend_timeout
run_test "status_to_label_mapping" test_status_to_label_mapping
run_test "get_session_id" test_get_session_id

# Tests that need a running backend
BACKEND_URL_ARG="${1:-}"
if [ -n "$BACKEND_URL_ARG" ]; then
  run_test "health_with_backend" test_health_with_backend "$BACKEND_URL_ARG"
else
  echo ""
  echo -e "${YELLOW}NOTE: Skipping health endpoint test. To run:${NC}"
  echo "  1. Start backend: go run ./cmd/claude-tap backend --port 18080 --db /tmp/b7_test.db"
  echo "  2. Run: bash $0 http://127.0.0.1:18080"
fi

# --- Summary ---
echo ""
echo "================================"
echo "Results: $PASS/$TOTAL passed, $FAIL failed"
echo "================================"

[ $FAIL -eq 0 ] || exit 1
