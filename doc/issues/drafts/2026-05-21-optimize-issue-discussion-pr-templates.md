---
title: 优化 Issue 初始模板、讨论模板和 PR 模板
labels: skill-system,enhancement,documentation
assignee: 
priority: P1
status: draft
created: 2026-05-21
---

## 描述

当前项目的模板体系较为简单：
- Issue 模板仅有 `tpl_bug.md` 和 `tpl_feature.md` 两个基础 Markdown 模板
- 无 PR 模板（`PULL_REQUEST_TEMPLATE.md`）
- 无 Discussion（讨论）模板

对比业界主流开源项目的模板实践，存在以下差距：
1. **Issue 模板缺少结构化表单**，用户填写随意，信息质量参差不齐
2. **无 PR 模板**，提 PR 时缺乏统一的变更描述和测试计划引导
3. **无 Discussion 模板**，功能讨论、求助等场景缺少结构化引导
4. **所有模板均为 Markdown 格式**，未使用 GitHub 表单（YAML）能力

---

## 网络调研：具体模板示例

### 一、Issue 模板示例

#### 1.1 Markdown 模板 —— VSCode（注释引导式）

**来源**：https://github.com/microsoft/vscode/blob/main/.github/ISSUE_TEMPLATE/bug_report.md

```markdown
---
name: Bug report
about: Create a report to help us improve
title: ''
labels: ''
assignees: ''
---

<!-- ⚠️⚠️ Do Not Delete This! bug_report_template ⚠️⚠️ -->
<!-- Please read our Rules of Conduct: https://opensource.microsoft.com/codeofconduct/ -->
<!-- 🔎 Search existing issues to avoid creating duplicates. -->
<!-- 🧪 Test using the latest Insiders build... -->
Does this issue occur when all extensions are disabled?: Yes/No

- VS Code Version: 
- OS Version: 

Steps to Reproduce:

1. 
2.
```

**特点**：大量使用 HTML 注释 `<!-- -->` 作为填写引导，不干扰最终渲染；预置问题（扩展是否禁用）快速定位根因。

---

#### 1.2 Markdown 模板 —— React（标题前缀 + 预置标签）

**来源**：https://github.com/facebook/react/blob/main/.github/ISSUE_TEMPLATE/bug_report.md

```markdown
---
name: "🐛 Bug Report"
about: Report a reproducible bug or regression.
title: 'Bug: '
labels: 'Status: Unconfirmed'
---

React version:

## Steps To Reproduce

1.
2.

Link to code example:

## The current behavior

## The expected behavior
```

**特点**：`title: 'Bug: '` 强制标题前缀；`labels: 'Status: Unconfirmed'` 自动打标签；明确强调无复现步骤可能被直接关闭。

---

#### 1.3 YAML 表单模板 —— React Compiler（分类复选框 + 必填验证）

**来源**：https://github.com/facebook/react/blob/main/.github/ISSUE_TEMPLATE/compiler_bug_report.yml

```yaml
name: "⚛️ ✨ Compiler bug report"
description: "Report a problem with React Compiler..."
title: "[Compiler Bug]: "
labels: ["Component: Optimizing Compiler", "Type: Bug", "Status: Unconfirmed"]
body:
- type: checkboxes
  attributes:
    label: What kind of issue is this?
    options:
      - label: React Compiler core...
      - label: babel-plugin-react-compiler...
      - label: eslint-plugin-react-hooks...
- type: input
  attributes:
    label: Link to repro
    placeholder: public GitHub repo, or Playground link
  validations:
    required: true
- type: textarea
  attributes:
    label: Repro steps
    placeholder: |
      1. Log in with username/password
      2. Click "Messages"...
  validations:
    required: true
- type: dropdown
  attributes:
    label: How often does this bug happen?
    options:
      - Every time
      - Often
      - Sometimes
      - Only once
  validations:
    required: true
```

**特点**：`checkboxes` 分类问题类型；`input` 强制复现链接；`dropdown` 量化频率；所有关键字段均 `required: true`。

---

#### 1.4 YAML 表单模板 —— Next.js（环境命令自动收集 + 多选下拉）

**来源**：https://github.com/vercel/next.js/blob/canary/.github/ISSUE_TEMPLATE/1.bug_report.yml

```yaml
name: Report an issue
description: Report a Next.js issue.
type: Bug
body:
  - type: markdown
    attributes:
      value: |
        Before opening a new issue, please do a search of existing issues...
        If you need help, start a discussion in the ["Help"](...) section.
  - type: input
    attributes:
      label: Link to the code that reproduces this issue
      description: |
        A link to a public GitHub repository or CodeSandbox minimal reproduction.
        **Skipping this will result in the issue being closed.**
      placeholder: 'https://github.com/user/my-reproduction'
    validations:
      required: true
  - type: textarea
    attributes:
      label: To Reproduce
      placeholder: |
        1. Start the application in development
        2. Click X
        3. Y will happen
    validations:
      required: true
  - type: textarea
    attributes:
      label: Current vs. Expected behavior
      placeholder: 'I expected A, but observed B'
    validations:
      required: true
  - type: textarea
    attributes:
      label: Provide environment information
      description: Please run `next info` and paste the results.
      render: bash
    validations:
      required: true
  - type: dropdown
    attributes:
      label: Which area(s) are affected?
      multiple: true
      options:
        - 'CSS'
        - 'Image (next/image)'
        - 'Middleware'
        - 'Server Actions'
        - ...
    validations:
      required: true
```

**特点**：`render: bash` 格式化环境输出；`multiple: true` 多选下拉精确归类；顶部 Markdown 区块引导用户先去 Discussion 求助。

---

#### 1.5 YAML 表单模板 —— Vite（验证勾选框 + 行为约束）

**来源**：https://github.com/vitejs/vite/blob/main/.github/ISSUE_TEMPLATE/bug_report.yml

```yaml
name: "🐞 Bug report"
description: Report an issue with Vite
labels: [pending triage]
type: Bug
body:
  - type: textarea
    id: bug-description
    attributes:
      label: Describe the bug
      placeholder: I am doing ... What I expect is ... What actually happening is ...
    validations:
      required: true
  - type: input
    id: reproduction
    attributes:
      label: Reproduction
      description: |
        A minimal reproduction is required. If no reproduction is provided after 3 days, it will be auto-closed.
      placeholder: Reproduction URL
    validations:
      required: true
  - type: dropdown
    id: package-manager
    attributes:
      label: Used Package Manager
      options:
        - npm
        - yarn
        - pnpm
        - bun
    validations:
      required: true
  - type: checkboxes
    id: checkboxes
    attributes:
      label: Validations
      description: Before submitting, please make sure you do the following
      options:
        - label: Follow our Code of Conduct
          required: true
        - label: Read the Contributing Guidelines
          required: true
        - label: Check that there isn't already an issue
          required: true
        - label: Make sure this is a Vite issue and not framework-specific
          required: true
        - label: The provided reproduction is minimal
          required: true
```

**特点**：`checkboxes` 带 `required: true` 选项，用户必须勾选才能提交；复现链接缺失明确告知 3 天后自动关闭。

---

#### 1.6 YAML 表单模板 —— Prisma（严重程度 + 频率 + 回归检测）

**来源**：https://github.com/prisma/prisma/blob/main/.github/ISSUE_TEMPLATE/bug_report.yml

```yaml
name: Bug report
labels: ['kind/bug']
body:
  - type: dropdown
    id: severity
    attributes:
      label: Severity
      options:
        - '🚨 Critical: Data loss, app crash, security issue'
        - '⚠️ Major: Breaks core functionality'
        - '🔹 Minor: Unexpected behavior'
    validations:
      required: true
  - type: dropdown
    id: frequency
    attributes:
      label: Frequency
      options:
        - 'Consistently reproducible'
        - 'Intermittent / Random'
        - 'Happened once'
    validations:
      required: true
  - type: textarea
    id: regression
    attributes:
      label: Is this a regression?
      value: |
        <!-- e.g., "Yes, last worked in Prisma 5.1.0" -->
    validations:
      required: true
  - type: textarea
    id: workaround
    attributes:
      label: Workaround
      value: |
        <!-- e.g., "Downgrading fixes it" or "No workaround" -->
    validations:
      required: true
```

**特点**：`Severity` + `Frequency` 双下拉，帮助维护者快速判断优先级；强制询问是否回归 + 是否有 workaround，减少重复提问。

---

#### 1.7 YAML 表单模板 —— Rust（当前输出 vs 期望输出对比）

**来源**：https://github.com/rust-lang/rust/blob/main/.github/ISSUE_TEMPLATE/diagnostics.yaml

```yaml
name: Diagnostic issue
description: Create a bug report for rustc's error output
labels: ["A-diagnostics", "T-compiler"]
body:
  - type: textarea
    id: code
    attributes:
      label: Code
      render: Rust
    validations:
      required: true
  - type: textarea
    id: output
    attributes:
      label: Current output
      render: Shell
    validations:
      required: true
  - type: textarea
    id: desired-output
    attributes:
      label: Desired output
      render: Shell
    validations:
      required: false
  - type: textarea
    id: version
    attributes:
      label: Rust Version
      placeholder: |
        $ rustc --version --verbose
        rustc 1.XX.Y ...
      render: Shell
    validations:
      required: true
```

**特点**：`render: Rust` / `render: Shell` 代码高亮；"Current output" 与 "Desired output" 对比式结构，特别适合编译器/诊断类问题。

---

#### 1.8 YAML 表单模板 —— shadcn/ui（组件级影响范围）

**来源**：https://github.com/shadcn-ui/ui/blob/main/.github/ISSUE_TEMPLATE/bug_report.yml

```yaml
name: "Bug report"
title: '[bug]: '
labels: ["bug"]
body:
  - type: input
    id: components-affected
    attributes:
      label: Affected component/components
      placeholder: ex. Button, Checkbox...
    validations:
      required: true
  - type: input
    id: codesandbox-stackblitz
    attributes:
      label: Codesandbox/StackBlitz link
      description: |
        > [!CAUTION]
        > If you skip this, this issue might be labeled with `please add a reproduction` and closed.
  - type: checkboxes
    id: terms
    attributes:
      label: Before submitting
      options:
        - label: I've made research efforts and searched the documentation
          required: true
        - label: I've searched for existing issues
          required: true
```

**特点**：组件库特有字段 "Affected component/components"；`[!CAUTION]` 警告区块强调复现的重要性。

---

### 二、Discussion 模板示例

#### 2.1 Next.js —— Feature Request Discussion（Goals / Non-Goals / Background / Proposal）

**来源**：https://github.com/vercel/next.js/blob/canary/.github/DISCUSSION_TEMPLATE/ideas.yml

```yaml
body:
  - type: textarea
    attributes:
      label: Goals
      description: Short list of what the feature request aims to address?
      value: |
        1.
        2.
        3.
    validations:
      required: true
  - type: textarea
    attributes:
      label: Non-Goals
      description: Short list of what the feature request does not aim to address?
      value: |
        1.
        2.
        3.
    validations:
      required: false
  - type: textarea
    attributes:
      label: Background
      description: Discuss prior art, why do you think this feature is needed?
    validations:
      required: true
  - type: textarea
    attributes:
      label: Proposal
      description: How should this feature be implemented? Are you interested in contributing?
    validations:
      required: true
```

**特点**：`Goals / Non-Goals` 双栏结构明确需求边界，避免范围蔓延；`Background` 强制做 prior art 调研；`Proposal` 鼓励社区贡献。

---

#### 2.2 Next.js —— Help Discussion

**来源**：https://github.com/vercel/next.js/blob/canary/.github/DISCUSSION_TEMPLATE/help.yml

```yaml
body:
  - type: textarea
    attributes:
      label: Summary
      description: What do you need help with?
    validations:
      required: true
  - type: textarea
    attributes:
      label: Additional information
      description: Any code snippets, error messages... (`next info`)
      render: js
  - type: input
    attributes:
      label: Example
      description: A link to a minimal reproduction is helpful!
```

**特点**：简洁三字段，降低求助门槛；`render: js` 鼓励粘贴代码而非截图。

---

### 三、PR 模板示例

#### 3.1 React —— 强调"如何测试此变更"

**来源**：https://github.com/facebook/react/blob/main/.github/PULL_REQUEST_TEMPLATE.md

```markdown
<!--
  Before submitting:
  1. Fork the repository and create your branch from `main`.
  2. Run `yarn` in the repository root.
  3. If you've fixed a bug or added code that should be tested, add tests!
  4. Ensure the test suite passes (`yarn test`).
  5. Run `yarn test --prod` to test in production environment.
  6. Format your code with prettier (`yarn prettier`).
  7. Make sure your code lints (`yarn lint`).
  8. Run Flow type checks (`yarn flow`).
  9. If you haven't already, complete the CLA.
-->

## Summary
<!-- Explain the motivation for making this change... -->

## How did you test this change?
<!--
  Demonstrate the code is solid. Example: The exact commands you ran and their output.
  If you leave this empty, your PR will very likely be closed.
-->
```

**特点**：头部 9 步贡献清单注释，**留空 `How did you test this change?` 可能导致 PR 被直接关闭**。

---

#### 3.2 Kubernetes —— PR 类型标记 + 发布说明

**来源**：https://github.com/kubernetes/kubernetes/blob/master/.github/PULL_REQUEST_TEMPLATE.md

```markdown
#### What type of PR is this?
<!--
/kind bug
/kind feature
/kind cleanup
/kind documentation
-->

#### What this PR does / why we need it:

#### Which issue(s) this PR is related to:
<!-- Fixes #<issue number> -->

#### Special notes for your reviewer:

#### Does this PR introduce a user-facing change?
```release-note

```

#### Additional documentation e.g., KEPs...
```docs

```
```

**特点**：`kind` 标记标准化（与 Prow 机器人集成）；`release-note` 代码块强制填写发布说明；`docs` 代码块关联文档变更。

---

#### 3.3 Astro —— Changes / Testing / Docs 三段式

**来源**：https://github.com/withastro/astro/blob/main/.github/PULL_REQUEST_TEMPLATE.md

```markdown
## Changes
- What does this change?
- Be short and concise. Bullet points can help!
- Don't forget a changeset! Run `pnpm changeset`.

## Testing
<!-- How was this change tested? -->
<!-- DON'T DELETE THIS SECTION! If no tests added, explain why. -->

## Docs
<!-- Could this affect a user's behavior? We probably need to update docs! -->
<!-- DON'T DELETE THIS SECTION! If no docs added, explain why.-->
```

**特点**：极简三段式，强调 `changeset`（版本变更日志）；`DON'T DELETE THIS SECTION!` 强制保留每个区块。

---

#### 3.4 Vite —— 注释引导式（不渲染到最终 PR）

**来源**：https://github.com/vitejs/vite/blob/main/.github/PULL_REQUEST_TEMPLATE.md

```markdown
<!--
- What is this PR solving? Write a clear and concise description.
- Reference the issues it solves (e.g. `fixes #123`).
- What other alternatives have you explored?
- Are there any parts you think require more attention from reviewers?

Also, please make sure you do the following:
- Read the Contributing Guidelines...
- Check that there isn't already a PR that solves the problem...
- Update the corresponding documentation if needed.
- Include relevant tests that fail without this PR but pass with it.
  If the tests are not included, explain why.
-->
```

**特点**：**纯注释模板**，PR 描述初始为空白，完全由贡献者自行填写；避免模板残留污染 PR 正文。

---

### 四、Issue 配置示例（config.yml）

#### 4.1 Next.js —— 引导 Feature Request 到 Discussion

**来源**：https://github.com/vercel/next.js/blob/canary/.github/ISSUE_TEMPLATE/config.yml

```yaml
blank_issues_enabled: false
contact_links:
  - name: Ask a question or discuss a topic
    url: https://github.com/vercel/next.js/discussions
    about: Ask questions or discuss with other Next.js users in discussions.
  - name: Feature or documentation request
    url: https://github.com/vercel/next.js/discussions/new?category=ideas
    about: Open a feature or documentation request in discussions.
```

**特点**：`blank_issues_enabled: false` 禁止空白 Issue；`contact_links` 将 Feature Request 和求助全部引导到 Discussion。

---

## 调研结论

### 业界趋势

1. **YAML 表单化**：大型项目（React、Next.js、Astro、Vite、Prisma、Rust）已全面转向 `.yml` 表单模板
2. **Feature Request 移入 Discussion**：Next.js、React 等将功能请求从 Issue 转移到 GitHub Discussion，减少 Issue 噪音
3. **PR 模板强调"测试证明"**：React PR 模板明确要求 `How did you test this change?`，留空可能被关闭
4. **检查清单（Checklist）普及**：Vite、shadcn-ui 等使用 `checkboxes` 强制用户确认前置条件
5. **环境信息自动化**：Next.js、Astro 要求运行 CLI 命令（`next info`、`astro info`）并粘贴输出
6. **严重程度分级**：Prisma 使用 Severity + Frequency 双下拉帮助维护者快速判断优先级

### Markdown vs YAML 对比

| 维度 | Markdown 模板 | YAML 表单模板 |
|------|--------------|---------------|
| 用户填写体验 | 自由编辑，容易遗漏 | 结构化输入，必填校验 |
| 信息质量 | 参差不齐 | 较高，强制关键字段 |
| 维护复杂度 | 低 | 中（需学习 YAML 语法） |
| GitHub 原生支持 | 基础支持 | 完整支持（下拉、复选框、文本域） |
| 适用场景 | 简单项目、Agent 生成 | 多人协作、社区贡献 |

---

## 方案设计

### 方案 A：全面 YAML 表单化（对标 Next.js / Astro）

全面采用 GitHub YAML 表单，将 Issue 模板、Discussion 模板全部表单化。

**Issue 模板（YAML）：**
- `bug_report.yml` —— 复现链接（必填）、复现步骤、当前/期望行为、环境信息、影响范围
- `feature_request.yml` —— 需求背景、期望行为、实现思路、是否愿意贡献 PR
- `documentation.yml` —— 文档位置、问题描述、建议修改

**Discussion 模板（YAML）：**
- `ideas.yml` —— Goals / Non-Goals / Background / Proposal（对标 Next.js）
- `help.yml` —— 问题描述、已尝试方案、环境信息、最小复现

**PR 模板（Markdown + 检查清单）：**
- 变更摘要、关联 Issue、Test Plan（检查清单）、截图/日志、Checklist（lint/test/doc）

**优点**：
- 信息质量最高，强制字段减少来回沟通
- 业界主流，Next.js、Astro、React 均采用
- Discussion 模板可分流 Feature Request，减轻 Issue 负担

**缺点**：
- Agent 自动生成 Issue/PR 时需额外处理 YAML 表单字段
- 需要学习 GitHub 表单语法

---

### 方案 B：Markdown + YAML 混合（平衡易用与规范）⭐

**Issue 模板：**
- Agent 生成的 Issue 保持现有 Markdown 模板（便于程序化填充）
- 面向人类用户的 GitHub Issue 使用 YAML 表单模板
- 双轨并行：Markdown 用于 Agent 草稿，YAML 用于最终发布

**Discussion 模板（YAML）：**
- 仅保留 `ideas.yml`，用于功能讨论

**PR 模板（Markdown，增强版）：**
- 在现有 PR 格式基础上，增加：
  - 头部 HTML 注释引导（对标 VSCode）
  - 更详细的 Test Plan 区块
  - 关联 Issue 自动关闭说明

**优点**：
- 兼顾 Agent 自动化和人类用户友好性
- 改动成本最低，可渐进式升级
- 保留现有 Markdown 模板资产

**缺点**：
- 双轨维护成本稍高
- 对人类用户的强制力不如纯 YAML

---

### 方案 C：轻量统一 Markdown（最小改动）

全部使用 Markdown 模板，不引入 YAML 表单，但统一增强现有模板结构。

**Issue 模板增强：**
- `tpl_bug.md`：增加"复现链接"、"影响范围"、"优先级自评"区块
- `tpl_feature.md`：增加"Goals / Non-Goals"、"替代方案"、"是否愿意实现"区块
- 新增 `tpl_discussion.md`：用于功能讨论前的方案预研

**PR 模板（新建）：**
- 简单 Markdown，包含：Summary / Changes / Test Plan / Checklist / Closes #N

**优点**：
- 实现最简单，无需学习 YAML
- Agent 生成最方便，直接字符串拼接
- 与现有 `001-2-issue` skill 草稿格式完全兼容

**缺点**：
- 无法强制必填字段
- 信息质量依赖填写者自觉
- 与业界趋势有一定差距

---

## 决策对比

| 决策点 | 方案 A | 方案 B ⭐ | 方案 C |
|--------|--------|----------|--------|
| Issue 模板格式 | YAML 表单 | Markdown（Agent）+ YAML（人类） | Markdown |
| PR 模板 | Markdown 增强 | Markdown 增强 | Markdown 基础 |
| Discussion 模板 | YAML 表单 | YAML 表单（仅 ideas） | Markdown |
| 对 Agent 友好度 | 中 | **高** | 最高 |
| 对人类用户友好度 | **最高** | 高 | 中 |
| 维护成本 | 中 | 中 | **低** |
| 信息强制力 | 强 | 中 | 弱 |

---

## 涉及文件

| 文件 | 改动说明 |
|------|---------|
| `.github/ISSUE_TEMPLATE/bug_report.yml` | 新增人类用户 Bug 报告 YAML 表单 |
| `.github/ISSUE_TEMPLATE/feature_request.yml` | 新增人类用户 Feature Request YAML 表单 |
| `.github/DISCUSSION_TEMPLATE/ideas.yml` | 新增功能讨论 YAML 表单 |
| `.github/PULL_REQUEST_TEMPLATE.md` | 新增 PR 模板 |
| `doc/issues/templates/tpl_bug.md` | 增强：增加复现链接、影响范围等 |
| `doc/issues/templates/tpl_feature.md` | 增强：增加 Goals/Non-Goals、替代方案等 |
| `.github/ISSUE_TEMPLATE/config.yml` | 新增：配置 Issue 创建引导，指向 Discussion |

## 验收标准

- [ ] 确定最终方案（A / B / C）
- [ ] PR 模板已创建并包含：Summary、Changes、Test Plan、Checklist、关联 Issue
- [ ] Issue YAML 表单（如选 A/B）已配置必填字段验证
- [ ] Discussion 模板（如选 A/B）已配置 Goals/Background/Proposal 结构
- [ ] 现有 Markdown 模板已同步增强
- [ ] `config.yml` 已配置（引导 Feature Request 到 Discussion）

## 发布记录

- Issue #17: https://github.com/xiaoheiDTF/claude-hk/issues/17 (发布于 2026-05-21 22:31)

