#!/bin/bash
# Python 保障模块 �?确保 Python 环境可用
# �?init.sh �?01-session-start 调用
#
# 策略:
#   1. 项目嵌入版（最优先，不依赖系统�?
#   2. 系统 Python（用户已安装�?
#   3. 自动下载（兜底，多源 + 重试�?
#
# 状态文�? \.codex/.python-state（纯文本，无需 Python 解析�?
#   status=not_tried|downloading|ready|failed|failed_permanent
#   path=...
#   attempts=N
#   last_attempt=YYYY-MM-DDTHH:MM:SS
#   version=...

PROJECT_DIR="$CLAUDE_PROJECT_DIR"
CLAUSE_DIR="$PROJECT_DIR/\.codex"
LOCAL_LANG_DIR="$CLAUSE_DIR/localLanguage"
STATE_FILE="$CLAUSE_DIR/.python-state"
MAX_ATTEMPTS=5

# ---- 平台检�?----
_detect_os() {
  case "$(uname -s)" in
    Linux*)   echo "linux" ;;
    Darwin*)  echo "macos" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *)        echo "unknown" ;;
  esac
}

# ---- 状态文件操作（�?bash�?----
_state_get() {
  local key="$1"
  [ -f "$STATE_FILE" ] && grep "^${key}=" "$STATE_FILE" | tail -1 | cut -d'=' -f2-
}

_state_set() {
  local key="$1"
  local val="$2"
  if [ -f "$STATE_FILE" ] && grep -q "^${key}=" "$STATE_FILE"; then
    local tmp="${STATE_FILE}.tmp"
    grep -v "^${key}=" "$STATE_FILE" > "$tmp"
    echo "${key}=${val}" >> "$tmp"
    mv "$tmp" "$STATE_FILE"
  else
    echo "${key}=${val}" >> "$STATE_FILE"
  fi
}

_state_init() {
  mkdir -p "$(dirname "$STATE_FILE")"
  [ -f "$STATE_FILE" ] || cat > "$STATE_FILE" << 'EOF'
status=not_tried
path=
attempts=0
last_attempt=
version=
EOF
}

# ---- Python 路径解析 ----
_resolve_embed() {
  local os=$(_detect_os)
  case "$os" in
    windows) echo "$LOCAL_LANG_DIR/python/python.exe" ;;
    linux)   echo "$LOCAL_LANG_DIR/python/bin/python3" ;;
    macos)   echo "$LOCAL_LANG_DIR/python/bin/python3" ;;
  esac
}

_resolve_python() {
  # 1. 嵌入�?
  local embed=$(_resolve_embed)
  if [ -x "$embed" ]; then
    echo "$embed"
    return
  fi
  # 2. 系统 Python
  for cmd in python3 python; do
    if command -v "$cmd" &>/dev/null && "$cmd" -c "pass" 2>/dev/null; then
      echo "$cmd"
      return
    fi
  done
  echo ""
}

# ---- 下载源配�?----
_get_download_urls() {
  local os=$(_detect_os)
  case "$os" in
    windows)
      # Python 嵌入版（多镜像源�?
      echo "https://www.python.org/ftp/python/3.13.9/python-3.13.9-embed-amd64.zip"
      echo "https://registry.npmmirror.com/-/binary/python/3.13.9/python-3.13.9-embed-amd64.zip"
      ;;
    linux)
      # python-build-standalone (x86_64)
      local ver="20250417"
      local tag="cpython-3.13.3+${ver}-x86_64-unknown-linux-gnu-install_only.tar.gz"
      echo "https://github.com/indygreg/python-build-standalone/releases/download/${ver}/${tag}"
      ;;
    macos)
      # python-build-standalone (arm64 + x86_64)
      local ver="20250417"
      local arch=$(uname -m)
      local platform="aarch64-apple-darwin"
      [ "$arch" = "x86_64" ] && platform="x86_64-apple-darwin"
      local tag="cpython-3.13.3+${ver}-${platform}-install_only.tar.gz"
      echo "https://github.com/indygreg/python-build-standalone/releases/download/${ver}/${tag}"
      ;;
  esac
}

# ---- 下载并安�?----
_download_python() {
  local python_dir="$LOCAL_LANG_DIR/python"
  mkdir -p "$python_dir"

  _state_set status "downloading"
  _state_set last_attempt "$(date +%Y-%m-%dT%H:%M:%S)"

  local os=$(_detect_os)
  local urls
  urls=$(_get_download_urls)

  while IFS= read -r url; do
    [ -z "$url" ] && continue
    local filename=$(basename "$url")
    local tmpfile="$python_dir/$filename"

    echo "[ensure_python] 尝试下载: $url"

    if curl -L --connect-timeout 15 --max-time 300 -o "$tmpfile" "$url" 2>/dev/null; then
      # 验证下载文件不为�?
      if [ -s "$tmpfile" ]; then
        echo "[ensure_python] 下载成功，开始安�?.."

        case "$os" in
          windows)
            cd "$python_dir" && unzip -o "$filename" 2>/dev/null && rm -f "$filename"
            # 修复 pip：移�?import 限制
            local pth_file="$python_dir/python313._pth"
            [ -f "$pth_file" ] && echo "import site" >> "$pth_file"
            ;;
          linux|macos)
            cd "$python_dir" && tar xzf "$filename" --strip-components=1 2>/dev/null && rm -f "$filename"
            ;;
        esac

        # 验证安装结果
        local embed=$(_resolve_embed)
        if [ -x "$embed" ]; then
          local ver=$("$embed" --version 2>&1 | head -1)
          _state_set status "ready"
          _state_set path "$embed"
          _state_set version "$ver"
          echo "[ensure_python] 安装成功: $embed ($ver)"
          return 0
        else
          echo "[ensure_python] 安装后验证失�?
          rm -f "$tmpfile"
        fi
      else
        echo "[ensure_python] 下载文件为空"
        rm -f "$tmpfile"
      fi
    else
      echo "[ensure_python] 下载失败: $url"
      rm -f "$tmpfile"
    fi
  done <<< "$urls"

  return 1
}

# ---- 主入�?----
ensure_python() {
  _state_init

  # 先尝试解�?
  local found=$(_resolve_python)
  if [ -n "$found" ]; then
    # 更新状态（可能之前是系�?Python，不是嵌入版�?
    [ "$(_state_get status)" != "ready" ] && {
      _state_set status "ready"
      _state_set path "$found"
      local ver=$("$found" --version 2>&1 | head -1)
      _state_set version "$ver"
    }
    echo "$found"
    return 0
  fi

  # 未找�?�?检查是否应该重�?
  local status=$(_state_get status)
  local attempts=$(_state_get attempts)
  attempts=${attempts:-0}

  case "$status" in
    ready)
      # 状态说�?ready 但实际找不到 �?重置
      _state_set status "not_tried"
      _state_set attempts "0"
      ;;
    failed_permanent)
      echo "[ensure_python] 已达最大重试次数，请手动安�?Python"
      echo ""
      return 1
      ;;
    downloading)
      # 可能上次中断�?�?允许重试
      ;;
  esac

  # 超过最大次�?
  if [ "$attempts" -ge "$MAX_ATTEMPTS" ]; then
    _state_set status "failed_permanent"
    echo "[ensure_python] 已尝�?$attempts 次，停止重试"
    echo "[ensure_python] 请手动安�?Python 并重�?Claude Code"
    echo ""
    return 1
  fi

  # 尝试下载
  _state_set attempts "$((attempts + 1))"
  echo "[ensure_python] Python 未找到，尝试自动下载 (�?$((attempts + 1))/$MAX_ATTEMPTS �?..."

  if _download_python; then
    _resolve_python
    return 0
  else
    _state_set status "failed"
    echo "[ensure_python] 本次下载失败，下次会话将重试"
    echo ""
    return 1
  fi
}
