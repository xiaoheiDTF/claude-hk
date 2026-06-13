#!/bin/bash
# ui-state-definition on_load: 加载 UI 设计规范 + 03-user-prompt-submit/*.md 补充上下文
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="003-6-1-ui-state-definition"
source "$PROJECT_DIR/.claude/skills/log.sh"

skill_log "INFO" "UI 设计规范 + 状态定义已加载"

SKILL_DIR="$PROJECT_DIR/.claude/skills/${SKILL_TAG}"
PROMPT_DIR="$SKILL_DIR/03-user-prompt-submit"

# 从 03-user-prompt-submit/ 目录加载补充上下文（MD 文件）
# 当技能在实际使用中发现问题时，将修正/补充写入此目录的 MD 文件
# 每次调用技能时自动加载，实现学习进化
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
