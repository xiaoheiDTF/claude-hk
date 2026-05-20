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

# 2. Python 可用性
ensure_python() {
  source "$CLAUDE_PROJECT_DIR/.claude/hooks/platform.sh"
  if [ -n "$PYTHON_CMD" ]; then
    log "INFO" "Python 已就绪: $PYTHON_CMD"
  else
    log "WARN" "Python 未找到，部分 skill 可能不可用"
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
ensure_python
ensure_dirs

hook_output 0 '{}'
