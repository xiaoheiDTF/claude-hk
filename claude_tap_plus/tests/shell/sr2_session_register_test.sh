#!/bin/bash
# SR-2: SessionStart Hook 会话注册 — Shell 集成测试
# 验收标准：
#   1. 代理运行时（后端可用），启动后能在后端查到会话记录
#   2. 代理未运行时（后端不可用），启动无报错，不发送请求
#   3. 后端不可达时，启动无报错，curl 静默失败
#   4. 注册请求包含所有 8 个字段
#   5. source 正确区分 startup 和 resume
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
BACKEND_SH="$PROJECT_DIR/.claude/skills/backend.sh"
BACKEND_CONF="$PROJECT_DIR/.claude/backend.conf"

export CLAUDE_PROJECT_DIR="$PROJECT_DIR"
export SKILL_TAG="sr2-test"
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

# --- Test 1: 后端不可用时 register_session 静默跳过 ---

test_backend_down_register_silent() {
  echo "BACKEND_URL=http://127.0.0.1:19999" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  # _backend_available returns false → register_session skips
  if _backend_available 2>/dev/null; then
    echo "FAIL: backend should not be available"
    return 1
  fi

  echo "OK: backend down → register_session silently skips"
  return 0
}

# --- Test 2: 未配置 backend.conf 时 register_session 静默跳过 ---

test_no_conf_register_silent() {
  rm -f "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  _load_backend_url
  if [ -n "$BACKEND_URL" ]; then
    echo "FAIL: BACKEND_URL should be empty without conf"
    return 1
  fi

  echo "OK: no conf → register_session silently skips"
  return 0
}

# --- Test 3: 后端可用时注册会话（完整字段验证）---

test_register_with_all_fields() {
  local url="${1:-}"
  if [ -z "$url" ]; then
    echo "SKIP: no backend URL"
    return 0
  fi

  echo "BACKEND_URL=$url" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  local sid="sess_sr2_test_$(date +%s)"
  local machine_id="test_user@test_host"
  local os_type="windows"
  local project_slug="test-project-sr2"
  local cwd="/test/cwd"
  local transcript_path="/test/transcript.jsonl"
  local model="test-model"
  local source_type="startup"

  local result
  result=$(_call_backend "/api/session/register" "{
    \"session_id\":\"$sid\",
    \"machine_id\":\"$machine_id\",
    \"os\":\"$os_type\",
    \"project_slug\":\"$project_slug\",
    \"project_cwd\":\"$cwd\",
    \"transcript_path\":\"$transcript_path\",
    \"model\":\"$model\",
    \"source\":\"$source_type\"
  }")

  if [ $? -ne 0 ] || [ -z "$result" ]; then
    echo "FAIL: register call failed"
    return 1
  fi

  local status
  status=$(echo "$result" | jq -r '.status')
  if [ "$status" != "registered" ]; then
    echo "FAIL: expected status=registered, got $status"
    return 1
  fi

  # 验证所有字段通过 GET /api/session/{id}
  local detail
  detail=$(_call_backend "/api/session/$sid" "")
  # GET doesn't work via _call_backend (which uses POST), use curl directly
  detail=$(curl -s --max-time 3 "$url/api/session/$sid")

  local got_model got_source got_os got_machine got_slug got_cwd got_transcript
  got_model=$(echo "$detail" | jq -r '.model')
  got_source=$(echo "$detail" | jq -r '.source')
  got_os=$(echo "$detail" | jq -r '.os')
  got_machine=$(echo "$detail" | jq -r '.machine_id')
  got_slug=$(echo "$detail" | jq -r '.project_slug')
  got_cwd=$(echo "$detail" | jq -r '.project_cwd')
  got_transcript=$(echo "$detail" | jq -r '.transcript_path')

  if [ "$got_model" != "$model" ]; then
    echo "FAIL: model expected=$model got=$got_model"
    return 1
  fi
  if [ "$got_source" != "$source_type" ]; then
    echo "FAIL: source expected=$source_type got=$got_source"
    return 1
  fi
  if [ "$got_os" != "$os_type" ]; then
    echo "FAIL: os expected=$os_type got=$got_os"
    return 1
  fi
  if [ "$got_machine" != "$machine_id" ]; then
    echo "FAIL: machine_id expected=$machine_id got=$got_machine"
    return 1
  fi
  if [ "$got_slug" != "$project_slug" ]; then
    echo "FAIL: project_slug expected=$project_slug got=$got_slug"
    return 1
  fi
  if [ "$got_cwd" != "$cwd" ]; then
    echo "FAIL: project_cwd expected=$cwd got=$got_cwd"
    return 1
  fi
  if [ "$got_transcript" != "$transcript_path" ]; then
    echo "FAIL: transcript_path expected=$transcript_path got=$got_transcript"
    return 1
  fi

  echo "OK: all 8 fields registered correctly"
  return 0
}

# --- Test 4: source 区分 startup 和 resume ---

test_source_startup_vs_resume() {
  local url="${1:-}"
  if [ -z "$url" ]; then
    echo "SKIP: no backend URL"
    return 0
  fi

  echo "BACKEND_URL=$url" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  # Register with source=startup
  local sid_startup="sess_sr2_startup_$(date +%s)"
  _call_backend "/api/session/register" "{
    \"session_id\":\"$sid_startup\",
    \"machine_id\":\"test@test\",
    \"os\":\"linux\",
    \"project_slug\":\"proj\",
    \"project_cwd\":\"/p\",
    \"transcript_path\":\"/t.jsonl\",
    \"model\":\"m\",
    \"source\":\"startup\"
  }" > /dev/null

  local detail_startup
  detail_startup=$(curl -s --max-time 3 "$url/api/session/$sid_startup")
  local got_startup
  got_startup=$(echo "$detail_startup" | jq -r '.source')
  if [ "$got_startup" != "startup" ]; then
    echo "FAIL: source=startup expected, got $got_startup"
    return 1
  fi

  # Register with source=resume
  local sid_resume="sess_sr2_resume_$(date +%s)"
  _call_backend "/api/session/register" "{
    \"session_id\":\"$sid_resume\",
    \"machine_id\":\"test@test\",
    \"os\":\"linux\",
    \"project_slug\":\"proj\",
    \"project_cwd\":\"/p\",
    \"transcript_path\":\"/t2.jsonl\",
    \"model\":\"m\",
    \"source\":\"resume\"
  }" > /dev/null

  local detail_resume
  detail_resume=$(curl -s --max-time 3 "$url/api/session/$sid_resume")
  local got_resume
  got_resume=$(echo "$detail_resume" | jq -r '.source')
  if [ "$got_resume" != "resume" ]; then
    echo "FAIL: source=resume expected, got $got_resume"
    return 1
  fi

  echo "OK: source correctly distinguishes startup vs resume"
  return 0
}

# --- Test 5: 重复注册返回 409 但不报错 ---

test_duplicate_register_no_crash() {
  local url="${1:-}"
  if [ -z "$url" ]; then
    echo "SKIP: no backend URL"
    return 0
  fi

  echo "BACKEND_URL=$url" > "$BACKEND_CONF"
  BACKEND_URL=""
  source "$BACKEND_SH"

  local sid="sess_sr2_dup_$(date +%s)"

  # First register
  _call_backend "/api/session/register" "{
    \"session_id\":\"$sid\",
    \"machine_id\":\"test@test\",
    \"os\":\"linux\",
    \"project_slug\":\"proj\",
    \"project_cwd\":\"/p\",
    \"transcript_path\":\"/t.jsonl\"
  }" > /dev/null

  # Second register (duplicate) → _call_backend returns response but status code is 409
  local result
  result=$(_call_backend "/api/session/register" "{
    \"session_id\":\"$sid\",
    \"machine_id\":\"test@test\",
    \"os\":\"linux\",
    \"project_slug\":\"proj\",
    \"project_cwd\":\"/p\",
    \"transcript_path\":\"/t.jsonl\"
  }")

  local error
  error=$(echo "$result" | jq -r '.error // empty')
  if [ "$error" != "session_exists" ]; then
    echo "FAIL: expected error=session_exists, got $error"
    return 1
  fi

  echo "OK: duplicate register returns session_exists (non-crashing)"
  return 0
}

# --- Main ---

echo "=== SR-2: SessionStart Hook 会话注册 — Shell 集成测试 ==="
echo "Project: $PROJECT_DIR"
echo ""

run_test "backend_down_register_silent" test_backend_down_register_silent
run_test "no_conf_register_silent" test_no_conf_register_silent

BACKEND_URL_ARG="${1:-}"
if [ -n "$BACKEND_URL_ARG" ]; then
  if command -v jq &>/dev/null; then
    run_test "register_with_all_fields" test_register_with_all_fields "$BACKEND_URL_ARG"
    run_test "source_startup_vs_resume" test_source_startup_vs_resume "$BACKEND_URL_ARG"
    run_test "duplicate_register_no_crash" test_duplicate_register_no_crash "$BACKEND_URL_ARG"
  else
    echo ""
    echo -e "${YELLOW}NOTE: Skipping backend-up tests (jq not found).${NC}"
  fi
else
  echo ""
  echo -e "${YELLOW}NOTE: Skipping backend-up tests. To run:${NC}"
  echo "  1. Start backend: cd claude_tap_plus && go run ./cmd/claude-tap backend --port 18085 --db /tmp/sr2_test.db"
  echo "  2. Run: bash $0 http://127.0.0.1:18085"
fi

echo ""
echo "================================"
echo "Results: $PASS/$TOTAL passed, $FAIL failed"
echo "================================"

[ $FAIL -eq 0 ] || exit 1
