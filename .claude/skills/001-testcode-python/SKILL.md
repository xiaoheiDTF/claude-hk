---
name: 001-testcode-python
description: 在 doc/testcode/python 目录下编写和管理 Python 测试脚本、API 自动化测试及其他脚本
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
          command: "bash \"$CLAUDE_PROJECT_DIR/.claude/skills/001-testcode-python/scripts/on_load.sh\""
---

# Testcode Python Skill

在 `$CLAUDE_PROJECT_DIR/doc/testcode/python/` 下编写和管理 Python 脚本。

## 目录规范

```
doc/testcode/python/
├── api/            # API 自动化测试
└── other/          # 其他脚本（测试、提示词、工具等）
```

## 规则

1. 所有脚本放到对应子目录，不在根目录散落文件
2. 每个脚本文件开头写明用途（一行注释）
3. API 测试脚本使用 `requests` 库，基础配置从环境变量或 `.env` 读取
4. 优先用标准库，需要第三方库时先在脚本顶部 `import` 前用注释标注依赖
5. 测试脚本命名：`test_<name>.py`，工具脚本命名：`<name>.py`

## 操作指引

- API 测试 → `doc/testcode/python/api/test_<name>.py`
- 其他脚本 → `doc/testcode/python/other/<name>.py`

运行脚本时使用项目自带 Python：

```bash
"$CLAUDE_PROJECT_DIR/.claude/localLanguage/python/python.exe" doc/testcode/python/<path>/script.py
```

如果自带 Python 不可用，回退到系统 `python`。
