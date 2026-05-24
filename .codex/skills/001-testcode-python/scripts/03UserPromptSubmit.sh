#!/bin/bash
# testcode-python on_load: skill 加载时注入动态上下文
# 日志写文件，stdout 输出 JSON �?hook 解析
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="001-testcode-python"
source "$PROJECT_DIR/\.codex/skills/log.sh"

TODAY=$(date +%Y-%m-%d)
EMBED_PYTHON="$PROJECT_DIR/\.codex/localLanguage/python/python.exe"

source "$PROJECT_DIR/\.codex/scripts/ensure_dirs.sh"
ensure_skill_dirs "testcode"

if command -v python &>/dev/null; then
  PYTHON_INFO="Python: 系统 python ($(python --version 2>&1))"
  skill_log "INFO" "Python 来源: 系统"
elif [ -x "$EMBED_PYTHON" ]; then
  PYTHON_INFO="Python: $EMBED_PYTHON (项目自带)"
  skill_log "INFO" "Python 来源: 项目自带"
else
  PYTHON_INFO="Python: 未找到（调用时将自动下载�?
  skill_log "WARN" "Python 未找到，将在 run.sh 中触�?init.sh 下载"
fi

skill_log "INFO" "日期: $TODAY"
skill_log "INFO" "脚本目录: $PROJECT_DIR/doc/testcode/python/"

CONTEXT="[testcode-python] 日期: $TODAY\n脚本目录: $PROJECT_DIR/doc/testcode/python/\n  - api/  �?API自动化测试\n  - other/ �?其他脚本\n$PYTHON_INFO"

echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"$CONTEXT\"}}"
