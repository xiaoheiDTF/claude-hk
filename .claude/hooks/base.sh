#!/bin/bash
# Hooks Base Script
# 1. 平台感知 (platform.sh)
# 2. 从 stdin 读取 JSON
# 3. 解析字段
# 4. 按日期写入日志，带 [EventName] 前缀
# 5. 输出结果 JSON

HOOKS_DIR="$CLAUDE_PROJECT_DIR/.claude/hooks"
LOG_DIR="$HOOKS_DIR/logs"
mkdir -p "$LOG_DIR"

source "$HOOKS_DIR/platform.sh"

HOOK_INPUT=$(cat)
LOG_FILE="$LOG_DIR/$(date +%Y-%m-%d).log"

_log_raw() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$1] $2" >> "$LOG_FILE"
}

json_get() {
  local key="$1"
  if [ -z "$HOOK_INPUT" ]; then echo ""; return; fi
  if command -v jq &>/dev/null; then
    printf '%s' "$HOOK_INPUT" | jq -r "$key" 2>/dev/null
    return
  fi
  if [ -n "$PYTHON_CMD" ]; then
    printf '%s' "$HOOK_INPUT" | "$PYTHON_CMD" "$HOOKS_DIR/json_get.py" "$key" 2>>"$LOG_FILE"
    return
  fi
  # 纯 sed fallback：提取 .key 或 ."key" 格式的顶层字段
  local bare_key="${key#.}"
  printf '%s' "$HOOK_INPUT" | sed -n "s/.*\"${bare_key}\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/p" | head -1
}

HOOK_EVENT=$(json_get '.hook_event_name')
[ -z "$HOOK_EVENT" ] && HOOK_EVENT="UNKNOWN"

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$1] [$HOOK_EVENT] $2" >> "$LOG_FILE"
}

log "DEBUG" "os=$OS_TYPE python=$PYTHON_CMD"
log "INFO" "Input: $HOOK_INPUT"

hook_output() {
  local exit_code="${1:-0}"
  local json="${2:-{}}"
  log "INFO" "Output (exit=$exit_code): $json"
  echo "$json"
  exit "$exit_code"
}
