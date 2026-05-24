---
name: "001-testcode-python"
description: "在 doc/testcode/python 目录下编写和管理 Python 测试脚本、API 自动化测试及其他脚本"
---

# 001-testcode-python

Use this skill when you need to add, update, or organize Python scripts under `doc/testcode/python/`.

Directory layout:
```text
doc/testcode/python/
  api/        # API automation tests (HTTP calls)
  other/      # other scripts (utilities, experiments, helpers)
```

Conventions:
1. Put scripts into the correct subfolder; do not scatter files at repo root.
2. Each script starts with a one-line comment stating its purpose.
3. API tests should read config from environment variables (or an existing `.env` used by the repo); do not hardcode secrets.
4. Prefer the standard library; if you must add a third-party dependency, document it at the top of the script.
5. Naming: tests use `test_<name>.py`, utilities use `<name>.py`.

