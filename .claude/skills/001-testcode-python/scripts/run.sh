#!/bin/bash
# 运行 doc/testcode/python 下的脚本，优先使用项目自带 Python
PROJECT_DIR="${CLAUDE_PROJECT_DIR:-$(cd "$(dirname "$0")/../../.." && pwd)}"
EMBED_PYTHON="$PROJECT_DIR/.claude/localLanguage/python/python.exe"
SCRIPT_PATH="$1"

if [ -z "$SCRIPT_PATH" ]; then
  echo "用法: run.sh <doc/testcode/python/下的脚本路径>"
  exit 1
fi

if [ -x "$EMBED_PYTHON" ]; then
  "$EMBED_PYTHON" "$PROJECT_DIR/$SCRIPT_PATH"
else
  python "$PROJECT_DIR/$SCRIPT_PATH"
fi
