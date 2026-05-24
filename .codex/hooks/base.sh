#!/bin/bash
# Hooks Base Script
# 1. 平台感知 (platform.sh)
# 2. �?stdin 读取 JSON
# 3. 解析字段
# 4. 按日期写入日志，�?[EventName] 前缀
# 5. 输出结果 JSON

HOOKS_DIR="$CLAUDE_PROJECT_DIR/\.codex/hooks"
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
  # �?sed fallback：提�?.key �?."key" 格式的顶层字�?
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

# Skill 感知分发：根�?.active 中的 session_id 匹配活跃 skill�?
# 调用对应 skill 目录下的生命周期脚本�?
# 用法: dispatch_to_skill "05" [额外参数...]
# 返回: 0=成功或无活跃 skill，非 0=skill 脚本执行失败
# stdout: skill 脚本的输出（如有�?
dispatch_to_skill() {
  local event_num="$1"
  shift

  local skills_dir="$CLAUDE_PROJECT_DIR/\.codex/skills"
  local active_file="$skills_dir/.active"

  # �?.active 或为�?�?跳过
  [ -s "$active_file" ] || return 0

  # 获取当前 session_id
  local sid
  sid=$(json_get '.session_id')
  [ -z "$sid" ] && return 0

  # 查找当前 session 激活的 skill
  source "$skills_dir/active.sh"
  local skill_name
  skill_name=$(active_get "$sid")
  [ -z "$skill_name" ] && return 0

  # 按事件编号映射脚本名
  local script_name
  case "$event_num" in
    02) script_name="02Setup" ;;
    04) script_name="04UserPromptExpansion" ;;
    05) script_name="05PreToolUse" ;;
    06) script_name="06PermissionRequest" ;;
    07) script_name="07PermissionDenied" ;;
    08) script_name="08PostToolUse" ;;
    09) script_name="09PostToolUseFailure" ;;
    10) script_name="10PostToolBatch" ;;
    11) script_name="11Notification" ;;
    12) script_name="12SubagentStart" ;;
    13) script_name="13SubagentStop" ;;
    14) script_name="14TaskCreated" ;;
    15) script_name="15TaskCompleted" ;;
    17) script_name="17StopFailure" ;;
    18) script_name="18TeammateIdle" ;;
    19) script_name="19InstructionsLoaded" ;;
    20) script_name="20ConfigChange" ;;
    21) script_name="21CwdChanged" ;;
    22) script_name="22FileChanged" ;;
    23) script_name="23WorktreeCreate" ;;
    24) script_name="24WorktreeRemove" ;;
    25) script_name="25PreCompact" ;;
    26) script_name="26PostCompact" ;;
    27) script_name="27Elicitation" ;;
    28) script_name="28ElicitationResult" ;;
    29) script_name="29SessionEnd" ;;
    *) return 0 ;;
  esac

  local script_path="$skills_dir/$skill_name/scripts/$script_name.sh"
  [ -f "$script_path" ] || return 0

  log "INFO" "Dispatching to skill=$skill_name script=$script_name.sh"
  bash "$script_path" "$@" 2>>"$LOG_FILE"
}
