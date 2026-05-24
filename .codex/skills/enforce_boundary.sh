#!/bin/bash
# enforce_boundary.sh �?A 层：工具级白名单拦截
# 读取 .active 中当�?session �?skill，解�?SKILL.md frontmatter allowed-tools�?
# 比对当前 tool_name，不在白名单�?deny

enforce_boundary() {
  local skills_dir="$CLAUDE_PROJECT_DIR/\.codex/skills"
  local active_file="$skills_dir/.active"

  # �?.active 或为�?�?放行
  [ -s "$active_file" ] || return 0

  # 获取当前 session_id
  local sid
  sid=$(json_get '.session_id')
  [ -z "$sid" ] && return 0

  # 查找当前 session 激活的 skill
  source "$skills_dir/active.sh"
  local skill_name
  skill_name=$(active_get "$sid")
  [ -z "$skill_name" ] && return 0

  # 解析 SKILL.md frontmatter 中的 allowed-tools
  local skill_md="$skills_dir/$skill_name/SKILL.md"
  if [ ! -f "$skill_md" ]; then
    log "WARN" "enforce_boundary: SKILL.md not found for $skill_name, allowing"
    return 0
  fi

  # 提取 allowed-tools 列表（frontmatter �?"  - ToolName" 格式的行�?
  local allowed=""
  local in_frontmatter=0
  while IFS= read -r line; do
    if [ "$line" = "---" ]; then
      if [ "$in_frontmatter" -eq 0 ]; then
        in_frontmatter=1
        continue
      else
        break
      fi
    fi
    if [ "$in_frontmatter" -eq 1 ]; then
      if echo "$line" | grep -qE '^\s+-\s+\S+'; then
        local tool
        tool=$(echo "$line" | sed -E 's/^\s+-\s+//')
        allowed="$allowed $tool"
      fi
    fi
  done < "$skill_md"

  # 空白名单 �?无约束，放行
  if [ -z "$allowed" ]; then
    log "WARN" "enforce_boundary: no allowed-tools for $skill_name, allowing"
    return 0
  fi

  # 比对当前 tool_name
  local current_tool
  current_tool=$(json_get '.tool_name')
  [ -z "$current_tool" ] && return 0

  # 检查是否在白名单中
  for t in $allowed; do
    if [ "$t" = "$current_tool" ]; then
      log "DEBUG" "enforce_boundary: $current_tool allowed for $skill_name"
      return 0
    fi
  done

  # 不在白名�?�?deny
  log "WARN" "enforce_boundary: DENY $current_tool for $skill_name (allowed:$allowed)"
  hook_output 2 "{\"hookSpecificOutput\":{\"hookEventName\":\"PreToolUse\",\"permissionDecision\":\"deny\",\"permissionDecisionReason\":\"${current_tool} 不在 ${skill_name} �?allowed-tools 白名单中\"}}"
}

enforce_boundary
