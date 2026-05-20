#!/bin/bash
# 运行 doc/testcode/python 下的脚本
# Python 优先级: 系统 → 项目自带 → init.sh 自动下载
PROJECT_DIR="${CLAUDE_PROJECT_DIR:-$(cd "$(dirname "$0")/../../.." && pwd)}"
EMBED_PYTHON="$PROJECT_DIR/.claude/localLanguage/python/python.exe"
SCRIPT_PATH="$1"

SKILL_TAG="testcode-python:run"
source "$PROJECT_DIR/.claude/skills/log.sh"

if [ -z "$SCRIPT_PATH" ]; then
  echo "用法: run.sh <doc/testcode/python/下的脚本路径>"
  skill_log "ERROR" "未指定脚本路径"
  exit 1
fi

resolve_python() {
  if command -v python &>/dev/null; then
    skill_log "INFO" "使用系统 Python"
    echo "python"
    return
  fi
  if [ -x "$EMBED_PYTHON" ]; then
    skill_log "INFO" "使用项目自带 Python: $EMBED_PYTHON"
    echo "$EMBED_PYTHON"
    return
  fi
  skill_log "WARN" "未找到 Python，执行 init.sh 自动下载..."
  bash "$PROJECT_DIR/.claude/init.sh" >&2
  if [ -x "$EMBED_PYTHON" ]; then
    skill_log "INFO" "下载完成，使用: $EMBED_PYTHON"
    echo "$EMBED_PYTHON"
    return
  fi
  skill_log "ERROR" "Python 下载失败"
  echo ""
}

PYTHON_CMD=$(resolve_python)
if [ -z "$PYTHON_CMD" ]; then
  echo "错误: Python 不可用且自动下载失败" >&2
  exit 1
fi

skill_log "INFO" "执行: $PYTHON_CMD $PROJECT_DIR/$SCRIPT_PATH"
"$PYTHON_CMD" "$PROJECT_DIR/$SCRIPT_PATH"
skill_log "INFO" "执行完成: $SCRIPT_PATH (退出码: $?)"
