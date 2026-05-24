#!/bin/bash
# 01-session-start: 会话开始或恢复时触�?(每个会话一�?

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

source_type=$(json_get '.source')
log "INFO" "source=$source_type"

# 首次运行 �?触发统一初始�?
[ ! -f "$CLAUDE_PROJECT_DIR/\.codex/.initialized" ] && bash "$CLAUDE_PROJECT_DIR/\.codex/init.sh"

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

# 2. Python 可用性（巡检 + 自动修复�?
ensure_python_check() {
  source "$CLAUDE_PROJECT_DIR/\.codex/scripts/ensure_python.sh"
  local result
  result=$(ensure_python)
  if [ -n "$result" ]; then
    log "INFO" "Python 已就�? $result"
  else
    log "WARN" "Python 不可用，部分功能受限"
  fi
}

# 3. 目录完整�?
ensure_dirs() {
  source "$CLAUDE_PROJECT_DIR/\.codex/scripts/ensure_dirs.sh"
  local missing
  missing=$(check_dirs 2>&1)
  if [ "$missing" = "ALL OK" ]; then
    log "INFO" "目录检查通过"
  else
    log "WARN" "目录缺失，重新创�? $missing"
    ensure_all_dirs "$LOG_FILE" > /dev/null
  fi
}

ensure_utf8
ensure_python_check
ensure_dirs

# 4. Skill 级巡检
ensure_skill_checks() {
  local skills_dir="$CLAUDE_PROJECT_DIR/\.codex/skills"
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

hook_output 0 '{}'
