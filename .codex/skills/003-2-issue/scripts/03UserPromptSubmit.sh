#!/bin/bash
# 003-2-issue 上下文注�?
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="003-2-issue"
source "$PROJECT_DIR/\.codex/skills/log.sh"

echo "=== Issue 创建上下�?==="
echo "日期: $(date +%Y-%m-%d)"
echo "分支: $(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null)"

# 列出可用模板
templates_dir="$PROJECT_DIR/doc/issues/templates"
if [ -d "$templates_dir" ]; then
  echo "可用模板:"
  for tpl in "$templates_dir"/tpl_*.md; do
    [ -f "$tpl" ] || continue
    tpl_name=$(grep '^name:' "$tpl" 2>/dev/null | head -1 | sed 's/name: *//')
    [ -z "$tpl_name" ] && tpl_name=$(basename "$tpl")
    echo "  - $tpl_name ($(basename "$tpl"))"
  done
fi

# 列出未发布草�?
drafts_dir="$PROJECT_DIR/doc/issues/drafts"
if [ -d "$drafts_dir" ]; then
  draft_count=$(ls "$drafts_dir"/*.md 2>/dev/null | wc -l | tr -d ' ')
  echo "未发布草稿数: $draft_count"
fi

skill_log "INFO" "[inject] issue context injected"
