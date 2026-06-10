#!/bin/bash
# learn on_load: 加载学习进化技能上下文 + 03-user-prompt-submit/*.md 补充上下文
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="999-other-120-learn"
source "$PROJECT_DIR/.claude/skills/log.sh"

skill_log "INFO" "学习进化技能已加载"

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
