# B6-2: 003-5 ~ 003-9 技能改造

> 创建时间：2026-05-27
> 模块：claude-tap-plus / Issue 管理 / 技能改造
> 简述：改造 issue-fix 到 issue-review 五个技能，在关键节点调用后端状态流转 API

---

## 需求描述

在 003-5 至 003-9 五个技能的关键操作后，增加调用 `POST /api/issue/status` 更新后端状态。

## 改造总览

`update_issue_status()` 定义在 `backend.sh`（见 B7），内部自动完成双写：调后端 API + 同步 GitHub label。技能脚本只需调用一行。

| 技能 | 关键操作 | status 值 | GitHub Label 变更（自动） |
|------|----------|-----------|--------------------------|
| 003-5-issue-fix | 创建分支后 | `fixing` | `in-progress` → `fixing` |
| 003-6-issue-done | 标记完成后 | `ready-for-pr` | `fixing` → `ready-for-pr` |
| 003-7-issue-pr | PR 创建后 | `pr-created` | `ready-for-pr` → `pr-created` |
| 003-8-issue-test | 开始测试 | `testing` | `pr-created` → `testing` |
| 003-9-issue-review | 审核开始 | `reviewing` | `testing` → `reviewing` |
| 003-9-issue-review merge | 合并后 | `merged` | `reviewing` → 关闭 issue |
| 003-9-issue-review reject | 打回后 | `rejected` | `reviewing` → `rejected` |

## 前置依赖

各技能脚本开头 source `backend.sh`：

```bash
source "$CLAUDE_PROJECT_DIR/.claude/skills/backend.sh"
```

## 各技能具体改动

### 003-5-issue-fix

**位置**：创建分支成功后

```bash
# 现有：创建分支
git checkout -b $BRANCH_NAME

# 新增：更新后端状态 + 同步 GitHub label（in-progress → fixing）
update_issue_status "$ISSUE_NUM" "fixing"

# 现有：gh issue comment
gh issue comment $ISSUE_NUM --body "开始开发，分支: $BRANCH_NAME"
```

### 003-6-issue-done

**位置**：开发完成标记后

```bash
# 新增：更新后端状态 + 同步 GitHub label（fixing → ready-for-pr）
update_issue_status "$ISSUE_NUM" "ready-for-pr"
```

> 不再需要手动 `gh issue edit --remove-label/--add-label`，`update_issue_status` 内部自动处理。

### 003-7-issue-pr

**位置**：PR 创建成功后

```bash
# 现有：创建 PR
pr_url=$(gh pr create ...)

# 新增：更新后端状态 + 同步 GitHub label（ready-for-pr → pr-created）
update_issue_status "$ISSUE_NUM" "pr-created"
```

### 003-8-issue-test

**位置**：开始执行 Test Plan 时

```bash
# 新增：更新后端状态 + 同步 GitHub label（pr-created → testing）
update_issue_status "$ISSUE_NUM" "testing"
```

### 003-9-issue-review

**位置**：审核开始时标记 `reviewing`，合并或打回后标记终态

```bash
# 新增：审核开始，同步 GitHub label（testing → reviewing）
update_issue_status "$ISSUE_NUM" "reviewing"

if [ "$ACTION" = "merge" ]; then
  # 现有：合并 PR
  gh pr merge $PR_NUM --merge
  # 新增：同步 GitHub label（reviewing → 关闭 issue）
  update_issue_status "$ISSUE_NUM" "merged"
elif [ "$ACTION" = "reject" ]; then
  # 新增：同步 GitHub label（reviewing → rejected）
  update_issue_status "$ISSUE_NUM" "rejected"
fi
```

> 不再需要手动 `gh issue edit --add-label "rejected"`，`update_issue_status` 内部自动处理。

## 降级策略

后端调用失败不影响技能主流程。`update_issue_status` 失败时静默跳过（`> /dev/null 2>&1`），GitHub 操作照常执行。

## 验收标准

- [ ] 每个技能在关键节点调用正确的 status 值
- [ ] 后端可用时，状态流转后 GitHub label 自动同步（旧 label 移除、新 label 添加）
- [ ] 后端不可用时不影响技能主流程，GitHub 不报错
- [ ] 后端可用时状态正确流转
- [ ] session_id 正确传入
- [ ] merged 状态自动关闭 GitHub issue
