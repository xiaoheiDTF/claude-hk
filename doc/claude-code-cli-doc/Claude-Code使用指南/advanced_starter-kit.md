# Claude-Code / Advanced / Starter-Kit

> 来源: claudecn.com

# 团队 Starter Kit（最小可用配置）

把 Claude Code 用在团队里，最容易踩的坑不是“不会用功能”，而是**每个人一套习惯**：有人只会口头描述，有人直接开改，有人不跑测试。Starter Kit 的目标是用一套最小配置，把团队协作里最关键的三件事固化下来：

- 先把问题说清楚（Plan）
- 再把改动做对（TDD/Build Fix）
- 最后把风险兜住（Code Review/Security）
## 目录结构（建议放进仓库，团队共享）

```text
your-repo/
├─ CLAUDE.md
└─ .claude/
   ├─ settings.local.json
   ├─ settings.json
   ├─ rules/
   │  ├─ security.md
   │  ├─ testing.md
   │  └─ coding-style.md
   ├─ agents/
   │  ├─ planner.md
   │  ├─ code-reviewer.md
   │  └─ build-error-resolver.md
   ├─ commands/
   │  ├─ plan.md
   │  ├─ tdd.md
   │  ├─ build-fix.md
   │  └─ code-review.md
   └─ skills/
      ├─ README.md
      └─ backend-patterns/
         └─ SKILL.md
```

## 1) CLAUDE.md：把“项目事实”写进去
这份文件的定位是“项目记忆”，重点写可执行、可验证的信息：

```markdown
# 项目概述
- 技术栈：…
- 目录结构入口：…

## 常用命令（必须准确）
- 安装依赖：…
- 运行测试：…
- 构建：…
- 代码格式化/静态检查：…

## 约束（团队共识）
- 默认先用 Plan Mode 分析，再动代码
- 任何会影响行为的改动：必须补测试/补文档
- 涉及凭证/权限：先安全审查，再合并
```

## 2) rules/：把“底线”拆成小文件
把规则拆到可组合、可审查的粒度。你可以从 3 个底线开始：

- rules/security.md：密钥与输入校验底线（禁硬编码、边界校验、最小权限）。
- rules/testing.md：测试底线（例如 TDD 流程、覆盖率门槛、关键路径必须 E2E）。
- rules/coding-style.md：代码组织底线（文件大小、函数长度、错误处理等）。
这些规则不要“抄一大段”，而是写成**团队真的会执行**的清单。

## 3) agents/：把“换脑子”的任务交给专用角色

社区仓库里最常用的 3 类 agent 是：

- planner：把需求拆成分阶段计划，并明确风险/依赖/验收标准。
- build-error-resolver：一次只修一个构建错误，避免“连锁反应”。
- code-reviewer：对未提交改动做安全与质量审查，按严重程度输出报告。
你不必一开始就做很多 agent，先把“规划/排障/审查”三件事固化下来，协作效率会有明显提升。

## 4) commands/：把流程变成 /plan、/tdd、/build-fix、/code-review

命令文件本质是“流程模板”，其关键点不是内容多，而是**强约束顺序**：

- /plan：必须先复述需求、列风险与步骤，并等待确认后再改代码。
- /tdd：必须先写测试（RED），再实现（GREEN），再重构（REFACTOR）。
- /build-fix：必须增量修复，修一个、跑一次构建。
- /code-review：必须扫描未提交改动，CRITICAL/HIGH 必须阻断合并。
如果你需要命令文件的写法与目录位置，先看：[自定义命令](https://claudecn.com/docs/claude-code/advanced/custom-commands/)。

把这些命令“连成一条闭环”后，团队协作会更稳：见 [团队质量门禁：Plan → TDD → Build Fix → Review](https://claudecn.com/docs/claude-code/workflows/quality-gates/)。

## 5) settings.json：把团队共识接到“工具行为”上

团队共享的 `.claude/settings.json` 建议只放两类东西：

- 权限/安全相关的 allow/deny（比如禁止读取 .env、禁止危险 Bash 模式等）
- Hooks（把“提醒/校验/阻断”自动化）
Hooks 的概念与语法见：[Hooks 系统](https://claudecn.com/docs/claude-code/advanced/hooks/)。如果你想把“会话连续性”和“战略性压缩”也工程化，可以看：[会话连续性与战略压缩](https://claudecn.com/docs/claude-code/workflows/session-continuity/)。

## 6) 把“团队共享”和“个人偏好”分开

很多团队越用越乱，不是因为能力太少，而是因为把“应该进仓库的东西”和“只该留在个人本地的东西”混在了一起。

建议按下面分层：

- 团队共享，进入仓库：CLAUDE.md、rules/、commands/、团队共识型 agents/、经过整理的 skills/
- 个人本地，不进仓库：settings.local.json、个人 API Key、本地 MCP 凭证、个人实验性提示词
- 运行时生成，不进仓库：学习型技能、导入型技能、临时会话产物、个人诊断日志
一个简单判断标准是：如果它需要评审、复用、版本化，就应该进仓库；如果它只属于某个人、某台机器或某次实验，就应该留在本地。

## 7) 给人看的 Skills 索引，比“只有一堆 SKILL.md”更重要

团队里最常见的问题不是“没有 Skills”，而是新成员根本不知道有哪些 Skills、什么时候该启用哪几个。

所以建议至少维护一个 `.claude/skills/README.md`，只做三件事：

- 列出每个 Skill 的 1 句话用途
- 按类别分组
- 给出常见任务的 Skills 组合
例如：

```text
### 新增 API
1. backend-patterns
2. testing-patterns
3. security-review
```

这类“给人看的入口”虽然很小，但对团队 adoption 的影响通常比再多写 5 个 Skill 更大。

## 8) Starter Kit 不要一次装满，按阶段启用

一个更稳妥的启用顺序通常是：

- 先上 CLAUDE.md + rules/security.md + /plan
- 再上 /code-review + code-reviewer
- 再上 testing.md + /tdd + build-error-resolver
- 最后再引入 Hooks、更多 Skills、更多专用 agents
原因很简单：团队最先需要的是“方向一致”和“底线一致”，而不是一开始就把所有自动化和角色系统都堆进来。

## 9) 给 Starter Kit 一个最小维护节奏

如果你希望这套配置长期有效，至少要有一个轻量维护节奏：

- 每周：抽查 1 个目录或 1 类改动，看规则是否真的被遵守
- 每月：同步 CLAUDE.md 里的命令、目录说明和架构变化
- 版本升级后：复查 settings.json、Hooks、命令文件是否还符合当前 Claude Code 行为
Starter Kit 的目标不是“一次写完”，而是让团队的协作方式可以持续校准。

## 参考

无
