#!/bin/bash
# 001-1-issue-init 首次初始化：创建标签体系
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="001-1-issue-init"
source "$PROJECT_DIR/.claude/skills/log.sh"

LABELS_CONF="$PROJECT_DIR/.claude/skills/001-1-issue-init/labels.conf"
INIT_MARKER="$PROJECT_DIR/.github/.issue-initialized"

# 已初始化 → 跳过
if [ -f "$INIT_MARKER" ]; then
  skill_log "INFO" "[init] 标签体系已初始化，跳过"
  exit 0
fi

# 检查 gh CLI
if ! command -v gh &>/dev/null; then
  skill_log "WARN" "[init] gh CLI 不可用"
  exit 1
fi

# 检查 labels.conf
if [ ! -f "$LABELS_CONF" ]; then
  skill_log "ERROR" "[init] labels.conf 不存在"
  exit 1
fi

# 逐行读取标签定义并创建
created=0
skipped=0
while IFS='|' read -r name color desc; do
  # 跳过空行和注释
  [[ -z "$name" || "$name" == \#* ]] && continue
  # 去除前后空格
  name=$(echo "$name" | xargs)
  color=$(echo "$color" | xargs)
  desc=$(echo "$desc" | xargs)

  if gh label create "$name" --color "$color" --description "$desc" --force &>/dev/null; then
    created=$((created + 1))
    skill_log "INFO" "[init] 标签已创建: $name"
  else
    skipped=$((skipped + 1))
    skill_log "WARN" "[init] 标签创建失败或已存在: $name"
  fi
done < "$LABELS_CONF"

# 创建标记文件
mkdir -p "$PROJECT_DIR/.github"
echo "{\"initialized_at\":\"$(date +%Y-%m-%dT%H:%M:%S)\",\"labels_created\":$created,\"labels_skipped\":$skipped}" > "$INIT_MARKER"
skill_log "INFO" "[init] 标签体系初始化完成: created=$created skipped=$skipped"
