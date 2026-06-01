---
name: 002-2-doc-testcode-python
description: 在 doc/testcode/python 目录下编写和管理 Python 测试脚本、API 自动化测试及其他脚本
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Write
  - Edit
  - Glob
  - Grep
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

---

## to 模式：将聊天记录转为测试代码文档

触发：用户明确输入 `/002-2-doc-testcode-python to` 并指定范围。

规则：
1. 必须用户明确触发，不可自动执行
2. 必须用户指定范围
3. 从聊天记录中提取测试相关内容，生成 Python 测试脚本
4. 脚本放到对应子目录（api/ 或 other/）
