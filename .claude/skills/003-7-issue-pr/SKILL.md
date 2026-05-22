---
name: 003-7-issue-pr
description: 创建 PR 关联 issue
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Edit
  - Write
  - Glob
  - Grep
---

# Issue PR Skill

创建 PR 并关联 issue。

## 操作流程

1. 用户输入 `/003-7-issue-pr #N`
2. 确认当前在解决分支上
3. **验证 PR body 必须包含 `## Test plan` 区块**
   - Test Plan 中每一项必须是可勾选的 checkbox：`- [ ] 测试项描述`
   - 如果缺少 Test Plan 区块或格式不对，拒绝创建并提示补充
4. `git push -u origin <branch>`
5. `gh pr create`，body 中包含 `Closes #N` 自动关联 issue
6. 输出 PR 编号和链接

## PR 格式

```
title: <type>: <简短描述>
body:
  ## Summary
  - 变更点1
  - 变更点2

  ## Test plan
  - [ ] 测试项1
  - [ ] 测试项2

  Closes #<N>
```

**Test Plan 格式要求：**
- 必须包含 `## Test plan` 标题（不区分大小写）
- 每项以 `- [ ]` 开头，不可用其他格式
- 提交 PR 时 Test Plan 为未勾选状态

## 规则

1. PR 必须通过 `Closes #N` 关联 issue
2. PR body 必须包含 `## Test plan` 区块，否则拒绝创建
3. 创建 PR 后提示执行 `/003-8-issue-test #N` 执行测试

## gh 命令参考

```bash
# 创建 PR
gh pr create --title "<title>" --body "<body>"

# 查看当前分支的 PR
gh pr list --head <branch> --json number,title,state

# 更新 PR body
gh pr edit <PR编号> --body "<更新后的body>"
```
