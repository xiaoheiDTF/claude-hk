#!/bin/bash
# git-push on_load: skill 加载时注入动态上下文
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="999-2-git-push"
source "$PROJECT_DIR/.claude/skills/log.sh"

BRANCH=$(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null || echo "unknown")

skill_log "INFO" "skill 调用: git-push | 分支: $BRANCH"

CONTEXT="[git-push] 提交并推送代码到远程仓库\n提交格式: <type>: <主描述>\n子描述: 每条以 - 开头\n常用 type: fix/feat/update/style/refactor/perf/test/docs/revert/build/chore\n操作: git diff → 分类 git add → git commit → git push"

echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"$CONTEXT\"}}"
