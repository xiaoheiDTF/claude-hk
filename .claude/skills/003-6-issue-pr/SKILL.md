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

提交 PR 并关联 issue，支持执行测试、审核合并和打回。

## 操作流程

### 提交 PR（默认）

1. 用户输入 `/003-6-issue-pr #N`
2. 确认当前在解决分支上
3. **验证 PR body 必须包含 `## Test plan` 区块**
   - Test Plan 中每一项必须是可勾选的 checkbox：`- [ ] 测试项描述`
   - 如果缺少 Test Plan 区块或格式不对，拒绝创建并提示补充
4. `git push -u origin <branch>`
5. `gh pr create`，body 中包含 `Closes #N` 自动关联 issue
6. 输出 PR 编号和链接

### 执行测试（test 子命令）

1. 用户输入 `/003-6-issue-pr test #N`
2. 根据 PR 编号找到对应 PR（通过 body 中的 `Closes #N` 匹配）
3. 拉取 PR 分支到本地验证
4. 执行 Test Plan 中列出的测试项
5. 每完成一项，更新 PR body 中的 checkbox：`- [ ]` → `- [x]`
6. 全部完成后输出测试结果摘要

**更新 checkbox 的方法：**
```bash
# 获取 PR body
gh pr view <PR编号> --json body --jq '.body'

# 替换后更新（将已完成项的 [ ] 改为 [x]）
gh pr edit <PR编号> --body "<更新后的body>"
```

### 合并（merge 子命令）

1. 用户输入 `/003-6-issue-pr merge #N`
2. 找到关联 issue #N 的 PR
3. **强制检查 Test Plan 完成度**：
   ```bash
   # 拉取 PR body
   gh pr view <PR编号> --json body --jq '.body'

   # 解析 ## Test plan 区块
   # 检查是否还存在未勾选的 - [ ]
   ```
4. **如果存在未勾选的 `- [ ]`**：
   - 阻止合并
   - 列出所有未完成项
   - 提示先执行 `/003-6-issue-pr test #N` 完成测试
5. **如果全部 `[x]`**：
   - 执行 `gh pr merge <PR> --merge`
   - 删除远程分支
   - 输出合并结果

### 打回流程

1. 用户输入 `/003-6-issue-pr reject #N`
2. `gh pr comment <PR> --body "打回原因：xxx"`
3. `gh issue edit <N> --remove-label "in-progress" --add-label "rejected"`
4. `gh issue reopen <N>`（如已关闭）
5. 后续重新走 claim → fix → pr 流程

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
- 提交 PR 时 Test Plan 为未勾选状态，测试通过后逐项勾选

## 规则

1. PR 必须通过 `Closes #N` 关联 issue
2. PR body 必须包含 `## Test plan` 区块，否则拒绝创建
3. **Test Plan 中存在未勾选的 `- [ ]` 时，禁止执行合并**
4. 必须通过 `test` 子命令逐项验证后才能合并
5. 打回必须写明原因
6. 打回后 issue 标记 `rejected`，重新领取时清除

## gh 命令参考

```bash
# 创建 PR
gh pr create --title "<title>" --body "<body>"

# 查看 PR
gh pr view <PR编号>
gh pr diff <PR编号>

# 获取 PR body
gh pr view <PR编号> --json body --jq '.body'

# 更新 PR body（勾选 checkbox）
gh pr edit <PR编号> --body "<更新后的body>"

# 合并（仅在 Test Plan 全部 [x] 时）
gh pr merge <PR编号> --merge

# 打回
gh pr comment <PR编号> --body "打回原因"
gh issue edit <N> --remove-label "in-progress" --add-label "rejected"
gh issue reopen <N>

# 评论
gh issue comment <N> --body "<内容>"
```
