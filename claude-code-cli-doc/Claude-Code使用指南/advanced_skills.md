# Claude-Code / Advanced / Skills

> 来源: claudecn.com

# Skills（技能）

Skills 用来扩展 Claude Code 的能力：你创建一个 `SKILL.md`（以及可选的支持文件），Claude 就会把它当作“可复用的工具箱”。Skill 既可以被 Claude 在合适场景**自动加载**，也可以由你通过 `/skill-name` **手动调用**。

- 内置命令（例如 /help、/compact）请参考「交互模式」文档（官方）。
- 自定义斜杠命令已经合并进 Skills：.claude/commands/review.md 与 .claude/skills/review/SKILL.md 都会提供 /review，用法等价；已有 .claude/commands/ 仍可继续工作。推荐优先用 Skills，因为它支持“支持文件目录”“调用控制”等增强能力。

Claude Code 的 Skills 基于 Agent Skills 开放标准，并在此基础上扩展了：

- 调用控制（谁可以触发、是否允许模型自动触发）
- 子代理执行（把 Skill 作为子代理任务在隔离上下文运行）
- 动态上下文注入（在发送给 Claude 前先执行命令，把输出注入提示）
## 进一步阅读

- 从工作原理开始：/docs/claude-code/advanced/agent-loop/
- Skills（按需加载的知识包）：/docs/claude-code/advanced/agent-loop/v4-skills/
## 快速开始：创建第一个 Skill

这个示例创建一个 `explain-code` Skill，让 Claude 在解释代码时**始终包含类比 + ASCII 图**。默认规则下，它既能被 Claude 自动加载，也能被你手动调用。

### 1) 创建 Skill 目录

个人 Skills 存放在 `~/.claude/skills/`，跨项目可用：

```bash
mkdir -p ~/.claude/skills/explain-code
```

### 2) 编写 SKILL.md
`SKILL.md` 由两部分组成：

- YAML frontmatter（--- 之间）：决定如何发现与调用
- Markdown 内容：Skill 被调用时 Claude 应遵循的指令
创建 `~/.claude/skills/explain-code/SKILL.md`：

```yaml
---
name: explain-code
description: 用“类比 + ASCII 图”解释代码。适用于讲解代码库、解释执行流程，或回答“这段代码是怎么工作的？”
---

解释代码时，始终包含：

1. **类比开头**：将代码比作日常生活中的事物
2. **画图说明**：使用 ASCII 艺术展示流程、结构或关系
3. **逐步讲解**：解释代码执行的每一步
4. **指出陷阱**：常见的错误或误解是什么？

保持解释口语化。对于复杂概念，使用多个类比。
```

### 3) 测试 Skill

- 让 Claude 自动触发：提问一个匹配 description 的问题（例如“这段代码是怎么工作的？”）
- 手动调用：直接执行 /explain-code src/auth/login.ts
## Skills 放在哪里（作用域与优先级）
存放位置决定谁可以使用：

| 位置 | 路径 | 适用范围 |
| --- | --- | --- |
| 企业/托管 | 参考「托管设置」 | 组织内所有用户 |
| 个人 | `~/.claude/skills//SKILL.md` | 你的所有项目 |
| 项目 | `.claude/skills//SKILL.md` | 仅当前仓库 |
| 插件 | `
/skills//SKILL.md` | 启用该插件的项目 |
同名冲突时：项目 Skill 会覆盖个人 Skill；如果同时存在 `.claude/commands/<name>.md` 与 `.claude/skills/<name>/SKILL.md`，Skill 优先。

### 嵌套目录自动发现（适合 monorepo）

当你在子目录（例如 `packages/frontend/`）中工作时，Claude Code 会自动发现 `packages/frontend/.claude/skills/` 下的 Skills，便于在 monorepo 中为不同包定制规则。

## Skill 目录结构与支持文件

一个 Skill 是一个目录，入口文件固定为 `SKILL.md`：

```
my-skill/
├── SKILL.md           # 必需：入口与主要指令
├── reference.md       # 可选：大段参考资料（需要时再加载）
├── examples/          # 可选：示例输出
└── scripts/           # 可选：可执行脚本（执行，不自动注入上下文）
```

把大段参考资料拆到单独文件，并在 `SKILL.md` 里链接它们，能让上下文更“轻”，也更容易维护。

## Frontmatter 参考（常用字段）

`SKILL.md` 顶部的 frontmatter 用来控制调用行为（字段均可选，但建议写 `description`）：

```yaml
---
name: my-skill
description: 这个 Skill 做什么，何时使用
disable-model-invocation: true
allowed-tools: Read, Grep
---
```

| 字段 | 说明 |
| --- | --- |
| `name` | Skill 名称（省略则使用目录名）；只允许小写字母/数字/连字符（最长 64） |
| `description` | 推荐填写；Claude 会用它判断何时自动加载 |
| `argument-hint` | 自动补全时显示的参数提示（例如 `[issue-number]`、`[file] [format]`） |
| `disable-model-invocation` | 设为 `true` 后，Claude 不会自动触发，只能由你通过 `/name` 手动触发 |
| `user-invocable` | 设为 `false` 后，从 `/` 菜单隐藏（更适合“背景知识”类 Skill） |
| `allowed-tools` | Skill 激活时允许“无需再次确认”可用的工具范围 |
| `model` | Skill 激活时使用的模型 |
| `context` | 设为 `fork` 时在隔离的子代理上下文运行 |
| `agent` | `context: fork` 时选择子代理类型（例如 `Explore`、`Plan`） |
| `hooks` | 仅对该 Skill 生效的 hooks（格式见 Hooks 文档） |

### 字符串替换（参数与会话 ID）

| 变量 | 含义 |
| --- | --- |
| `$ARGUMENTS` | 调用 Skill 时传入的全部参数（若内容里未出现，Claude Code 会在末尾追加 `ARGUMENTS: ...`） |
| `${CLAUDE_SESSION_ID}` | 当前会话 ID（便于做日志/产物命名） |

## 控制“谁”可以触发 Skill
默认情况下：你可以 `/skill-name` 调用；Claude 也可以在匹配场景自动加载。你可以用两个字段收敛触发权：

- disable-model-invocation: true：只允许你手动触发（适合 /deploy、/commit 这类有副作用的流程）
- user-invocable: false：只允许 Claude 自动触发，但从菜单隐藏（适合背景知识）
| Frontmatter | 你能手动调用 | Claude 能自动调用 | 何时进入上下文 |
| --- | --- | --- | --- |
| 默认 | 是 | 是 | 描述常驻；被触发时加载完整内容 |
| `disable-model-invocation: true` | 是 | 否 | 描述不进入上下文；仅你触发时加载 |
| `user-invocable: false` | 否 | 是 | 描述常驻；被触发时加载完整内容 |

`user-invocable` 只影响菜单可见性，不等同于“禁止程序化触发”。如果要彻底避免 Claude 自动触发，请使用 `disable-model-invocation: true`。

## 高级模式

### 动态上下文注入（先执行，再注入）
在 Skill 内容里使用 `!`command` `占位符，Claude Code 会在把 Skill 发送给 Claude **之前**执行命令，并用输出替换占位符；Claude 看到的是“已注入的数据”，不是命令本身。

```yaml
---
name: pr-summary
description: 汇总一个 PR 的改动（使用 GitHub CLI 拉取实时信息）
context: fork
agent: Explore
allowed-tools: Bash(gh:*)
---

## Pull request context
- PR diff: !`gh pr diff`
- PR comments: !`gh pr view --comments`
- Changed files: !`gh pr diff --name-only`

## Your task
基于以上信息，用中文给出这次 PR 的摘要、风险点与建议验证清单。
```

如果希望 Skill 默认启用扩展思考，把 `ultrathink` 一词放进 Skill 内容即可（可放在任意位置）。

### 在子代理中运行 Skill（隔离上下文）
当你希望 Skill 不污染主对话（或需要专门的 agent 类型）时，可以设置：

```yaml
---
context: fork
agent: Explore
---
```

`context: fork` 更适合“有明确任务步骤”的 Skill；如果只是风格/规范类的背景知识，放到 `context: fork` 往往得不到有效产出。

## 下一步
[自定义命令从 commands 过渡到 skills
](../custom-commands/)[配置参考权限、hooks、环境变量
](../../reference/settings/)

### 在 Subagent 上下文中运行 Skill
使用 `context: fork` 和 `agent` 在派生的 Subagent 中运行 Skill，保持独立上下文。

---

## 实用示例

### 简单 Skill（单文件）

生成清晰提交信息的 Skill：

```yaml
---
name: generating-commit-messages
description: 从 git diff 生成清晰的提交信息。编写提交信息或审查暂存更改时使用。
---

# 生成提交信息

## 指令

1. 运行 `git diff --staged` 查看更改
2. 我会建议一个提交信息，包括：
   - 50 字符以内的摘要
   - 详细描述
   - 受影响的组件

## 最佳实践

- 使用现在时态
- 解释做了什么和为什么，而非怎么做
```

### 多文件 Skill
PDF 处理 Skill，使用渐进式加载和工具限制：

```
pdf-processing/
├── SKILL.md              # 概览和快速开始
├── FORMS.md              # 表单字段映射和填写指令
├── REFERENCE.md          # pypdf 和 pdfplumber 的 API 详情
└── scripts/
    ├── fill_form.py      # 填充表单字段的工具
    └── validate.py       # 检查 PDF 必填字段
```

**`SKILL.md`**：

```yaml
---
name: pdf-processing
description: 提取文本、填写表单、合并 PDF。处理 PDF 文件、表单或文档提取时使用。需要 pypdf 和 pdfplumber 包。
allowed-tools: Read, Bash(python:*)
---

# PDF 处理

## 快速开始

提取文本：
```python
import pdfplumber
with pdfplumber.open("doc.pdf") as pdf:
    text = pdf.pages[0].extract_text()
```

表单填写见 `FORMS.md`。
详细 API 参考见 `REFERENCE.md`。

## 依赖

需要在环境中安装：

```bash
pip install pypdf pdfplumber
```

```
---

## 与其他功能的对比

| 使用场景 | 推荐选择 | 说明 |
|----------|----------|------|
| 给 Claude 专业知识 | **Skills** | Claude 自动在需要时使用 |
| 创建可复用提示 | [斜杠命令](/docs/claude-code/advanced/custom-commands/) | 需要输入 `/命令` 运行 |
| 设置项目级指令 | [CLAUDE.md](/docs/claude-code/workflows/context-management/) | 每次对话自动加载 |
| 委派任务到独立上下文 | [Subagents](/docs/claude-code/advanced/subagents/) | 独立工具访问 |
| 事件触发脚本 | [Hooks](/docs/claude-code/advanced/hooks/) | 特定工具事件触发 |
| 连接外部工具和数据源 | [MCP 服务器](/docs/claude-code/advanced/mcp-servers/) | Claude 按需调用 |

**Skills vs MCP**：Skills 告诉 Claude *如何*使用工具；MCP *提供*工具。例如，MCP 服务器连接 Claude 到数据库，Skill 教 Claude 你的数据模型和查询模式。

---

## 故障排除

### Skill 不触发

`description` 字段是 Claude 决定是否使用 Skill 的关键。模糊的描述如"帮助处理文档"无法让 Claude 匹配到相关请求。

**好的描述**回答两个问题：
1. 这个 Skill 做什么？列出具体功能
2. Claude 应该何时使用？包含用户会提到的触发词

```yaml
description: 从 PDF 文件提取文本和表格，填写表单，合并文档。处理 PDF 文件或用户提到 PDF、表单、文档提取时使用。
```

### Skill 不加载
**检查文件路径**。必须在正确目录，文件名必须精确为 `SKILL.md`（区分大小写）：

| 类型 | 路径 |
| --- | --- |
| 个人 | `~/.claude/skills/my-skill/SKILL.md` |
| 项目 | `.claude/skills/my-skill/SKILL.md` |
| 插件 | `skills/my-skill/SKILL.md`（插件目录内） |

**检查 YAML 语法**。元数据必须以 `---` 开始和结束，第一行前不能有空行，使用空格缩进（不是 Tab）。

**运行调试模式**：`claude --debug` 查看 Skill 加载错误。

### 多个 Skill 冲突

如果 Claude 使用了错误的 Skill 或在相似 Skill 间混淆，说明描述太相似。使用具体的触发词区分：

不要两个 Skill 都写"数据分析"，而要区分：一个用于"Excel 文件和 CRM 导出的销售数据"，另一个用于"日志文件和系统指标"。

---

## 延伸阅读

- 站内博客：/blog/understanding-agent-skills/
- 站内博客：/blog/claude-skills-architecture/
- 站内博客：/blog/claude-skills-landing-guide/
- 文档入口：/docs/agent-skills/
## 下一步
[Subagents创建专用子代理处理特定任务
](../subagents/)[Hooks 系统在事件触发时自动执行脚本
](../hooks/)[MCP 服务器连接外部工具和数据源
](../mcp-servers/)
