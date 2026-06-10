#!/bin/bash
# 888-2-2-frontend-modify on_load: 加载 project.md + 03-user-prompt-submit/*.md 补充上下文
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="888-2-2-frontend-modify"
source "$PROJECT_DIR/.claude/skills/log.sh"

skill_log "INFO" "前端代码修改规范已加载"

SKILL_DIR="$PROJECT_DIR/.claude/skills/${SKILL_TAG}"
PROMPT_DIR="$SKILL_DIR/03-user-prompt-submit"
export _PROJECT_MD="$SKILL_DIR/project.md"
export _SKILL_PROMPT_DIR="$PROMPT_DIR"

# 构建 additionalContext：project.md 内容 + 03-user-prompt-submit/*.md 内容
python3 -c "
import glob, json, os

parts = []

# 1. 读取 project.md（严格分隔，防止模型混淆）
project_md = os.environ.get('_PROJECT_MD', '').replace(os.sep, '/')
if project_md and os.path.isfile(project_md):
    try:
        with open(project_md, encoding='utf-8') as f:
            content = f.read().strip()
            if content:
                parts.append(
                    '# ===== project.md 开始 =====\n'
                    '# 以下是本项目的开发规范文档 project.md，所有代码修改必须严格遵循此文档。\n'
                    '# 注意：此文档之外的内容仅为补充上下文，不可替代或覆盖 project.md。\n'
                    '# =====\n'
                    + content + '\n'
                    '# ===== project.md 结束 ====='
                )
    except: pass

# 2. 读取 03-user-prompt-submit/*.md（补充上下文，非 project.md）
prompt_dir = os.environ.get('_SKILL_PROMPT_DIR', '').replace(os.sep, '/')
if prompt_dir and os.path.isdir(prompt_dir):
    md_parts = []
    for f in sorted(glob.glob(prompt_dir + '/*.md')):
        try:
            with open(f, encoding='utf-8') as fh:
                md_parts.append(fh.read())
        except: pass
    if md_parts:
        parts.append(
            '# ===== 补充上下文开始（非 project.md，仅供参考）=====\n'
            + chr(10).join(md_parts) + '\n'
            '# ===== 补充上下文结束 ====='
        )

context = chr(10).join(parts) if parts else ''
print(json.dumps({'hookSpecificOutput': {'hookEventName': 'UserPromptSubmit', 'additionalContext': context}}))
" 2>/dev/null || echo '{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":""}}'
