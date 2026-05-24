#!/bin/bash
# skills 统一日志模块
# 写入两处: hooks/logs (统一) + skills/log (按模�?
# 用法: 先设 SKILL_TAG，再 source 此文�?#   SKILL_TAG="testcode-python"
#   source "$CLAUDE_PROJECT_DIR/\.codex/skills/log.sh"
#   skill_log "INFO" "内容"

PROJECT_DIR="$CLAUDE_PROJECT_DIR"

# 统一日志: \.codex/hooks/logs/<日期>.log
UNIFIED_LOG_DIR="$PROJECT_DIR/\.codex/hooks/logs"
mkdir -p "$UNIFIED_LOG_DIR"
UNIFIED_LOG_FILE="$UNIFIED_LOG_DIR/$(date +%Y-%m-%d).log"

# 模块日志: \.codex/skills/log/<skill-tag>/<日期>.log
MODULE_LOG_DIR="$PROJECT_DIR/\.codex/skills/log/$SKILL_TAG"
mkdir -p "$MODULE_LOG_DIR"
MODULE_LOG_FILE="$MODULE_LOG_DIR/$(date +%Y-%m-%d).log"

skill_log() {
  local level="$1"
  shift
  local line="[$(date '+%Y-%m-%d %H:%M:%S')] [$level] [$SKILL_TAG] $*"
  # 写入统一日志
  echo "$line" >> "$UNIFIED_LOG_FILE"
  # 写入模块日志
  echo "$line" >> "$MODULE_LOG_FILE"
}
