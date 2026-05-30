#!/bin/bash
# 01-session-start: 会话开始或恢复时触发 (每个会话一次)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

source_type=$(json_get '.source')
log "INFO" "source=$source_type"

# 首次运行 → 触发统一初始化
[ ! -f "$CLAUDE_PROJECT_DIR/.claude/.initialized" ] && bash "$CLAUDE_PROJECT_DIR/.claude/init.sh"

# ---- 每次会话确保基础环境 ----

# 1. UTF-8 编码
ensure_utf8() {
  local os_type
  case "$(uname -s)" in
    Linux*)   os_type="linux" ;;
    Darwin*)  os_type="macos" ;;
    MINGW*|MSYS*|CYGWIN*) os_type="windows" ;;
    *)        os_type="unknown" ;;
  esac

  export LANG="${LANG:-en_US.UTF-8}"
  export LC_ALL="${LC_ALL:-en_US.UTF-8}"

  if [ "$os_type" = "windows" ]; then
    chcp.com 65001 > /dev/null 2>&1
    log "INFO" "Windows code page set to 65001"
  fi

  log "INFO" "UTF-8 ensured (OS=$os_type, LANG=$LANG)"
}

# 2. Python 可用性（巡检 + 自动修复）
ensure_python_check() {
  source "$CLAUDE_PROJECT_DIR/.claude/scripts/ensure_python.sh"
  local result
  result=$(ensure_python)
  if [ -n "$result" ]; then
    log "INFO" "Python 已就绪: $result"
  else
    log "WARN" "Python 不可用，部分功能受限"
  fi
}

# 3. 目录完整性
ensure_dirs() {
  source "$CLAUDE_PROJECT_DIR/.claude/scripts/ensure_dirs.sh"
  local missing
  missing=$(check_dirs 2>&1)
  if [ "$missing" = "ALL OK" ]; then
    log "INFO" "目录检查通过"
  else
    log "WARN" "目录缺失，重新创建: $missing"
    ensure_all_dirs "$LOG_FILE" > /dev/null
  fi
}

ensure_utf8
ensure_python_check
ensure_dirs

# 4. Skill 级巡检
ensure_skill_checks() {
  local skills_dir="$CLAUDE_PROJECT_DIR/.claude/skills"
  [ -d "$skills_dir" ] || return
  for check_script in "$skills_dir"/*/scripts/init_check.sh; do
    [ -f "$check_script" ] || continue
    local skill_name
    skill_name=$(basename "$(dirname "$(dirname "$check_script")")")
    if bash "$check_script" >> "$LOG_FILE" 2>&1; then
      log "INFO" "Skill check OK: $skill_name"
    else
      log "WARN" "Skill check failed: $skill_name (non-blocking)"
    fi
  done
}

ensure_skill_checks

# 5. 初始化代理 trace 路径（通过后端 8080 转发到代理）
init_trace_path() {
  source "$CLAUDE_PROJECT_DIR/.claude/lib/config.sh"
  load_backend_config || return 0

  local sid
  sid=$(json_get '.session_id')
  [ -z "$sid" ] && return 0

  local machine_id
  machine_id="$(whoami 2>/dev/null || echo unknown)@$(hostname 2>/dev/null || echo unknown)"

  local transcript_path
  transcript_path=$(json_get '.transcript_path')

  local project_slug
  project_slug=$(echo "$transcript_path" | sed -n 's/.*[\.]claude[\\/]\{1\}projects[\\/]\{1\}\([^\\/]*\).*/\1/p')
  [ -z "$project_slug" ] && project_slug=$(basename "$(json_get '.cwd')" 2>/dev/null)

  # Windows 路径反斜杠转义：\ → \\（否则 Go JSON 解析器拒绝非法转义序列如 \U）
  local safe_transcript_path="${transcript_path//\\/\\\\}"

  # 通过后端转发 trace-init 到代理
  local result
  result=$(curl -s --max-time 3 -X POST "$BACKEND_URL/api/proxy/trace-init" \
    -H "Content-Type: application/json" \
    -d "{
      \"session_id\":\"$sid\",
      \"machine_id\":\"$machine_id\",
      \"project_slug\":\"$project_slug\",
      \"transcript_path\":\"$safe_transcript_path\"
    }" 2>/dev/null)

  if [ -n "$result" ]; then
    log "INFO" "Trace initialized: $(echo "$result" | jq -r '.trace_path // "unknown"' 2>/dev/null)"
  else
    log "DEBUG" "Trace init skipped (backend/proxy unreachable)"
  fi
}

init_trace_path

# 6. 注册会话到后端（SR-2）
register_session() {
  source "$CLAUDE_PROJECT_DIR/.claude/skills/backend.sh"
  _load_backend_url
  [ -z "$BACKEND_URL" ] && return 0

  if ! _backend_available; then
    log "DEBUG" "Backend unreachable, skipping session registration"
    return 0
  fi

  local session_id transcript_path cwd model source_type os_type machine_id project_slug

  session_id=$(json_get '.session_id')
  [ -z "$session_id" ] && return 0

  transcript_path=$(json_get '.transcript_path')
  cwd=$(json_get '.cwd')
  model=$(json_get '.model')
  source_type=$(json_get '.source')

  machine_id="$(whoami 2>/dev/null || echo unknown)@$(hostname 2>/dev/null || echo unknown)"

  case "$(uname -s)" in
    Linux*)   os_type="linux" ;;
    Darwin*)  os_type="macos" ;;
    MINGW*|MSYS*|CYGWIN*) os_type="windows" ;;
    *)        os_type="unknown" ;;
  esac

  project_slug=$(echo "$transcript_path" | sed -n 's/.*[\.]claude[\\/]\{1\}projects[\\/]\{1\}\([^\\/]*\).*/\1/p')
  [ -z "$project_slug" ] && project_slug=$(basename "$cwd" 2>/dev/null)

  # Windows 路径反斜杠转义
  local safe_cwd="${cwd//\\/\\\\}"
  local safe_transcript_path="${transcript_path//\\/\\\\}"

  _call_backend "/api/session/register" "{
    \"session_id\":\"$session_id\",
    \"machine_id\":\"$machine_id\",
    \"os\":\"$os_type\",
    \"project_slug\":\"$project_slug\",
    \"project_cwd\":\"$safe_cwd\",
    \"transcript_path\":\"$safe_transcript_path\",
    \"model\":\"$model\",
    \"source\":\"$source_type\"
  }" > /dev/null 2>&1

  log "INFO" "Session registered to backend: $session_id"
}

register_session

hook_output 0 '{}'
