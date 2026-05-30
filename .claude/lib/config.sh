# .claude/lib/config.sh
# 从 ~/.claude-tap-plus/backend.json 读取后端/代理配置
# 各 hooks/skills 统一 source 此文件，避免重复解析逻辑

# 读取 backend.json，设置 BACKEND_URL 全局变量
# 返回: 0=成功, 1=文件不存在或解析失败
load_backend_config() {
  [ -n "$BACKEND_URL" ] && return 0

  local json_file="$HOME/.claude-tap-plus/backend.json"
  [ -f "$json_file" ] || return 1

  local host port
  host=$(grep -o '"host"[[:space:]]*:[[:space:]]*"[^"]*"' "$json_file" 2>/dev/null | head -1 | sed 's/.*: *"//;s/"//')
  port=$(grep -o '"port"[[:space:]]*:[[:space:]]*[0-9]*' "$json_file" 2>/dev/null | head -1 | sed 's/.*: *//')
  [ -z "$host" ] || [ -z "$port" ] && return 1

  BACKEND_URL="http://$host:$port"
}
