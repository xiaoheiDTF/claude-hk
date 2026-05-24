---
name: "003-1-issue-init"
description: "初始化 GitHub 项目的 issue 标签体系（一次性）"
---

# 003-1-issue-init

Use this skill when the repo needs its GitHub issue labels set up (or reconciled) to match the local conventions.

Guidelines:
1. Prefer using `gh label` / `gh api` to create/update labels in the target GitHub repo.
2. Keep label names consistent and avoid duplicates that differ only by case or punctuation.
3. If the repo contains a `labels.conf` (or similar), treat it as source of truth.
4. Do not delete labels unless the user explicitly asks; deprecate by renaming if needed.

