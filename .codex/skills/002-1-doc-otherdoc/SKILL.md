---
name: "002-1-doc-otherdoc"
description: "将用户需要记录的内容以 Markdown 文件存储到 doc/otherDoc 目录，按日期归档"
---

# 002-1-doc-otherdoc

Use this skill to record miscellaneous notes as markdown under `doc/otherDoc/`.

Rules:
1. Store files under `doc/otherDoc/YYYY-MM-DD/` (create the date folder if missing).
2. Use a short, descriptive filename ending in `.md`.
3. Put a short header at the top (date/time + summary), then the body.
4. If appending to an existing file, read it first, then append (do not overwrite).

