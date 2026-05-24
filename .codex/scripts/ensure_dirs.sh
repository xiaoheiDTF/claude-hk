#!/bin/bash
# 读取 dirs.conf，确保所有配置目录存�?
# 用法: source 此文件后调用 ensure_all_dirs �?ensure_skill_dirs <skill�?

CLAUSE_DIR="$CLAUDE_PROJECT_DIR/\.codex"
DIRS_CONF="$CLAUSE_DIR/dirs.conf"

# 读取配置并创建所有缺失目�?
ensure_all_dirs() {
  local log_file="${1:-/dev/null}"
  local created=0
  local skipped=0

  while IFS= read -r line; do
    # 跳过空行和注�?
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    # 取第一列（路径�?
    local dir=$(echo "$line" | awk '{print $1}')
    [ -z "$dir" ] && continue

    local full_path="$CLAUDE_PROJECT_DIR/$dir"
    if [ -d "$full_path" ]; then
      skipped=$((skipped + 1))
    else
      mkdir -p "$full_path" 2>>"$log_file"
      created=$((created + 1))
      [ -n "$log_file" ] && [ "$log_file" != "/dev/null" ] && \
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] [DIRS] 创建: $dir" >> "$log_file"
    fi
  done < "$DIRS_CONF"

  echo "{\"total\":$(wc -l < "$DIRS_CONF"),\"created\":$created,\"skipped\":$skipped}"
}

# 检查单�?skill 相关目录是否存在
ensure_skill_dirs() {
  local skill_keyword="$1"
  while IFS= read -r line; do
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    local dir=$(echo "$line" | awk '{print $1}')
    [[ "$dir" == *"$skill_keyword"* ]] || continue
    mkdir -p "$CLAUDE_PROJECT_DIR/$dir"
  done < "$DIRS_CONF"
}

# 报告目录状态（不创建，仅检查）
check_dirs() {
  local missing=0
  while IFS= read -r line; do
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    local dir=$(echo "$line" | awk '{print $1}')
    [ -z "$dir" ] && continue
    if [ ! -d "$CLAUDE_PROJECT_DIR/$dir" ]; then
      echo "MISSING: $dir"
      missing=$((missing + 1))
    fi
  done < "$DIRS_CONF"
  [ "$missing" -eq 0 ] && echo "ALL OK"
  return $missing
}
