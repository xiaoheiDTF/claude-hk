---
title: "[TEST] 多标签 issue 领取与释放"
labels: documentation,good-first-issue,P3
assignee:
priority: P3
status: draft
created: 2026-05-31
---

## 描述

测试用途：验证多标签 issue 的领取、释放机制，以及 29-session-end 自动释放功能。

**预期行为：**
1. 001-4-issue-claim 领取成功，后端记录 claim
2. 同一 issue 重复领取应被拒绝（原子性）
3. 29-session-end 钩子触发时自动释放该 session 的所有 issue
4. documentation 类型在 001-5-issue-fix 中应生成 `docs/` 前缀分支

**验证范围：** 后端原子领取、并发冲突、session-end 自动释放

## 发布记录

- Issue #31: https://github.com/xiaoheiDTF/claude-hk/issues/31 (发布于 2026-05-31)
