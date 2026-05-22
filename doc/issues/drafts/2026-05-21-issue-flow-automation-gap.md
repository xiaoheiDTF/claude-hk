---
title: Issue 生命周期流程缺少自动推进机制，存在未合并分支和缺失实现分支
labels: skill-system,enhancement,process
assignee: 
priority: P1
status: draft
created: 2026-05-21
---

## 描述

在梳理 Issue 闭环流程时发现：**Issue 从创建到完成的整个生命周期不会自动按照规定的流程推进**。具体表现为：

1. Issue 创建后停留在草稿状态，不会自动进入 discuss 阶段
2. Discuss 结束后没有明确的"讨论完成"标记，无法自动流转到 claim
3. Claim 后不会自动触发 fix（创建分支 + 开始解决）
4. Fix 完成后不会自动进入 pr（提交 PR + 测试 + 合并）
5. PR 合并后不会自动关闭关联 Issue 并清理分支

整个流程完全依赖人工判断和手动执行 Skill/Hook，缺乏自动化的状态流转和推进机制。

## 根因分析

### 原因一：存在长期未合并的分支

| 分支 | 状态 | 影响 |
|------|------|------|
| `fix/issue-11-skill-tag-stop-sh` | ⚠️ 刚才才合并到 main | SKILL_TAG 和 16Stop.sh 的补全长期滞留在分支中，主分支的 skill 脚本处于不完整状态 |

该分支包含 3 个 commit，涉及全部 8 个 skill 的 SKILL_TAG 补全和 16Stop.sh 新增。由于长期未合并，主分支上的 skill 在 Hook 触发时缺少必要的 `.active` 清理逻辑，导致技能调度异常。

### 原因二：Issue #5 ~ #8 的实现在 issue 序列中缺失

Issue 列表存在编号断层：

```
#1  CLOSED  16-stop 事件 .active 清理
#2  CLOSED  优化 003-issues Skill，补全生命周期
#3  CLOSED  Skill 级初始化与巡检
#4  CLOSED  Hook Skill 感知调度
       ↓ 缺失 #5 ~ #8
#9  CLOSED  Skill 边界约束
#10 CLOSED  Skill 无参数时列出可选项
#11 CLOSED  SKILL_TAG 和 16Stop.sh 补全
```

虽然 003-5-issue-fix 和 003-6-issue-pr 的 SKILL.md 文件在 PR #8 中一并创建了，但：
- **缺少 Issue #5**：003-5-issue-fix 的"创建分支 + 解决问题"拆分（fix 阶段细化）
- **缺少 Issue #6**：003-6-issue-pr 的"提交PR + 测试 + 合并"拆分（pr 阶段细化）
- **缺少 Issue #7**：003-7-issue-pr-test（执行 Test Plan）
- **缺少 Issue #8**：003-8-issue-pr-merge（合并前 Test Plan 检查）

这些缺失导致 issue 流程中最关键的 **fix → pr → merge** 阶段没有独立的实现分支和跟踪 issue，流程自动化无从谈起。

### 原因三：缺少自动流转触发器

当前流程中，阶段之间的流转完全依赖用户手动触发：

```
当前实际流程：
  issue 创建 → [人工判断] → /003-3-issue-discuss
  discuss → [人工判断] → /003-4-issue-claim
  claim → [人工判断] → /003-5-issue-fix
  fix → [人工判断] → git commit → git push → /003-6-issue-pr
  pr → [人工判断] → /003-6-issue-pr review #N → merge

期望的自动流程：
  issue 创建 → 自动标记 ready-for-discuss
  discuss 达成共识 → 自动标记 ready-for-claim
  claim 成功 → 自动触发 fix（创建分支）
  fix 完成 → 自动标记 ready-for-pr
  pr test 通过 → 自动触发 merge
  merge 完成 → 自动关闭 issue + 清理分支
```

缺少触发器的原因是：
1. 没有定义"阶段完成"的明确条件（如 discuss 怎么算"完成"）
2. 没有定义阶段之间的自动流转规则（如 claim 后是否立即创建分支）
3. 没有定时任务或事件监听来驱动流转（如 stale issue 自动清理）

## 已修复

- [x] `fix/issue-11-skill-tag-stop-sh` 已合并到 main（commit `1771e07`）

## 待修复

- [ ] 补充 Issue #5：003-5-issue-fix 阶段拆分（创建分支 → 解决问题完成）
- [ ] 补充 Issue #6：003-6-issue-pr 阶段拆分（提交PR → 执行测试 → 合并）
- [ ] 定义 discuss → claim 的"讨论完成"判定条件
- [ ] 定义 claim → fix 的自动触发规则（claim 成功后是否立即创建分支？）
- [ ] 定义 fix → pr 的自动触发规则（开发完成后是否自动提醒提 PR？）
- [ ] 定义 pr → merge 的自动触发规则（Test Plan 全部通过是否自动合并？）
- [ ] 建立分支清理自动化（merge 后删除远程/本地分支）

## 涉及文件

| 文件 | 改动说明 |
|------|---------|
| `.claude/skills/003-5-issue-fix/SKILL.md` | 增加"解决问题完成"标记和自动流转逻辑 |
| `.claude/skills/003-6-issue-pr/SKILL.md` | 拆分 PR 阶段，增加自动合并规则 |
| `.claude/settings.json` | 增加阶段流转 Hooks（PostToolUse 自动标记） |
| 新增 issue #5 / #6 / #7 / #8 | 补充缺失的实现跟踪 issue |

## 验收标准

- [ ] Issue 创建后自动获得初始状态标签
- [ ] Discuss 达成共识后自动标记 ready-for-claim
- [ ] Claim 成功后自动创建分支并标记 in-progress
- [ ] Fix 完成后自动标记 ready-for-pr
- [ ] PR Test Plan 全部通过后自动触发合并（或至少提醒）
- [ ] Merge 后自动关闭关联 Issue 并清理分支
- [ ] 所有阶段流转有明确的触发条件和日志记录

## 发布记录

- Issue #21: https://github.com/xiaoheiDTF/claude-hk/issues/21 (发布于 2026-05-21 23:50)

