#!/bin/bash
# skill-inject: 从用�?prompt 提取 skill 名，匹配注册表后注入上下�?
# �?03-user-prompt-submit/base.sh 调用
# 参数: $1 = 用户 prompt, $2 = session_id（来�?hook 输入 JSON�?
#
# 匹配规则:
#   - 首字符必须是 /
#   - skill 名到第一个空格结�?
#   - /xxx 后面的内容作为参数传�?

PROMPT="$1"
SESSION_ID="$2"
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILLS_DIR="$PROJECT_DIR/\.codex/skills"
REGISTRY="$SKILLS_DIR/registry.conf"

# 首字符不�?/ �?不是 skill 调用
[[ "$PROMPT" != /* ]] && exit 0

# 提取 skill 名：去掉开头的 /，取到第一个空�?
PROMPT_TRIMMED="${PROMPT#/}"
SKILL_NAME="${PROMPT_TRIMMED%% *}"

# 提取参数：skill 名之后的内容
SKILL_ARGS=""
if [[ "$PROMPT_TRIMMED" == *" "* ]]; then
  SKILL_ARGS="${PROMPT_TRIMMED#* }"
fi

[ -z "$SKILL_NAME" ] && exit 0

# 注册表匹配（�?ASCII�?
MATCH=$(awk -v name="$SKILL_NAME" '$1 == name {print $1}' "$REGISTRY" 2>/dev/null)
[ -z "$MATCH" ] && exit 0

# 查找并运�?03UserPromptSubmit.sh
ON_LOAD="$SKILLS_DIR/$MATCH/scripts/03UserPromptSubmit.sh"
[ ! -f "$ON_LOAD" ] && exit 0

# 执行获取上下�?
CONTEXT=$(bash "$ON_LOAD" 2>/dev/null)
[ -z "$CONTEXT" ] && exit 0

# 写入 .active（使用真�?session_id�?
source "$SKILLS_DIR/active.sh"
active_add "$SESSION_ID" "$MATCH" 2>/dev/null

# 输出：第一行元数据，第二行�?CONTEXT
echo "$MATCH|$SKILL_ARGS"
echo "$CONTEXT"
