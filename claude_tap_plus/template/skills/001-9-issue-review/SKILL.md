---
name: 001-9-issue-review
description: 审核 PR：合并或打回
user-invocable: true
allowed-tools:
  - Bash
  - Read
---

# Issue Review Skill

审核 PR，执行合并或打回操作。

## 操作流程

### 合并（merge）

1. 用户输入 `/001-9-issue-review merge #N`
2. 找到关联 issue #N 的 PR
3. **强制检查 Test Plan 完成度**：
   ```bash
   # 拉取 PR body
   gh pr view <PR编号> --json body --jq '.body'
   # 解析 ## Test plan 区块，检查是否还存在未勾选的 - [ ]
   ```
4. **如果存在未勾选的 `- [ ]`**：
   - 阻止合并
   - 列出所有未完成项
   - 提示先执行 `/001-8-issue-test #N` 完成测试
5. **如果全部 `[x]`**：
   - 执行 `gh pr merge <PR> --merge`
   - 删除远程分支
   - 输出合并结果

### 打回（reject）

1. 用户输入 `/001-9-issue-review reject #N`
2. `gh pr comment <PR> --body "打回原因：xxx"`
3. `gh issue edit <N> --remove-label "in-progress" --add-label "rejected"`
4. `gh issue reopen <N>`（如已关闭）
5. 后续重新走 claim → fix → done → pr → test → review 流程

## 规则

1. **Test Plan 中存在未勾选的 `- [ ]` 时，禁止执行合并**
2. 打回必须写明原因
3. 打回后 issue 标记 `rejected`，重新领取时清除

## gh 命令参考

```bash
# 找到关联 issue 的 PR
gh pr list --state open --json number,body --jq ".[] | select(.body | test(\"Closes #$ISSUE_NUM\"))"

# 合并（仅在 Test Plan 全部 [x] 时）
gh pr merge <PR编号> --merge

# 打回
gh pr comment <PR编号> --body "打回原因"
gh issue edit <N> --remove-label "in-progress" --add-label "rejected"
gh issue reopen <N>
```
