#!/bin/bash
# 001-testcode-python init_check: 检�?Python 可用
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="001-testcode-python"
source "$PROJECT_DIR/\.codex/skills/log.sh"

source "$PROJECT_DIR/\.codex/scripts/ensure_python.sh"
PYTHON_CMD=$(ensure_python)
if [ -n "$PYTHON_CMD" ]; then
  skill_log "INFO" "[check] Python 可用: $PYTHON_CMD"
  exit 0
else
  skill_log "WARN" "[check] Python 不可�?
  exit 1
fi
