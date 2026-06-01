---
name: 001-8-issue-test
description: 执行 PR 的 Test Plan 并更新 checkbox
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Edit
  - Glob
  - Grep
---

# Issue Test Skill

执行 PR 的 Test Plan，逐项验证并更新 checkbox。

## 操作流程

1. 用户输入 `/001-8-issue-test #N`
2. 根据 issue #N 找到关联 PR（通过 PR body 中的 `Closes #N` 匹配）
3. 拉取 PR 分支到本地验证
4. 读取 PR body 中的 `## Test plan` 区块
5. 逐项执行 Test Plan 中列出的测试项
6. 每完成一项，更新 PR body 中的 checkbox：`- [ ]` → `- [x]`
7. 全部完成后输出测试结果摘要

## 更新 checkbox 的方法

```bash
# 获取 PR body
gh pr view <PR编号> --json body --jq '.body'

# 替换后更新（将已完成项的 [ ] 改为 [x]）
gh pr edit <PR编号> --body "<更新后的body>"
```

## 规则

1. 必须找到关联 issue 的 PR 才能执行
2. Test Plan 中每一项都需要实际验证
3. 验证通过后更新 checkbox
4. 全部通过后提示执行 `/001-9-issue-review merge #N`

## gh 命令参考

```bash
# 找到关联 issue 的 PR
gh pr list --state open --json number,body --jq ".[] | select(.body | test(\"Closes #$ISSUE_NUM\"))"

# 获取 PR body
gh pr view <PR编号> --json body --jq '.body'

# 更新 PR body
gh pr edit <PR编号> --body "<更新后的body>"

# 拉取 PR 分支
gh pr checkout <PR编号>
```
