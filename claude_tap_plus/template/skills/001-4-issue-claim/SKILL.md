---
name: 001-4-issue-claim
description: 原子领取 GitHub Issue，防止多 agent 冲突
user-invocable: true
allowed-tools:
  - Bash
  - Read
---

# Issue 领取 Skill

原子领取 issue，防止多人/多 agent 同时领取同一个 issue。

## 操作流程

1. 用户输入 `/001-4-issue-claim #N`
2. 原子操作：`gh issue edit N --add-assignee @me --add-label "in-progress"`
3. 验证 assignee 是否为自己
4. 如果 issue 有 `rejected` label → 先清除 `rejected` 再加 `in-progress`
5. 输出领取结果

## 规则

1. 领取前不检查 assignee（避免 TOCTOU），直接 assign 后验证
2. 验证时检查 assignee 列表中是否包含自己
3. 被打回的 issue（有 `rejected` label）允许重新领取
4. 领取成功后 issue 进入 `in-progress` 状态

## gh 命令参考

```bash
# 原子领取
gh issue edit <编号> --add-assignee @me --add-label "in-progress"

# 清除打回标记
gh issue edit <编号> --remove-label "rejected"

# 验证 assignee
gh issue view <编号> --json assignees --jq '.assignees[].login'

# 查看 issue 状态
gh issue view <编号> --json state,labels
```
