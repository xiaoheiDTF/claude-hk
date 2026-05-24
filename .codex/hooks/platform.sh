#!/bin/bash
# 平台检测与 Python 路径解析

detect_os() {
  case "$(uname -s)" in
    Linux*)   echo "linux" ;;
    Darwin*)  echo "macos" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *)        echo "unknown" ;;
  esac
}

OS_TYPE=$(detect_os)

resolve_python() {
  local embed
  case "$OS_TYPE" in
    windows) embed="$CLAUDE_PROJECT_DIR/\.codex/localLanguage/python/python.exe" ;;
    *)       embed="$CLAUDE_PROJECT_DIR/\.codex/localLanguage/python/bin/python3" ;;
  esac
  if [ -x "$embed" ]; then echo "$embed"; return; fi
  for cmd in python3 python; do
    if command -v "$cmd" &>/dev/null && "$cmd" -c "pass" 2>/dev/null; then
      echo "$cmd"; return
    fi
  done
  echo ""
}

PYTHON_CMD=$(resolve_python)
