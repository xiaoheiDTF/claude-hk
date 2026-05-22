# 贡献指南

感谢你对 claude-hk 项目的关注！本文档说明如何参与贡献。

## 提 Issue

### Bug 报告

使用 [Bug 报告模板](../../.github/ISSUE_TEMPLATE/bug_report.yml)，包含：
- 问题描述和复现步骤
- 严重程度（Critical / Major / Minor）
- 环境信息（OS、Claude Code 版本）
- 受影响的文件

### 功能请求

使用 [功能请求模板](../../.github/ISSUE_TEMPLATE/feature_request.yml)，包含：
- 目标和非目标（Goals / Non-Goals）
- 预期行为
- 验收标准

### 讨论想法

使用 [Ideas 讨论模板](../../.github/DISCUSSION_TEMPLATE/ideas.yml) 在 Discussions 中发起讨论。

## 提 PR

### 流程

```
1. 选择或创建 Issue
2. /003-4-issue-claim #N           # 领取
3. /003-5-issue-fix #N             # 创建分支
4. 编码...
5. /003-6-issue-done #N            # 标记完成
6. /003-7-issue-pr #N              # 创建 PR
7. /003-8-issue-test #N            # 执行测试
8. /003-9-issue-review merge #N    # 合并
```

### PR 要求

- PR body 必须包含 `## Test plan` 区块
- Test Plan 每项必须使用 `- [ ]` checkbox 格式
- 通过 `Closes #N` 关联 Issue
- 全部测试通过后才可合并

详见 [PR 模板](../../.github/PULL_REQUEST_TEMPLATE/default.md)。

## 新增 Skill

详见 [如何新增 Skill](../features/how-to-add-skill.md)。

简要步骤：
1. 在 `.claude/skills/` 下创建目录
2. 编写 `SKILL.md`（frontmatter + 使用说明）
3. 编写 `scripts/03UserPromptSubmit.sh`（上下文注入）
4. 编写 `scripts/16Stop.sh`（清理）
5. 自动注册（无需手动操作）

## Git 提交规范

详见 [Git 工作流文档](../features/git-workflow.md)。

格式：`<type>: <主描述>`，使用中文。禁止 `git add .`，必须按分组 add。
