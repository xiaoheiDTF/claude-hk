#!/bin/bash
# git-commit on_load: skill 加载时注入动态上下文
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="999-1-git-commit"
source "$PROJECT_DIR/.claude/skills/log.sh"

BRANCH=$(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null || echo "unknown")

skill_log "INFO" "skill 调用: git-commit | 分支: $BRANCH"

CONTEXT="[git-commit] 仅提交代码到本地仓库（不推送）\n提交格式: <type>: <主描述>\n子描述: 每条以 - 开头\n常用 type: fix/feat/update/style/refactor/perf/test/docs/revert/build/chore\n操作: git diff → 分类 git add → git commit（无 push）"

echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"$CONTEXT\"}}"
