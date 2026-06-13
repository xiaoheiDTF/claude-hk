#!/bin/bash
# _load_supplementary.sh - 共享助手：加载 03-user-prompt-submit/*.md 补充上下文
# 用法:
#   source "$PROJECT_DIR/.claude/skills/_load_supplementary.sh"
#   _load_supplementary "/path/to/skill/dir"
# 结果:
#   $SUPPLEMENTARY_JSON — JSON 安全转义后的字符串（用于 additionalContext）
#   $SUPPLEMENTARY_TEXT — 原始文本（用于 echo 输出）

_load_supplementary() {
  local skill_dir="$1"
  local prompt_dir="$skill_dir/03-user-prompt-submit"

  SUPPLEMENTARY_JSON=""
  SUPPLEMENTARY_TEXT=""

  [ -d "$prompt_dir" ] || return 0

  # 检查是否有 MD 文件
  local md_count=0
  for f in "$prompt_dir"/*.md; do
    [ -f "$f" ] || continue
    md_count=$((md_count + 1))
  done
  [ "$md_count" -gt 0 ] || return 0

  # 用 Python 做 JSON 安全转义
  export _SKILL_PROMPT_DIR="$prompt_dir"
  SUPPLEMENTARY_JSON=$(python3 -c "
import glob, json, os
parts = []
d = os.environ.get('_SKILL_PROMPT_DIR', '').replace(os.sep, '/')
if d and os.path.isdir(d):
    for f in sorted(glob.glob(d + '/*.md')):
        try:
            with open(f, encoding='utf-8') as fh:
                parts.append(fh.read())
        except: pass
text = chr(10).join(parts) if parts else ''
print(json.dumps(text)[1:-1] if text else '')
" 2>/dev/null || echo "")

  # 同时提供原始文本版本（用于 TEXT 模式脚本）
  SUPPLEMENTARY_TEXT=""
  for f in "$prompt_dir"/*.md; do
    [ -f "$f" ] || continue
    content=$(cat "$f" 2>/dev/null)
    if [ -n "$content" ]; then
      if [ -n "$SUPPLEMENTARY_TEXT" ]; then
        SUPPLEMENTARY_TEXT="${SUPPLEMENTARY_TEXT}

---

${content}"
      else
        SUPPLEMENTARY_TEXT="${content}"
      fi
    fi
  done
}
