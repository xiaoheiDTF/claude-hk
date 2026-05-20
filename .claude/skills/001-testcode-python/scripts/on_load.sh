#!/bin/bash
# testcode-python skill 加载时触发
TODAY=$(date +%Y-%m-%d)
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
PYTHON_PATH="$PROJECT_DIR/.claude/localLanguage/python/python.exe"

source "$PROJECT_DIR/.claude/scripts/ensure_dirs.sh"
ensure_skill_dirs "testcode"

cat <<EOF
{"hookSpecificOutput":{"hookEventName":"InstructionsLoaded","additionalContext":"[testcode-python] 日期: $TODAY\n脚本目录: $PROJECT_DIR/doc/testcode/python/\n  - api/  → API自动化测试\n  - other/ → 其他脚本\nPython路径: $PYTHON_PATH"}}
EOF
