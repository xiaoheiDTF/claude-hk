---
name: 003-6-issue-pr
description: 提交 PR 关联 issue，审核合并或打回
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Edit
  - Glob
  - Grep
---

# Issue PR Skill

提交 PR 并关联 issue，支持审核、合并和打回。

## 操作流程

### 提 PR

1. 用户输入 `/003-6-issue-pr #N`
2. 确认当前在解决分支上
3. `git push -u origin <branch>`
4. `gh pr create`，body 中包含 `Closes #N` 自动关联 issue
5. 输出 PR 编号和链接

### 审核/合并

1. 用户输入 `/003-6-issue-pr review #N`
2. 拉取 PR diff，审核代码
3. 在本地 main 测试
4. **通过**：`gh pr merge` 合并
5. **不通过**：打回流程

### 打回流程

1. `gh pr comment <PR> --body "打回原因：xxx"`
2. `gh issue edit <N> --remove-label "in-progress" --add-label "rejected"`
3. `gh issue reopen <N>`（如已关闭）
4. 后续重新走 claim → fix → pr 流程

## PR 格式

```
title: <type>: <简短描述>
body:
  ## Summary
  - 变更点1
  - 变更点2

  ## Test plan
  - [x] 测试项1
  - [x] 测试项2

  Closes #<N>
```

## 规则

1. PR 必须通过 `Closes #N` 关联 issue
2. 合并前必须先测试
3. 打回必须写明原因
4. 打回后 issue 标记 `rejected`，重新领取时清除

## gh 命令参考

```bash
# 创建 PR
gh pr create --title "<title>" --body "<body>"

# 查看 PR
gh pr view <PR编号>
gh pr diff <PR编号>

# 合并
gh pr merge <PR编号> --merge

# 打回
gh pr comment <PR编号> --body "打回原因"
gh issue edit <N> --remove-label "in-progress" --add-label "rejected"
gh issue reopen <N>

# 评论
gh issue comment <N> --body "<内容>"
```
