---
title: "[TEST] bug 类型标签验证"
labels: bug,P2
assignee:
priority: P2
status: draft
created: 2026-05-31
---

## 描述

测试用途：验证 003-4-issue-claim 领取 bug 类型 issue 后，003-5-issue-fix 能正确生成 `fix/` 前缀分支名。

**预期行为：**
1. 领取后 issue 状态变为 in-progress
2. 创建分支名格式为 `fix/<编号>-bug-label-check`
3. 003-6-issue-done 标记完成后可正常进入 PR 流程

**验证范围：** 003-4 → 003-5 → 003-6 流程

## 发布记录

- Issue #29: https://github.com/xiaoheiDTF/claude-hk/issues/29 (发布于 2026-05-31)
