#!/bin/bash
# .active 管理模块 — 纯 bash，零外部依赖
# 格式: 每行一条 session_id|skill_name
# 不需要 Python、不需要 jq、不需要路径转换

PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILLS_DIR="$PROJECT_DIR/.claude/skills"
ACTIVE_FILE="$SKILLS_DIR/.active"
LOCK_FILE="$SKILLS_DIR/.active.lock"

source "$SKILLS_DIR/lock.sh"

_ensure_active() {
  [ -f "$ACTIVE_FILE" ] || touch "$ACTIVE_FILE"
}

# 添加条目（幂等，同 session_id 不重复）
active_add() {
  local sid="$1"
  local skill_name="$2"

  lock_acquire "$LOCK_FILE" 10 || return 1
  _ensure_active

  # 已存在则更新
  if grep -q "^${sid}|" "$ACTIVE_FILE" 2>/dev/null; then
    local tmp="${ACTIVE_FILE}.tmp"
    sed "s/^${sid}|.*/${sid}|${skill_name}/" "$ACTIVE_FILE" > "$tmp"
    mv "$tmp" "$ACTIVE_FILE"
  else
    echo "${sid}|${skill_name}" >> "$ACTIVE_FILE"
  fi

  lock_release
}

# 删除条目
active_remove() {
  local sid="$1"

  lock_acquire "$LOCK_FILE" 10 || return 1
  _ensure_active

  local tmp="${ACTIVE_FILE}.tmp"
  grep -v "^${sid}|" "$ACTIVE_FILE" > "$tmp" 2>/dev/null || true
  mv "$tmp" "$ACTIVE_FILE"

  lock_release
}

# 查询条目 → 输出 skill 名或空
active_get() {
  local sid="$1"
  [ -f "$ACTIVE_FILE" ] || return
  local line=$(grep "^${sid}|" "$ACTIVE_FILE" 2>/dev/null | head -1)
  [ -n "$line" ] && echo "$line" | cut -d'|' -f2
}

# 按 skill 名删除所有匹配条目
active_remove_by_skill() {
  local skill_name="$1"

  lock_acquire "$LOCK_FILE" 10 || return 1
  _ensure_active

  local tmp="${ACTIVE_FILE}.tmp"
  grep -v "|${skill_name}$" "$ACTIVE_FILE" > "$tmp" 2>/dev/null || true
  mv "$tmp" "$ACTIVE_FILE"

  lock_release
}

# 列出所有 skill 名（去重）
active_skills() {
  [ -f "$ACTIVE_FILE" ] || return
  cut -d'|' -f2 "$ACTIVE_FILE" | sort -u
}

# 列出全部
active_list() {
  _ensure_active
  cat "$ACTIVE_FILE"
}
