#!/bin/bash
# 003-5-issue-fix 16Stop: Claude 响应结束后按 session_id 清理 .active
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="003-5-issue-fix"
source "$PROJECT_DIR/\.codex/skills/log.sh"
source "$PROJECT_DIR/\.codex/skills/active.sh"

# $1 = session_id（从 16-stop/base.sh 传入�?
SESSION_ID="$1"

skill_log "INFO" "[stop] skill completed | session: $SESSION_ID"

if [ -n "$SESSION_ID" ]; then
  active_remove "$SESSION_ID"
  skill_log "INFO" "[stop] .active removed session: $SESSION_ID"
else
  active_remove_by_skill "$SKILL_TAG"
  skill_log "WARN" "[stop] no session_id, cleaned by skill name: $SKILL_TAG"
fi
skill_log "INFO" "[stop] skill lifecycle end"
