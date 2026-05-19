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

# ---- 1. 创建项目目录结构 ----
mkdir -p "$PROJECT_DIR/doc/testcode/python/api"
mkdir -p "$PROJECT_DIR/doc/testcode/python/other"
mkdir -p "$PROJECT_DIR/doc/otherDoc/$(date +%Y-%m-%d)"
init_log "doc 目录已就绪"

# ---- 2. 平台检测 ----
source "$CLAUSE_DIR/hooks/platform.sh"

# ---- 3. 配置 Python 环境 ----
LOCAL_LANG_DIR="$CLAUSE_DIR/localLanguage"

setup_python() {
  # 已有可用 Python → 跳过
  if [ -n "$PYTHON_CMD" ]; then
    init_log "Python 已就绪: $PYTHON_CMD"
    return 0
  fi

  # Windows: 下载嵌入版到 localLanguage/python/
  if [ "$OS_TYPE" = "windows" ]; then
    local python_dir="$LOCAL_LANG_DIR/python"
    local python_url="https://www.python.org/ftp/python/3.13.9/python-3.13.9-embed-amd64.zip"
    init_log "下载 Python 嵌入版到 localLanguage/python/..."
    mkdir -p "$python_dir"
    if curl -L -o "$python_dir/python-embed.zip" "$python_url" 2>>"$LOG_FILE"; then
      cd "$python_dir" && unzip -o python-embed.zip >>"$LOG_FILE" 2>&1 && rm -f python-embed.zip
      PYTHON_CMD="$python_dir/python.exe"
      init_log "Python 嵌入版安装完成: $PYTHON_CMD"
      return 0
    else
      init_log "Python 下载失败"
      return 1
    fi
  fi

  # Linux/macOS: 提示安装
  init_log "未找到 Python，请安装 python3"
  init_log "  macOS: xcode-select --install 或 brew install python3"
  init_log "  Linux: sudo apt install python3"
  return 1
}

setup_python

# ---- 4. 写入标记文件 ----
echo "{\"os\":\"$OS_TYPE\",\"python\":\"$PYTHON_CMD\",\"initialized_at\":\"$(date +%Y-%m-%dT%H:%M:%S)\"}" > "$INIT_MARKER"
init_log "初始化完成"

exit 0
