#!/bin/bash
# git-push 16Stop: Claude 响应结束后按 session_id 清理 .active
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="004-git-push"
source "$PROJECT_DIR/.claude/skills/log.sh"
source "$PROJECT_DIR/.claude/skills/active.sh"

# $1 = session_id（从 16-stop/base.sh 传入）
SESSION_ID="$1"

BRANCH=$(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null || echo "unknown")
skill_log "INFO" "[stop] skill completed | branch: $BRANCH | session: $SESSION_ID"

# 按 session_id 精确删除 .active 条目
if [ -n "$SESSION_ID" ]; then
  active_remove "$SESSION_ID"
  skill_log "INFO" "[stop] .active removed session: $SESSION_ID"
else
  # 无 session_id 时按 skill 名清理（兜底）
  active_remove_by_skill "$SKILL_TAG"
  skill_log "WARN" "[stop] no session_id, cleaned by skill name: $SKILL_TAG"
fi
skill_log "INFO" "[stop] skill lifecycle end"
