#!/bin/bash
# 888-2-1-frontend-project-init on_load: 注入前端项目初始化提示上下文
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="888-2-1-frontend-project-init"
source "$PROJECT_DIR/.claude/skills/log.sh"

skill_log "INFO" "前端项目初始化 skill 已加载"

SKILL_DIR="$PROJECT_DIR/.claude/skills/${SKILL_TAG}"
PROMPT_DIR="$SKILL_DIR/03-user-prompt-submit"

# 从 03-user-prompt-submit/ 目录加载补充上下文（MD 文件）
if [ -d "$PROMPT_DIR" ]; then
  export _SKILL_PROMPT_DIR="$PROMPT_DIR"
  python3 -c "
import glob, json, os
parts = []
d = os.environ.get('_SKILL_PROMPT_DIR', '').replace(os.sep, '/')
if d and os.path.isdir(d):
    for f in sorted(glob.glob(d + '/*.md')):
        try:
            with open(f, encoding='utf-8') as fh:
                parts.append(fh.read())
        except: pass
context = chr(10).join(parts) if parts else ''
print(json.dumps({'hookSpecificOutput': {'hookEventName': 'UserPromptSubmit', 'additionalContext': context}}))
" 2>/dev/null || echo '{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":""}}'
else
  echo '{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":""}}'
fi
