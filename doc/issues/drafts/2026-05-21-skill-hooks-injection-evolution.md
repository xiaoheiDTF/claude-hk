---
title: Skill Hooks 注入机制演变记录与架构决策
labels: architecture,hooks,skills
priority: P1
created: 2026-05-21
---

## 描述

记录 skill 动态上下文注入机制从最初设计到最终方案的完整演变过程，包括每个阶段的方案、遇到的问题、最终决策及其原因。供后续维护和新 skill 开发参考。

---

## 阶段一：InstructionsLoaded 全局 Hook（已废弃）

### 方案

在 `settings.json` 的 `InstructionsLoaded` 事件中为每个 skill 注册 hook，会话启动时自动运行 `on_load.sh` 注入上下文。

### 代码示例

**settings.json 配置：**
```json
{
  "InstructionsLoaded": [
    {
      "matcher": "",
      "hooks": [
        {
          "type": "command",
          "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/skills/001-testcode-python/scripts/on_load.sh\""
        }
      ]
    }
  ]
}
```

**on_load.sh 输出格式：**
```bash
cat <<EOF
{"hookSpecificOutput":{"hookEventName":"InstructionsLoaded","additionalContext":"[testcode-python] 日期: $TODAY\n..."}}
EOF
```

### 问题

1. **每次会话启动时所有 skill 都注入**，即使没有调用，浪费上下文窗口
2. 用户要求「只在调用 skill 时才注入」
3. 多个 skill 同时注入会导致上下文冗余

### 涉及文件

- `.claude/settings.json` — InstructionsLoaded hook 注册
- `.claude/skills/*/scripts/on_load.sh` — 注入脚本
- `.claude/hooks/19-instructions-loaded/base.sh` — 全局 InstructionsLoaded hook

---

## 阶段二：PreToolUse matcher 匹配 Skill 工具（已废弃）

### 方案

在 `PreToolUse` hook 中设置 `matcher: "Skill"`，当调用 `/技能名` 时拦截 Skill 工具，运行对应 `on_load.sh` 注入上下文。

### 代码示例

**settings.json 配置：**
```json
{
  "PreToolUse": [
    {
      "matcher": "Skill",
      "hooks": [
        {
          "type": "command",
          "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/05-pre-tool-use/skill-inject.sh\""
        }
      ]
    }
  ]
}
```

**skill-inject.sh 逻辑：**
```bash
tool_name=$(json_get '.tool_name')
[ "$tool_name" != "Skill" ] && exit 0
skill_name=$(json_get '.tool_input.skill')
# 查找并运行 on_load.sh
ON_LOAD="$SKILLS_DIR/$skill_name/scripts/on_load.sh"
CONTEXT=$(bash "$ON_LOAD" 2>/dev/null)
echo "{\"hookSpecificOutput\":{\"hookEventName\":\"PreToolUse\",\"additionalContext\":\"$ESCAPED\"}}"
```

### 问题

1. **调用 skill 时 PreToolUse hook 根本不触发** — skill 的调用不走标准的工具调用流程
2. 日志验证：`grep "tool_name" logs | sort -u` 只有 Bash/Read/Write/Edit/Glob/Grep，没有 Skill

### 涉及文件

- `.claude/settings.json` — PreToolUse matcher 配置
- `.claude/hooks/05-pre-tool-use/skill-inject.sh` — 匹配脚本（已删除）

---

## 阶段三：UserPromptSubmit + skill-inject.sh 全局匹配（曾使用，已替换）

### 方案

在 `03-user-prompt-submit` hook 中检测用户输入，匹配 `/数字开头` 的 prompt，提取 skill 名，查注册表后运行 `on_load.sh` 注入上下文。

### 完整源码

**03-user-prompt-submit/base.sh（阶段三版本）：**
```bash
#!/bin/bash
# 03-user-prompt-submit: 用户提交提示后、Claude 处理之前触发 (每轮一次)
# 退出 2 可阻止提示处理并从上下文中删除提示

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../base.sh"

prompt=$(json_get '.prompt')

log "INFO" "prompt=$prompt"

# skill 注入逻辑：检测 /xxx 格式调用
INJECT=$(bash "$SCRIPT_DIR/skill-inject.sh" "$prompt")
if [ -n "$INJECT" ] && [ "$INJECT" != "" ]; then
  log "INFO" "Skill context injected"
  hook_output 0 "$INJECT"
fi

hook_output 0 '{}'
```

**03-user-prompt-submit/skill-inject.sh（已删除，阶段三核心注入脚本）：**
```bash
#!/bin/bash
# skill-inject: 从用户 prompt 中提取 skill 名，匹配注册表后注入上下文
# 被 03-user-prompt-submit/base.sh 调用
# 参数: $1 = 用户 prompt 文本

PROMPT="$1"
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILLS_DIR="$PROJECT_DIR/.claude/skills"
REGISTRY="$SKILLS_DIR/registry.conf"

# 提取 /xxx 格式的 skill 名（匹配 /数字开头的 prompt）
SKILL_NAME=$(echo "$PROMPT" | sed -n 's|^[[:space:]]*/\([^[:space:]]*\).*$|\1|p')

# 没有 /xxx 格式，不是 skill 调用
[ -z "$SKILL_NAME" ] && exit 0

# 注册表匹配（纯 ASCII，防止 Windows GBK 乱码）
MATCH=$(awk -v name="$SKILL_NAME" '$1 == name {print $1}' "$REGISTRY" 2>/dev/null)

# 不在注册表中，跳过（防止用户误输入）
[ -z "$MATCH" ] && exit 0

# 查找并运行 on_load.sh
ON_LOAD="$SKILLS_DIR/$MATCH/scripts/on_load.sh"
[ ! -f "$ON_LOAD" ] && exit 0

# 执行 on_load.sh 获取上下文
CONTEXT=$(bash "$ON_LOAD" 2>/dev/null)
[ -z "$CONTEXT" ] && exit 0

# 转义: 反斜杠 → 双反斜杠，双引号 → 转义双引号，换行 → \n
ESCAPED=$(echo "$CONTEXT" | sed 's/\\/\\\\/g; s/"/\\"/g' | awk '{printf "%s%s", (NR>1?"\\n":""), $0}')

# 输出 JSON 格式的 hook 注入结果
echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"$ESCAPED\"}}"
```

**registry.conf（纯 ASCII，避免 Windows GBK 乱码）：**
```
# skills registry
# skill-inject.sh reads this file for matching
# one skill name per line, no chinese, no description
001-testcode-python
002-otherdoc
003-issues
004-git-push
005-git-commit
```

**各 skill 的 on_load.sh（阶段三版本，与阶段四相同，不需修改）：**

on_load.sh 在阶段三和阶段四中完全相同——stdout 输出 JSON。区别仅在于**调用方**：
- 阶段三：由 `skill-inject.sh` 调用，经过注册表匹配 + 转义后注入
- 阶段四：由 SKILL.md frontmatter 的 `UserPromptSubmit` hook 直接调用

以 `004-git-push/scripts/on_load.sh` 为例：
```bash
#!/bin/bash
# git-push on_load: skill 加载时注入动态上下文
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="git-push"
source "$PROJECT_DIR/.claude/skills/log.sh"

BRANCH=$(git -C "$PROJECT_DIR" branch --show-current 2>/dev/null || echo "unknown")
skill_log "INFO" "skill 调用: git-push | 分支: $BRANCH"

CONTEXT="[git-push] 提交并推送代码到远程仓库\n提交格式: <type>: <主描述>\n子描述: 每条以 - 开头\n常用 type: fix/feat/update/style/refactor/perf/test/docs/revert/build/chore\n操作: git diff → 分类 git add → git commit → git push"

echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"$CONTEXT\"}}"
```

### 优点

1. 成功注入，在 Claude 处理用户输入之前生效
2. 只有实际调用 skill 时才注入
3. 注册表机制防止用户误输入

### 问题

1. 依赖全局 hook 中转，新增 skill 需要手动注册到 registry.conf
2. 所有 skill 的注入逻辑集中在 `skill-inject.sh`，不够解耦
3. 没有自动注册机制（后通过 `16-stop/skill-register.sh` 解决）

### 涉及文件

- `.claude/hooks/03-user-prompt-submit/base.sh` — 调用 skill-inject.sh
- `.claude/hooks/03-user-prompt-submit/skill-inject.sh` — 匹配+注入（已删除）
- `.claude/skills/registry.conf` — skill 注册表
- `.claude/hooks/16-stop/skill-register.sh` — 自动注册新 skill

---

### 阶段三→阶段四回退操作指南

如需从阶段四（当前 frontmatter hooks 方案）回退到阶段三（全局 skill-inject.sh 方案），需修改以下文件：

#### 回退步骤

**步骤 1：恢复 `03-user-prompt-submit/skill-inject.sh`（重新创建）**

将上方「完整源码」中的 `skill-inject.sh` 内容写入：
```
.claude/hooks/03-user-prompt-submit/skill-inject.sh
```
确保文件有可执行权限。

**步骤 2：修改 `03-user-prompt-submit/base.sh`（恢复注入调用）**

将当前版本：
```bash
prompt=$(json_get '.prompt')
log "INFO" "prompt=$prompt"
hook_output 0 '{}'
```

改为阶段三版本（在 `log "INFO"` 和 `hook_output` 之间插入注入逻辑）：
```bash
prompt=$(json_get '.prompt')
log "INFO" "prompt=$prompt"

# skill 注入逻辑：检测 /xxx 格式调用
INJECT=$(bash "$SCRIPT_DIR/skill-inject.sh" "$prompt")
if [ -n "$INJECT" ] && [ "$INJECT" != "" ]; then
  log "INFO" "Skill context injected"
  hook_output 0 "$INJECT"
fi

hook_output 0 '{}'
```

**步骤 3：移除所有 SKILL.md frontmatter 中的 hooks 字段**

对每个 `.claude/skills/*/SKILL.md`，删除 frontmatter 中的 `hooks:` 整个块。以 `004-git-push/SKILL.md` 为例：

修改前：
```yaml
---
name: 004-git-push
description: 按规范格式提交代码并推送到远程仓库（commit + push）
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Edit
  - Glob
  - Grep
hooks:
  UserPromptSubmit:
    - matcher: ""
      hooks:
        - type: command
          command: "bash \"$CLAUDE_PROJECT_DIR/.claude/skills/004-git-push/scripts/on_load.sh\""
  Stop:
    - matcher: ""
      hooks:
        - type: command
          command: "bash \"$CLAUDE_PROJECT_DIR/.claude/skills/004-git-push/scripts/on_stop.sh\""
---
```

修改后：
```yaml
---
name: 004-git-push
description: 按规范格式提交代码并推送到远程仓库（commit + push）
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Edit
  - Glob
  - Grep
---
```

需要同步修改的 SKILL.md：
- `.claude/skills/001-testcode-python/SKILL.md`
- `.claude/skills/002-otherdoc/SKILL.md`
- `.claude/skills/003-issues/SKILL.md`
- `.claude/skills/004-git-push/SKILL.md`
- `.claude/skills/005-git-commit/SKILL.md`

**步骤 4：确认 `on_load.sh` 无需修改**

阶段三和阶段四的 `on_load.sh` 内容完全相同（stdout 输出 JSON），无需任何修改。

**步骤 5：`on_stop.sh` 可保留或删除**

阶段三不使用 `on_stop.sh`（收尾日志是阶段四 frontmatter Stop hook 引入的）。文件保留不影响功能，但不会被执行。

**步骤 6：确认 `registry.conf` 已包含所有 skill**

`16-stop/skill-register.sh` 在每次 Claude 响应结束时自动扫描并注册新 skill，无需手动维护。

#### 回退文件同步清单

| 文件 | 操作 | 备注 |
|------|------|------|
| `.claude/hooks/03-user-prompt-submit/skill-inject.sh` | **新建** | 恢复阶段三完整源码 |
| `.claude/hooks/03-user-prompt-submit/base.sh` | **修改** | 插入 skill-inject.sh 调用 |
| `.claude/skills/001-testcode-python/SKILL.md` | **修改** | 移除 frontmatter hooks |
| `.claude/skills/002-otherdoc/SKILL.md` | **修改** | 移除 frontmatter hooks |
| `.claude/skills/003-issues/SKILL.md` | **修改** | 移除 frontmatter hooks |
| `.claude/skills/004-git-push/SKILL.md` | **修改** | 移除 frontmatter hooks |
| `.claude/skills/005-git-commit/SKILL.md` | **修改** | 移除 frontmatter hooks |
| `.claude/skills/*/scripts/on_load.sh` | 不变 | 阶段三/四内容一致 |
| `.claude/skills/*/scripts/on_stop.sh` | 可选删除 | 阶段三不使用 |
| `.claude/skills/registry.conf` | 不变 | 已由 skill-register.sh 维护 |
| `.claude/skills/log.sh` | 不变 | 日志模块通用 |
| `.claude/hooks/16-stop/skill-register.sh` | 不变 | 自动注册逻辑通用 |
| `.claude/settings.json` | 不变 | 无 skill 相关 hook 配置 |

---

## 阶段四：Skill Frontmatter 自带 Hooks（当前方案）

### 方案

利用 Claude Code 的 skill frontmatter hooks 特性：在 SKILL.md 的 YAML frontmatter 中直接定义 `UserPromptSubmit` 和 `Stop` hooks。Hooks 的范围限于 skill 的生命周期，skill 完成时自动清理。

### 代码示例

**SKILL.md frontmatter：**
```yaml
---
name: 004-git-push
description: 按规范格式提交代码并推送到远程仓库（commit + push）
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Edit
  - Glob
  - Grep
hooks:
  UserPromptSubmit:
    - matcher: ""
      hooks:
        - type: command
          command: "bash \"$CLAUDE_PROJECT_DIR/.claude/skills/004-git-push/scripts/on_load.sh\""
  Stop:
    - matcher: ""
      hooks:
        - type: command
          command: "bash \"$CLAUDE_PROJECT_DIR/.claude/skills/004-git-push/scripts/on_stop.sh\""
---
```

**on_load.sh（stdout 输出 JSON）：**
```bash
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="git-push"
source "$PROJECT_DIR/.claude/skills/log.sh"

skill_log "INFO" "skill 调用: git-push | 分支: $(git -C ... branch --show-current)"

CONTEXT="[git-push] 提交并推送代码到远程仓库\n..."
echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"$CONTEXT\"}}"
```

**on_stop.sh（收尾日志）：**
```bash
SKILL_TAG="git-push"
source "$PROJECT_DIR/.claude/skills/log.sh"
skill_log "INFO" "[stop] skill completed"
skill_log "INFO" "[stop] skill lifecycle end"
```

### 优点

1. **自包含**：每个 skill 自带 hooks，不依赖全局 hook 中转
2. **自动生命周期管理**：skill 激活时 hooks 生效，完成时自动清理
3. **新增 skill 简单**：只需在 SKILL.md frontmatter 加 hooks 字段 + 写对应脚本
4. **日志双写**：统一日志（hooks/logs）+ 模块日志（skills/log/\<skill名\>/）

### 已知问题

1. **多 skill 上下文叠加**：连续调用多个 skill 时，上一个 skill 的 `UserPromptSubmit` hook 仍在活跃，会叠加注入上下文。因为 skill 的 `Stop` hook 要到 Claude 停止响应时才触发清理，用户如果在中途调用新 skill，前一个的 hooks 还没清理。
2. `InstructionsLoaded` 在 skill frontmatter 中不生效（仅在会话启动加载 CLAUDE.md 时触发）

### 涉及文件

- `.claude/skills/*/SKILL.md` — frontmatter 中定义 hooks
- `.claude/skills/*/scripts/on_load.sh` — 注入动态上下文（stdout 输出 JSON）
- `.claude/skills/*/scripts/on_stop.sh` — 收尾日志
- `.claude/skills/log.sh` — 统一日志模块
- `.claude/hooks/16-stop/skill-register.sh` — 自动注册新 skill（保留）

---

## 文件修改清单

| 文件 | 阶段 | 变更 |
|------|------|------|
| `.claude/settings.json` | 阶段1→3 | 移除 InstructionsLoaded/PreToolUse 的 skill 相关 hook |
| `.claude/hooks/03-user-prompt-submit/base.sh` | 阶段3→4 | 移除 skill-inject.sh 调用，恢复原始 |
| `.claude/hooks/03-user-prompt-submit/skill-inject.sh` | 阶段3 | 已删除 |
| `.claude/hooks/05-pre-tool-use/skill-inject.sh` | 阶段2 | 已删除 |
| `.claude/hooks/16-stop/base.sh` | 阶段3 | 新增 skill-register.sh 调用 |
| `.claude/hooks/16-stop/skill-register.sh` | 阶段3 | 新增，自动扫描注册 |
| `.claude/hooks/29-session-end/base.sh` | 全阶段 | 清理模板注释 |
| `.claude/skills/*/SKILL.md` | 阶段4 | frontmatter 新增 UserPromptSubmit + Stop hooks |
| `.claude/skills/*/scripts/on_load.sh` | 阶段4 | stdout 输出 JSON 格式 |
| `.claude/skills/*/scripts/on_stop.sh` | 阶段4 | 新增收尾日志脚本 |
| `.claude/skills/log.sh` | 阶段4 | 新增统一日志模块 |
| `.claude/skills/registry.conf` | 阶段3 | 新增纯 ASCII 注册表 |

---

## 提示词注入逻辑总结

### 注入时机

- `UserPromptSubmit`：用户提交输入后、Claude 处理前 → **即时注入当前轮**
- `Stop`：Claude 响应结束后 → 注入下一轮（当前方案用于收尾日志）
- `InstructionsLoaded`：仅会话启动时 → 不适合 skill 动态注入

### 注入格式

```bash
echo "{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"<转义后的内容>\"}}"
```

- stdout 必须是合法 JSON
- 日志写文件（不影响 stdout）
- `\n` 转义换行，`\"` 转义引号，`\\` 转义反斜杠

### 日志架构

```
.claude/hooks/logs/2026-05-21.log          ← 统一日志（所有模块）
.claude/skills/log/testcode-python/2026-05-21.log  ← 模块日志
.claude/skills/log/git-push/2026-05-21.log
.claude/skills/log/git-commit/2026-05-21.log
```

---

## 新增 Skill 检查清单

1. 创建目录 `.claude/skills/<skill名>/`
2. 编写 `SKILL.md`（含 frontmatter hooks 定义）
3. 编写 `scripts/on_load.sh`（stdout JSON + 日志双写）
4. 编写 `scripts/on_stop.sh`（收尾日志）
5. `16-stop/skill-register.sh` 会自动注册到 `registry.conf`
