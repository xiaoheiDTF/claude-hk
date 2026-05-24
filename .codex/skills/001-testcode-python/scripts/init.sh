#!/bin/bash
# 001-testcode-python init: 检�?pip 可用�?
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="001-testcode-python"
source "$PROJECT_DIR/\.codex/skills/log.sh"

# 确保 Python 可用
source "$PROJECT_DIR/\.codex/scripts/ensure_python.sh"
PYTHON_CMD=$(ensure_python)
if [ -z "$PYTHON_CMD" ]; then
  skill_log "WARN" "[init] Python 不可用，跳过 pip 检�?
  exit 0
fi

# 检�?pip
if "$PYTHON_CMD" -m pip --version &>/dev/null; then
  skill_log "INFO" "[init] pip 已就�?
else
  skill_log "WARN" "[init] pip 不可用，部分功能受限"
fi
