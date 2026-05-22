# 如何新增 Skill

## 概述

Skill 是一个独立的功能单元，通过 `/技能名` 调用。每个 Skill 由目录、定义文件和脚本组成。

## 目录结构

```
.claude/skills/
└── XXX-skill-name/
    ├── SKILL.md                # Skill 定义
    └── scripts/
        ├── 03UserPromptSubmit.sh   # 上下文注入（必须）
        ├── 16Stop.sh               # 清理（必须）
        ├── init.sh                 # 首次运行（可选）
        └── init_check.sh           # 每次会话检查（可选）
```

## 步骤

### 1. 创建目录

```bash
mkdir -p .claude/skills/006-my-skill/scripts
```

编号规则：3 位数字 + 短横线 + 名称，按功能顺序编号。

### 2. 编写 SKILL.md

```markdown
---
name: 006-my-skill
description: 一句话描述
user-invocable: true
allowed-tools:
  - Bash
  - Read
---

# Skill 标题

详细的使用说明...

## 操作流程
1. 步骤一
2. 步骤二

## 规则
- 规则一
- 规则二
```

**frontmatter 字段**：

| 字段 | 必需 | 说明 |
|------|------|------|
| `name` | 是 | 技能名，必须与目录名一致 |
| `description` | 是 | 一句话描述 |
| `user-invocable` | 否 | 是否可通过 `/` 调用（默认 false） |
| `allowed-tools` | 否 | 允许的工具白名单，不设则全部允许 |

### 3. 编写 03UserPromptSubmit.sh

上下文注入脚本，在 Skill 被调用时执行，输出注入到会话。

```bash
#!/bin/bash
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="006-my-skill"
source "$PROJECT_DIR/.claude/skills/log.sh"

PROMPT="$1"

echo "=== My Skill 上下文 ==="
echo "日期: $(date +%Y-%m-%d)"
# 输出动态上下文信息...

skill_log "INFO" "[inject] my-skill context injected"
```

### 4. 编写 16Stop.sh

清理脚本，在 Claude 完成响应后执行。

```bash
#!/bin/bash
PROJECT_DIR="$CLAUDE_PROJECT_DIR"
SKILL_TAG="006-my-skill"
source "$PROJECT_DIR/.claude/skills/log.sh"
source "$PROJECT_DIR/.claude/skills/active.sh"

SESSION_ID="$1"
skill_log "INFO" "[stop] skill completed | session: $SESSION_ID"

if [ -n "$SESSION_ID" ]; then
  active_remove "$SESSION_ID"
else
  active_remove_by_skill "$SKILL_TAG"
fi
```

### 5. 配置输出目录（如需要）

在 `.claude/dirs.conf` 中添加：

```
doc/my-skill/output         输出目录说明
```

### 6. 注册

`skill-register.sh` 会在每次 Stop 事件时自动扫描 `skills/` 目录并注册新 Skill。

如果需要立即使用，手动在 `registry.conf` 中添加一行：

```
006-my-skill
```

## 参考实现

查看现有 Skill 作为参考：

- 简单示例：`002-otherdoc/` — 文档归档
- 复杂示例：`003-7-issue-pr/` — 创建 PR 关联 issue
