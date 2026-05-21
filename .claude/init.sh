#!/bin/bash
# 统一初始化脚本 - 管理项目首次运行的所有配置

PROJECT_DIR="$CLAUDE_PROJECT_DIR"
CLAUSE_DIR="$PROJECT_DIR/.claude"
LOG_DIR="$CLAUSE_DIR/hooks/logs"
mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/$(date +%Y-%m-%d).log"
INIT_MARKER="$CLAUSE_DIR/.initialized"

init_log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] [INIT] $*" >> "$LOG_FILE"; }

# 已初始化 → 跳过
[ -f "$INIT_MARKER" ] && exit 0

init_log "首次运行，开始初始化..."

# ---- 1. 从配置文件创建项目目录 ----
source "$CLAUSE_DIR/scripts/ensure_dirs.sh"
dir_result=$(ensure_all_dirs "$LOG_FILE")
# otherDoc 额外创建今日子目录
mkdir -p "$PROJECT_DIR/doc/otherDoc/$(date +%Y-%m-%d)"
init_log "目录初始化完成: $dir_result"

# ---- 2. 平台检测 ----
source "$CLAUSE_DIR/hooks/platform.sh"

# ---- 3. 配置 Python 环境（使用 ensure_python 模块） ----
source "$CLAUSE_DIR/scripts/ensure_python.sh"
PYTHON_CMD=$(ensure_python)
if [ -n "$PYTHON_CMD" ]; then
  init_log "Python 已就绪: $PYTHON_CMD"
else
  init_log "WARN" "Python 安装未成功，将在下次会话重试"
fi

# ---- 4. 配置 UTF-8 编码（跨平台） ----
setup_utf8() {
  local bashrc="$HOME/.bashrc"
  local marker="# >>> claude-hk-utf8 >>>"
  local ender="# <<< claude-hk-utf8 <<<"
  local block=""

  case "$OS_TYPE" in
    windows)
      block="${marker}
export LANG=zh_CN.UTF-8
export LC_ALL=zh_CN.UTF-8
chcp.com 65001 > /dev/null 2>&1
${ender}"
      ;;
    linux)
      block="${marker}
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8
${ender}"
      ;;
    macos)
      block="${marker}
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8
${ender}"
      ;;
  esac

  # 检查是否已有配置：标记块 或 关键内容
  if [ -f "$bashrc" ]; then
    if grep -q "$marker" "$bashrc" 2>/dev/null; then
      init_log "UTF-8 标记块已存在，跳过"
      return 0
    fi
    if grep -q "chcp.com 65001\|LANG=.*UTF-8" "$bashrc" 2>/dev/null; then
      init_log "UTF-8 配置已存在（无标记），补充写入标记块"
      # 不重复写，只补标记方便下次判断
      sed -i "1i $marker" "$bashrc"
      echo "$ender" >> "$bashrc"
      return 0
    fi
  fi

  echo "" >> "$bashrc"
  echo "$block" >> "$bashrc"
  init_log "UTF-8 编码配置已写入 $bashrc (OS=$OS_TYPE)"

  if [ "$OS_TYPE" = "windows" ]; then
    setx LANG "zh_CN.UTF-8" >> "$LOG_FILE" 2>&1
    setx LC_ALL "zh_CN.UTF-8" >> "$LOG_FILE" 2>&1
    init_log "Windows 系统环境变量 LANG/LC_ALL 已设置"
  fi
}

setup_utf8

# ---- 5. Skill 级首次初始化 ----
SKILLS_DIR="$CLAUSE_DIR/skills"
if [ -d "$SKILLS_DIR" ]; then
  for skill_init in "$SKILLS_DIR"/*/scripts/init.sh; do
    [ -f "$skill_init" ] || continue
    skill_name=$(basename "$(dirname "$(dirname "$skill_init")")")
    init_log "Running init for skill: $skill_name"
    if bash "$skill_init" >> "$LOG_FILE" 2>&1; then
      init_log "Skill init OK: $skill_name"
    else
      init_log "WARN: Skill init failed: $skill_name (non-blocking)"
    fi
  done
fi

# ---- 6. 写入标记文件 ----
echo "{\"os\":\"$OS_TYPE\",\"python\":\"$PYTHON_CMD\",\"utf8\":\"true\",\"initialized_at\":\"$(date +%Y-%m-%dT%H:%M:%S)\"}" > "$INIT_MARKER"
init_log "初始化完成"

exit 0
