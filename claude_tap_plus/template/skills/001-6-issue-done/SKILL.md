---
name: 001-6-issue-done
description: 标记 issue 开发完成，准备提 PR
user-invocable: true
allowed-tools:
  - Bash
  - Read
---

# Issue Done Skill

标记 issue 开发完成，将状态从 `in-progress` 切换到 `ready-for-pr`。

## 操作流程

1. 用户输入 `/001-6-issue-done #N`
2. 检查当前分支与 issue 对应（分支名包含 `issue-<N>`）
3. 检查是否有未提交的变更（`git status`），如有则提示先提交
4. 在 issue 中 comment："开发完成，等待提 PR"
5. 移除 `in-progress` label，添加 `ready-for-pr` label
6. 提示用户执行 `/001-7-issue-pr #N` 提交 PR

## 规则

1. 必须在对应的 issue 分支上执行
2. 如果有未提交变更，必须先提交再标记完成
3. 自动更新 labels：移除 `in-progress`，添加 `ready-for-pr`

## gh 命令参考

```bash
# 标记完成
gh issue comment <编号> --body "开发完成，等待提 PR"
gh issue edit <编号> --remove-label "in-progress" --add-label "ready-for-pr"
```
