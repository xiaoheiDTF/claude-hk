---
name: 003-issues
description: 编写和管理 GitHub Issues，支持本地草稿、模板和通过 gh CLI 发布
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Write
  - Edit
  - Glob
  - Grep
hooks:
  InstructionsLoaded:
    - matcher: ""
      hooks:
        - type: command
          command: "bash \"$CLAUDE_PROJECT_DIR/.claude/skills/003-issues/scripts/on_load.sh\""
---

# Issues Skill

在 `$CLAUDE_PROJECT_DIR/doc/issues/` 下编写和管理 GitHub Issues。

## 目录规范

```
doc/issues/
├── drafts/          # 本地草稿，未发布的 issue
└── templates/       # 可复用的 issue 模板
```

## 规则

1. 新建 issue 先写本地草稿到 `doc/issues/drafts/`，确认后再通过 `gh` 发布
2. 草稿文件命名：`<YYYY-MM-DD>-<简短描述>.md`
3. 模板文件命名：`tpl_<名称>.md`，放在 `doc/issues/templates/`
4. 每个草稿/模板必须包含：标题、描述正文、标签（labels）、优先级
5. 发布后保留本地草稿，在文件末尾追加发布记录（issue 编号、URL、发布时间）
6. 优先使用 `gh issue` 命令操作 GitHub，不手动调用 API

## 草稿格式

```markdown
---
title: <issue 标题>
labels: <标签,逗号分隔>
assignee: <指派人,可选>
priority: <P0|P1|P2|P3>
created: <YYYY-MM-DD>
---

## 描述

<正文内容>

## 发布记录

- Issue #<编号>: <URL> (发布于 YYYY-MM-DD HH:MM)
```

## 模板格式

```markdown
---
name: <模板名称>
labels: <默认标签>
priority: <默认优先级>
---

## 描述

<模板正文，用 {{placeholder}} 标记待替换字段>
```

## 操作指引

- 新建草稿 → `doc/issues/drafts/<YYYY-MM-DD>-<描述>.md`
- 新建模板 → `doc/issues/templates/tpl_<名称>.md`
- 从模板创建 → 读取模板，替换占位符，保存为草稿
- 发布 issue → 读取草稿，用 `gh issue create` 发布，追加发布记录
- 查看 issue → `gh issue view <编号>`
- 列出 issue → `gh issue list`

## gh 命令参考

```bash
# 发布
gh issue create --title "<标题>" --body "<正文>" --label "<标签>"

# 查看
gh issue list --state open
gh issue view <编号>

# 关闭
gh issue close <编号>
```
