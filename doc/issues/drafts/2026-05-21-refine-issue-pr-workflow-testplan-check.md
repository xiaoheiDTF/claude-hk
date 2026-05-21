---
title: 细化 issue PR 阶段流程，增加 Test Plan 强制检查机制
labels: skill-system,enhancement
assignee: 
priority: P1
status: draft
created: 2026-05-21
---

## 描述

当前 issue 闭环流程中，存在两个阶段的粒度问题：

**1. Fix 阶段（`003-5-issue-fix`）只到创建分支，缺少"解决问题"的明确环节**
- 当前 fix 的终点是"创建分支并输出分支信息"，后续开发过程无 skill 覆盖
- "解决问题"是一个黑盒过程，缺少明确的开始/结束标记

**2. PR 阶段（`003-6-issue-pr`）过于粗放：提 PR → 审核/合并 一步完成**
- 缺少测试执行的独立环节
- 没有对 PR 中 `Test Plan` 完成度的强制校验

这导致存在以下风险：
1. Fix 阶段职责不清，开发过程无追踪点
2. 测试可能被跳过或遗漏，合并代码质量不可控
3. `Test Plan` 只是格式模板，未勾选完也能合并
4. 审核、测试、合并三个职责混在一起，难以追溯

## 期望行为

将 PR 阶段拆分为 **提交 PR → 执行测试 → 合并** 三个独立步骤，并在合并前强制检查 `Test Plan` 的所有 checkbox 是否已完成。

## 当前流程 vs 目标流程

```
当前流程:
  issue → discuss → claim → fix(创建分支→[黑盒开发]) → pr(提PR→审核/合并一步完成)

目标流程:
  issue → discuss → claim → fix(创建分支 → 解决问题) → pr(提交PR → 执行测试 → 合并)
                                ↑                         ↑
                        分支创建后标记       PR body必须含Test Plan
                        in-progress          合并前强制检查全部[x]
```

## 具体改进方案

### Fix 阶段拆分

#### 1. 创建分支（003-5-issue-fix 现有）

- 检查 assignee
- 根据 label 判断分支类型（bug→fix / enhancement→feat / 其他→chore）
- 自动生成分支名并创建
- **新增**：创建分支后，在 issue comment 中记录 "开始解决，分支: `<branch>`"

#### 2. 解决问题（新增明确环节）

- 在 fix skill 或独立 skill 中，增加「解决问题完成」的显式标记
- 开发完成后，agent 可以执行一个命令（如 `/003-5-issue-fix done #N` 或 `/003-5-issue-done #N`）来：
  - 检查本地是否有未提交的变更
  - 执行基本的代码检查（如 lint、类型检查，如果项目支持）
  - 在 issue 中 comment："开发完成，等待提 PR"
  - 可选：移除 `in-progress` label，添加 `ready-for-pr` label

---

### PR 阶段拆分

#### 1. 提交 PR（已有，需强化）

- PR body **必须**包含 `## Test plan` 区块
- Test Plan 中的每一项必须是可勾选的 checkbox：`- [ ] 测试项描述`
- 如果 PR body 缺少 Test Plan，拒绝创建并提示补充

#### 2. 执行测试（新增环节）

- 新增独立操作或子命令，用于执行 Test Plan 中的测试项
- 执行完成后，在 PR body 中勾选对应项 `[x]`
- 支持在本地 main 分支验证

#### 3. 合并（增加强制检查）

合并前必须执行以下校验：

```bash
# 1. 拉取 PR body
gh pr view <PR> --json body

# 2. 解析 ## Test plan 区块
# 3. 检查是否还存在未勾选的 `- [ ]`
#    - 若存在：阻止合并，列出所有未完成项，提示继续测试
#    - 若全部 [x]：允许执行 gh pr merge
```

**阻断规则**：只要 Test Plan 中还有任一 `[ ]` 未勾选，`gh pr merge` 不得执行。

### Skill 调整建议

#### Fix 阶段

方案 A（推荐，改动小）：
- 扩展 `003-5-issue-fix`，增加子命令：
  - `/003-5-issue-fix #N` —— 创建分支（现有行为）
  - `/003-5-issue-fix done #N` —— 标记"解决问题完成"，添加 `ready-for-pr` label

方案 B（职责更清）：
- 保留 `003-5-issue-fix` 仅负责创建分支
- 新增 `003-5-1-issue-resolve` 或类似 skill，负责开发完成标记和前置检查

#### PR 阶段

方案 A（推荐，改动小）：
- 在 `003-6-issue-pr` 中增加子命令区分：
  - `/003-6-issue-pr #N` —— 提交 PR
  - `/003-6-issue-pr test #N` —— 执行测试并更新 checkbox
  - `/003-6-issue-pr merge #N` —— 检查 Test Plan → 合并

方案 B（职责更清）：
- 拆分出独立 skill：
  - `003-6-issue-pr` —— 仅负责提交 PR
  - `003-7-issue-pr-test` —— 负责执行 Test Plan
  - `003-8-issue-pr-merge` —— 负责检查 Test Plan 完成度并合并

## 涉及文件

| 文件 | 改动说明 |
|------|---------|
| `.claude/skills/003-6-issue-pr/SKILL.md` | 拆分 PR 阶段流程，增加 Test Plan 检查逻辑 |
| `.claude/skills/003-5-issue-fix/SKILL.md` | 增加"解决问题完成"标记逻辑 |
| `.claude/skills/003-6-issue-pr/`（或新增 skill） | 拆分 PR 阶段，增加 Test Plan 检查逻辑 |

## 验收标准

- [ ] Fix 阶段支持"创建分支"和"解决问题完成"两个明确步骤
- [ ] 开发完成后有显式标记（如 `ready-for-pr` label 或 comment）
- [ ] PR body 缺少 `## Test plan` 时，提交被阻断并提示
- [ ] Test Plan 中存在 `[ ]` 未勾选时，合并被阻断并列出未完成项
- [ ] Test Plan 全部 `[x]` 后，可以正常合并
- [ ] 流程文档（SKILL.md）已更新，明确拆分后的各步操作

## 发布记录

- Issue #15: https://github.com/xiaoheiDTF/claude-hk/issues/15 (发布于 2026-05-21 22:23)

