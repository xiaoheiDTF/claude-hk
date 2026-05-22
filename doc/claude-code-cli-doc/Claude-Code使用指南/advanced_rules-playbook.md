# Claude-Code / Advanced / Rules-Playbook

> 来源: claudecn.com

# 团队规则库：用 Rules 固化底线

当你希望 Claude Code 在团队里“长期稳定可控”，Rules 往往比技巧更重要：它们把安全、测试、代码风格、Git 流程等底线固化下来，减少每次都要口头纠正的成本。

下文给出一套团队可落地的规则库结构与取舍建议，重点是让规则可审查、可执行、可迭代。

## 1) 规则库建议拆分（从 5 个文件起步）

推荐把规则拆成小文件，便于审查与迭代：

- security.md：Secrets、输入校验、注入/XSS/CSRF 等底线
- testing.md：TDD 流程与关键路径测试要求
- coding-style.md：不可变、文件组织、错误处理、输入校验
- git-workflow.md：提交格式、PR 流程、变更范围
- performance.md：模型选择策略、上下文窗口管理、排障节奏
提醒：不要一开始就把规则写得非常“理想化”。规则能落地、团队能执行，比“看起来很强”更重要。

## 2) coding-style：不可变与组织方式（示例节选）

`coding-style.md` 把“不可变”写成硬规则：

```javascript
// WRONG: Mutation
function updateUser(user, name) {
  user.name = name
  return user
}

// CORRECT: Immutability
function updateUser(user, name) {
  return { ...user, name }
}
```

同时给出可操作的组织标准（节选）：

- 多小文件优于少大文件（高内聚低耦合）
- 函数尽量小（例如 < 50 行）
- 文件尽量聚焦（例如 < 800 行）
## 3) security：提交前的必过清单（示例节选）

`security.md` 的定位是“合并门禁前必须过”的清单：

- 禁止硬编码 secrets
- 所有用户输入必须校验
- 参数化查询防 SQL 注入
- HTML 输出要防 XSS
- CSRF 防护
- 鉴权/授权检查
- 速率限制
- 错误信息不泄露敏感数据
如果你希望把安全审查流程化，建议结合：

- 安全审查工作流：从 Secrets 到 OWASP
- 代码审查工作流：分级输出与合并门禁
## 4) testing：把 TDD 变成团队共识（示例节选）

`testing.md` 的核心是“测试先行 + 覆盖关键路径”（节选）：

- 单元测试 / 集成测试 / E2E 都要覆盖关键旅程
- 采用 Red → Green → Refactor 的节奏
E2E 的落地建议见：[E2E 测试工作流：Playwright 关键旅程与产物管理](https://claudecn.com/docs/claude-code/workflows/e2e-testing/)。

## 5) git-workflow：让 PR 审查的输入质量更高（示例节选）

`git-workflow.md` 给出两条非常实用的约束（节选）：

- 提交信息使用固定类型（feat/fix/refactor/docs/test/…）
- PR 创建时看全量差异：git diff [base-branch]...HEAD
这些规则能降低“审查只看到了最后一次提交”的漏检风险。

## 6) performance：把上下文当作有限资源管理（示例节选）

`performance.md` 强调两点：

- 大任务不要把上下文窗口用到最后 20%（容易跑偏/遗忘）
- 复杂任务用 Plan Mode + 多轮 critique
相关页面：

- 计划模式
- 会话连续性与战略压缩
## 7) 如何落地到团队仓库

落地的最小方式是：

- 把规则文件放到项目级 .claude/rules/ 并纳入版本控制
- 在 CLAUDE.md 写清楚“本项目必须遵守的规则入口”
- 用 /command 固化流程（plan/tdd/build-fix/code-review）
快速模板见：[团队 Starter Kit（最小可用配置）](https://claudecn.com/docs/claude-code/advanced/starter-kit/)。

## 参考

无
