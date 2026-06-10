#!/bin/bash
# skill-register: 扫描 skills 目录，将新增的 skill 自动注册到 registry.conf
# 被 29-session-end/base.sh 调用

PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILLS_DIR="$PROJECT_DIR/.claude/skills"
REGISTRY="$SKILLS_DIR/registry.conf"

[ -d "$SKILLS_DIR" ] || exit 0

# 确保 registry.conf 存在
touch "$REGISTRY"

# 扫描所有包含 SKILL.md 的目录
CHANGED=0
for skill_dir in "$SKILLS_DIR"/*/; do
  [ -d "$skill_dir" ] || continue
  SKILL_NAME=$(basename "$skill_dir")
  [ -f "$skill_dir/SKILL.md" ] || continue

  # 检查是否已在 registry 中
  if ! grep -qx "$SKILL_NAME" "$REGISTRY" 2>/dev/null; then
    echo "$SKILL_NAME" >> "$REGISTRY"
    CHANGED=1
  fi
done

# 输出变更信息供 base.sh 日志使用
[ "$CHANGED" -eq 1 ] && echo "UPDATED"
