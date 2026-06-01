---
title: 继续拆分 issue-fix 和 issue-pr：单职责化 Skill 拆分方案
labels: enhancement
assignee:
priority: P1
status: draft
created: 2026-05-22
---

## 描述

当前 `001-5-issue-fix` 承担了 2 个职责（创建分支 + 标记完成），`001-6-issue-pr` 承担了 4 个职责（创建 PR + 执行测试 + 合并 + 打回）。这两个 skill 违反了单职责原则，需要进一步拆分。

## 现状分析

### 001-5-issue-fix（2 个职责）

| 子命令 | 职责 | 触发方式 |
|--------|------|---------|
| 默认 | 根据 issue 创建分支并开始开发 | `/001-5-issue-fix #N` |
| done | 标记开发完成，准备提 PR | `/001-5-issue-fix done #N` |

### 001-6-issue-pr（4 个职责）

| 子命令 | 职责 | 触发方式 |
|--------|------|---------|
| 默认 | 创建 PR 关联 issue | `/001-6-issue-pr #N` |
| test | 执行 Test Plan 并勾选 checkbox | `/001-6-issue-pr test #N` |
| merge | 检查 Test Plan 完成度后合并 | `/001-6-issue-pr merge #N` |
| reject | 打回 PR 并标记 issue | `/001-6-issue-pr reject #N` |

### 问题

1. **一个 skill 承担多个角色**：创建 PR 是开发者动作，测试是 QA 动作，合并/打回是审核者动作，混在一起导致 SKILL.md 定义臃肿
2. **allowed-tools 权限过宽**：test 需要 Run/Verify，merge 只需要 Read+Bash，但因为混在一起只能给最高权限
3. **流程不线性**：用户需要记住子命令，而且 `merge` 和 `reject` 是互斥操作却放在同一个 skill 里
4. **`done` 子命令语义不清晰**：`/001-5-issue-fix done` 听起来像是"完成了修复"，实际只是"标记完成准备提 PR"

## 拆分方案

将 001-5 拆为 2 个 skill，001-6 拆为 3 个 skill，总计从 2 个变为 5 个：

### 拆分后的完整工作流

```
claim(001-4) → fix(001-5) → [编码] → done(001-6) → pr(001-7) → test(001-8) → review(001-9)
                                                                    ├─ merge ✓
                                                                    └─ reject ✗ → 回到 fix(001-5)
```

### 新 skill 定义

| 编号 | 名称 | 职责 | 来源 |
|------|------|------|------|
| 001-5-issue-fix | 创建分支并开始解决 | 从 issue labels 生成分支名，创建分支，评论"开始解决" | 原 001-5 默认动作 |
| 001-6-issue-done | 标记开发完成 | 检查未提交变更，评论"开发完成"，移除 in-progress 添加 ready-for-pr | 原 001-5 done 子命令 |
| 001-7-issue-pr | 创建 PR | 验证 Test Plan 区块，push 并创建 PR 关联 issue | 原 001-6 默认动作 |
| 001-8-issue-test | 执行测试 | 拉取 PR 分支，逐项执行 Test Plan，更新 checkbox | 原 001-6 test 子命令 |
| 001-9-issue-review | 审核合并/打回 | 检查 Test Plan 完成度后合并，或打回并标记 rejected | 原 001-6 merge+reject 子命令 |

### 每个 skill 的 SKILL.md 核心

#### 001-5-issue-fix（不变）

触发：`/001-5-issue-fix #N`
- 检查 assignee → 根据 labels 生成分支名 → 创建分支 → 评论
- allowed-tools: Bash, Read, Edit, Write, Glob, Grep

#### 001-6-issue-done（从 001-5 拆出）

触发：`/001-6-issue-done #N`
- 检查当前分支与 issue 对应 → 检查未提交变更 → 评论"开发完成" → 更新 labels
- allowed-tools: Bash, Read

#### 001-7-issue-pr（从 001-6 拆出，简化）

触发：`/001-7-issue-pr #N`
- 验证 PR body 包含 `## Test plan` 区块 → `git push -u` → `gh pr create`（含 `Closes #N`）
- allowed-tools: Bash, Read, Edit, Write, Glob, Grep

#### 001-8-issue-test（从 001-6 拆出）

触发：`/001-8-issue-test #N`
- 根据 PR body 中的 `Closes #N` 找到 PR → 拉取分支 → 执行 Test Plan → 更新 checkbox
- allowed-tools: Bash, Read, Edit, Glob, Grep

#### 001-9-issue-review（从 001-6 拆出）

触发：`/001-9-issue-review merge #N` 或 `/001-9-issue-review reject #N`
- **merge**：检查 Test Plan 全部 `[x]` → `gh pr merge` → 删除分支
- **reject**：评论打回原因 → 标记 issue rejected → reopen
- allowed-tools: Bash, Read

### 权限收敛对比

| skill（拆分前） | allowed-tools | skill（拆分后） | allowed-tools |
|----------------|--------------|----------------|--------------|
| 001-5-issue-fix | Bash,R,W,E,Glob,Grep | 001-5-issue-fix | Bash,R,W,E,Glob,Grep |
| | | 001-6-issue-done | **Bash,R**（收敛） |
| 001-6-issue-pr | Bash,R,E,Glob,Grep | 001-7-issue-pr | Bash,R,W,E,Glob,Grep |
| | | 001-8-issue-test | Bash,R,E,Glob,Grep |
| | | 001-9-issue-review | **Bash,R**（收敛） |

## 涉及文件

| 文件 | 操作 |
|------|------|
| `.claude/skills/001-5-issue-fix/SKILL.md` | 修改：移除 done 子命令 |
| `.claude/skills/001-5-issue-fix/scripts/03UserPromptSubmit.sh` | 修改：移除 done 相关上下文 |
| `.claude/skills/001-5-issue-fix/scripts/16Stop.sh` | 修改：移除 done 清理逻辑 |
| `.claude/skills/001-5-issue-fix/scripts/init.sh` | 修改：移除 done 初始化 |
| `.claude/skills/001-5-issue-fix/scripts/init_check.sh` | 修改：移除 done 检查 |
| `.claude/skills/001-6-issue-done/SKILL.md` | **新增** |
| `.claude/skills/001-6-issue-done/scripts/03UserPromptSubmit.sh` | **新增** |
| `.claude/skills/001-6-issue-done/scripts/16Stop.sh` | **新增** |
| `.claude/skills/001-6-issue-done/scripts/init.sh` | **新增** |
| `.claude/skills/001-6-issue-done/scripts/init_check.sh` | **新增** |
| `.claude/skills/001-6-issue-pr/SKILL.md` | 修改：移除 test/merge/reject，只保留创建 PR |
| `.claude/skills/001-6-issue-pr/scripts/03UserPromptSubmit.sh` | 修改：简化上下文注入 |
| `.claude/skills/001-6-issue-pr/scripts/16Stop.sh` | 修改：简化 |
| `.claude/skills/001-6-issue-pr/scripts/init.sh` | 修改：简化 |
| `.claude/skills/001-6-issue-pr/scripts/init_check.sh` | 修改：简化 |
| `.claude/skills/001-7-issue-pr/` | **重命名** 从 001-6-issue-pr 改编号 |
| `.claude/skills/001-8-issue-test/SKILL.md` | **新增** |
| `.claude/skills/001-8-issue-test/scripts/*` | **新增** |
| `.claude/skills/001-9-issue-review/SKILL.md` | **新增** |
| `.claude/skills/001-9-issue-review/scripts/*` | **新增** |
| `.claude/skills/registry.conf` | 修改：更新注册表 |
| `.claude/dirs.conf` | 修改：如需新增目录 |
| `CLAUDE.md` | 修改：更新 Skills Summary 表格 |

> **注意**：编号方案有两种选择，见下方"编号方案对比"。

## 编号方案对比

### 方案 A：直接续编号（001-6~001-9）

```
001-5-issue-fix      ← 不变
001-6-issue-done     ← 原 001-5 done
001-7-issue-pr       ← 原 001-6 默认
001-8-issue-test     ← 原 001-6 test
001-9-issue-review   ← 原 001-6 merge+reject
```

- 优点：编号连续，简单直觉
- 缺点：需要把原 `001-6-issue-pr` 重命名为 `001-7-issue-pr`，所有引用都要改

### 方案 B：子编号方案（001-5-1~001-5-2, 001-6-1~001-6-3）

```
001-5-1-issue-fix    ← 原 001-5 默认
001-5-2-issue-done   ← 原 001-5 done
001-6-1-issue-pr     ← 原 001-6 默认
001-6-2-issue-test   ← 原 001-6 test
001-6-3-issue-review ← 原 001-6 merge+reject
```

- 优点：从编号就能看出"同属 fix 阶段"和"同属 pr 阶段"，分组语义清晰
- 缺点：skill 名称更长，registry 匹配规则需要适配 `-` 分隔

**推荐方案 A**：续编号更简洁，且 skill 调用链本身已经是线性的，不需要额外的分组语义。

## 验收标准

- [ ] 001-5-issue-fix 只保留"创建分支"职责
- [ ] 001-6-issue-done 独立处理"标记完成"
- [ ] 001-7-issue-pr 只处理"创建 PR"
- [ ] 001-8-issue-test 只处理"执行测试"
- [ ] 001-9-issue-review 只处理"合并或打回"
- [ ] 所有新 skill 的 `03UserPromptSubmit.sh`、`16Stop.sh`、`init.sh`、`init_check.sh` 就绪
- [ ] `registry.conf` 已更新
- [ ] `CLAUDE.md` Skills Summary 表格已更新
- [ ] labels.conf 可能需要新增 `ready-for-pr` 标签

## 发布记录

- Issue #23: https://github.com/xiaoheiDTF/claude-hk/issues/23 (发布于 2026-05-22)
