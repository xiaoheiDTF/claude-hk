#!/bin/bash
# learn 16Stop: Claude 响应结束后按 session_id 清理 .active
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="999-other-120-learn"
source "$PROJECT_DIR/.claude/skills/log.sh"
source "$PROJECT_DIR/.claude/skills/active.sh"

SESSION_ID="$1"

if [ -n "$SESSION_ID" ]; then
  active_remove "$SESSION_ID"
  skill_log "INFO" "[stop] .active removed session: $SESSION_ID"
else
  active_remove_by_skill "$SKILL_TAG"
  skill_log "WARN" "[stop] no session_id, cleaned by skill name: $SKILL_TAG"
fi
skill_log "INFO" "[stop] skill lifecycle end"
