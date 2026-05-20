---
name: 002-otherdoc
description: 将用户需要记录的内容以 Markdown 文件存储到 doc/otherDoc 目录，按日期归档
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Write
  - Edit
  - Glob
  - Grep
hooks:
  UserPromptSubmit:
    - matcher: ""
      hooks:
        - type: command
          command: "bash \"$CLAUDE_PROJECT_DIR/.claude/skills/002-otherdoc/scripts/on_load.sh\""
  Stop:
    - matcher: ""
      hooks:
        - type: command
          command: "bash \"$CLAUDE_PROJECT_DIR/.claude/skills/002-otherdoc/scripts/on_stop.sh\""
---

# OtherDoc Skill

将用户需要记录的内容写入 Markdown 文件，按日期归档。

## 目录

- 存储根目录：`$CLAUDE_PROJECT_DIR/doc/otherDoc/`
- 按日期建子目录：`YYYY-MM-DD/`
- 文件命名：简短中文或英文描述，扩展名 `.md`

## 规则

1. 每次写入前先确认日期目录存在，不存在则创建
2. 文件名从用户描述的内容中提取关键词
3. 文件开头写明创建时间和简述
4. 如果用户要追加内容到已有文件，先读取再追加

## 示例路径

```
doc/otherDoc/2026-05-19/会议记录.md
doc/otherDoc/2026-05-19/API设计思路.md
```
