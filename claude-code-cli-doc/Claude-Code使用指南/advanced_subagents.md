# Claude-Code / Advanced / Subagents

> 来源: claudecn.com

# Subagents 子代理

Subagents（子代理）是专门处理特定类型任务的 AI 助手。每个 Subagent 在独立的上下文窗口中运行，拥有自定义系统提示、特定工具访问权限和独立权限设置。

## 为什么使用 Subagents

Subagents 帮助你：

- 保留上下文 - 将探索和实现保留在独立上下文中，不污染主对话
- 强制约束 - 限制 Subagent 可使用的工具
- 复用配置 - 用户级 Subagent 跨项目可用
- 专业化行为 - 使用专注的系统提示处理特定领域
- 控制成本 - 将任务路由到更快、更便宜的模型如 Haiku
---

## 内置 Subagents

Claude Code 包含以下内置 Subagent，Claude 会在适当时自动使用：

### Explore

快速、只读的代理，优化用于搜索和分析代码库。

- 模型：Haiku（快速、低延迟）
- 工具：只读工具（禁止 Write 和 Edit）
- 用途：文件发现、代码搜索、代码库探索
Claude 在需要搜索或理解代码库但不做修改时委托给 Explore。

调用时可指定彻底程度：**quick**（快速定向查找）、**medium**（平衡探索）、**very thorough**（全面分析）。

### Plan

计划模式下使用的研究代理，在呈现计划之前收集上下文。

- 模型：继承主对话模型
- 工具：只读工具
- 用途：规划阶段的代码库研究
### general-purpose

通用代理，处理需要探索和操作的复杂多步任务。

- 模型：继承主对话模型
- 工具：所有工具
- 用途：复杂研究、多步操作、代码修改
### 其他辅助代理

Claude Code 还包含一些特定任务的辅助代理，通常自动调用：

| 代理 | 模型 | 用途 |
| --- | --- | --- |
| Bash | 继承 | 在独立上下文中运行终端命令 |
| statusline-setup | Sonnet | 运行 `/statusline` 配置状态栏时使用 |
| Claude Code Guide | Haiku | 回答关于 Claude Code 功能的问题 |

---

## 创建自定义 Subagent

### 使用 /agents 命令

推荐使用交互式界面创建：

```
> /agents
```

选择 **Create new agent**，然后选择 **User-level**（用户级，跨项目可用）或 **Project-level**（项目级）。

#### 使用 Claude 生成

选择 **Generate with Claude**，描述你想要的 Subagent：

```
一个代码改进代理，扫描文件并建议可读性、性能和最佳实践的改进。
它应该解释每个问题，展示当前代码，并提供改进版本。
```

Claude 会生成系统提示和配置。按 `e` 在编辑器中打开进行自定义。

#### 选择工具和模型

- 工具：选择 Subagent 可用的工具。只读审查器只需 Read-only tools
- 模型：选择 Sonnet（平衡能力和速度）、Opus（最强）、Haiku（最快）或 inherit（继承主对话）
#### 选择颜色

为 Subagent 选择背景颜色，帮助在 UI 中识别正在运行的是哪个 Subagent。

### 手动创建

Subagent 是带有 YAML 元数据的 Markdown 文件：

```markdown
---
name: code-reviewer
description: 审查代码质量和最佳实践
tools: Read, Glob, Grep
model: sonnet
---

你是代码审查员。被调用时，分析代码并提供
关于质量、安全性和最佳实践的具体、可操作的反馈。
```

元数据定义配置，正文成为指导 Subagent 行为的系统提示。

---

## 存放位置

| 位置 | 范围 | 优先级 |
| --- | --- | --- |
| `--agents` CLI 参数 | 当前会话 | 最高 |
| `.claude/agents/` | 当前项目 | 高 |
| `~/.claude/agents/` | 你的所有项目 | 中 |
| 插件的 `agents/` 目录 | 启用该插件的项目 | 最低 |

同名时优先级高的生效。

---

## 配置详解

### 元数据字段

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `name` | 是 | 唯一标识符，小写字母和连字符 |
| `description` | 是 | 描述何时应委托给此 Subagent |
| `tools` | 否 | 可用工具列表，省略则继承所有工具 |
| `disallowedTools` | 否 | 禁用的工具 |
| `model` | 否 | `sonnet`、`opus`、`haiku` 或 `inherit`，默认 `sonnet` |
| `permissionMode` | 否 | 权限模式 |
| `skills` | 否 | 要加载的 Skills 列表 |
| `hooks` | 否 | 生命周期钩子 |

### 选择模型

- sonnet：平衡能力和速度
- opus：最强能力
- haiku：最快最便宜
- inherit：与主对话使用相同模型
### 权限模式

| 模式 | 行为 |
| --- | --- |
| `default` | 标准权限检查，有提示 |
| `acceptEdits` | 自动接受文件编辑 |
| `dontAsk` | 自动拒绝权限提示 |
| `bypassPermissions` | 跳过所有权限检查（谨慎使用） |
| `plan` | 计划模式（只读探索） |

---

## 前台与后台执行

### 前台 Subagent

阻塞主对话直到完成。权限提示和澄清问题会传递给你。

### 后台 Subagent

并发运行，你可以继续工作。继承父级权限，自动拒绝未预授权的内容。

```
> 在后台运行这个任务
```

或按 **Ctrl+B** 将运行中的任务放到后台。

后台 Subagent 完成时，结果返回主对话。运行多个返回详细结果的 Subagent 可能占用大量上下文。

---

## 实用模式

### 隔离高输出操作
最有效的用途之一是隔离产生大量输出的操作。运行测试、获取文档或处理日志文件会消耗大量上下文。

```
> 使用 Subagent 运行测试套件，只报告失败的测试及错误信息
```

详细输出留在 Subagent 的上下文中，只有相关摘要返回主对话。

### 并行研究

对于独立的调查，启动多个 Subagent 同时工作：

```
> 使用独立的 Subagent 并行研究认证、数据库和 API 模块
```

每个 Subagent 独立探索各自领域，然后 Claude 综合发现。

### 链式 Subagents

对于多步工作流，让 Claude 按顺序使用 Subagents：

```
> 使用 code-reviewer Subagent 找出性能问题，然后使用 optimizer Subagent 修复它们
```

---

## 经典示例

### 代码审查员
只读 Subagent，审查代码但不修改：

```markdown
---
name: code-reviewer
description: 专业代码审查员。代码修改后主动审查质量、安全性和可维护性。
tools: Read, Grep, Glob, Bash
model: inherit
---

你是确保高标准代码质量和安全性的资深代码审查员。

被调用时：
1. 运行 git diff 查看最近更改
2. 关注修改的文件
3. 立即开始审查

审查清单：
- 代码清晰可读
- 函数和变量命名良好
- 没有重复代码
- 错误处理得当
- 没有暴露的密钥或 API 密钥
- 实现了输入验证
- 测试覆盖良好
- 考虑了性能

按优先级组织反馈：
- 严重问题（必须修复）
- 警告（应该修复）
- 建议（考虑改进）

包含如何修复问题的具体示例。
```

## 进一步阅读

- 先理解整体控制流：/docs/claude-code/advanced/agent-loop/
- 上下文隔离（子代理机制）：/docs/claude-code/advanced/agent-loop/v3-subagents/
### 调试专家
可以分析和修复问题的 Subagent：

```markdown
---
name: debugger
description: 错误、测试失败和意外行为的调试专家。遇到任何问题时主动使用。
tools: Read, Edit, Bash, Grep, Glob
---

你是专注于根因分析的调试专家。

被调用时：
1. 捕获错误信息和堆栈跟踪
2. 识别复现步骤
3. 定位失败位置
4. 实现最小修复
5. 验证解决方案有效

调试过程：
- 分析错误信息和日志
- 检查最近代码更改
- 形成并测试假设
- 添加策略性调试日志
- 检查变量状态

对于每个问题，提供：
- 根因解释
- 支持诊断的证据
- 具体代码修复
- 测试方法
- 预防建议

专注于修复根本问题，而非症状。
```

### 数据科学家
领域特定的数据分析 Subagent：

```markdown
---
name: data-scientist
description: SQL 查询、BigQuery 操作和数据洞察的数据分析专家。数据分析任务和查询时主动使用。
tools: Bash, Read, Write
model: sonnet
---

你是专注于 SQL 和 BigQuery 分析的数据科学家。

被调用时：
1. 理解数据分析需求
2. 编写高效 SQL 查询
3. 在适当时使用 BigQuery 命令行工具 (bq)
4. 分析和总结结果
5. 清晰呈现发现

关键实践：
- 编写带适当过滤器的优化 SQL 查询
- 使用适当的聚合和连接
- 包含解释复杂逻辑的注释
- 格式化结果以便阅读
- 提供数据驱动的建议

对于每个分析：
- 解释查询方法
- 记录任何假设
- 突出关键发现
- 根据数据建议下一步

始终确保查询高效且成本有效。
```

### 数据库查询验证器
使用 PreToolUse 钩子验证只读查询的 Subagent：

```markdown
---
name: db-reader
description: 执行只读数据库查询。分析数据或生成报告时使用。
tools: Bash
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "./scripts/validate-readonly-query.sh"
---

你是具有只读访问权限的数据库分析师。执行 SELECT 查询回答数据问题。

分析数据时：
1. 识别包含相关数据的表
2. 编写带适当过滤器的高效 SELECT 查询
3. 清晰呈现结果和上下文

你无法修改数据。如果被要求 INSERT、UPDATE、DELETE 或修改模式，解释你只有只读访问权限。
```

配套验证脚本 `scripts/validate-readonly-query.sh`：

```bash
#!/bin/bash
# 阻止 SQL 写操作，允许 SELECT 查询

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

if [ -z "$COMMAND" ]; then
  exit 0
fi

# 阻止写操作（不区分大小写）
if echo "$COMMAND" | grep -iE '\b(INSERT|UPDATE|DELETE|DROP|CREATE|ALTER|TRUNCATE|REPLACE|MERGE)\b' > /dev/null; then
  echo "已阻止：不允许写操作。只能使用 SELECT 查询。" >&2
  exit 2
fi

exit 0
```

---

## 何时使用 Subagent vs 主对话

### 使用主对话

- 任务需要频繁来回或迭代完善
- 多个阶段共享大量上下文（规划 → 实现 → 测试）
- 进行快速、有针对性的更改
- 延迟很重要（Subagent 从头开始，可能需要时间收集上下文）
### 使用 Subagent

- 任务产生大量输出，不需要在主上下文中保留
- 想要强制特定的工具限制或权限
- 工作是自包含的，可以返回摘要
---

## 恢复 Subagent
每次 Subagent 调用创建新实例。要继续现有 Subagent 的工作而非重新开始：

```
> 使用 code-reviewer Subagent 审查认证模块
[代理完成]

> 继续那个代码审查，现在分析授权逻辑
[Claude 恢复带有完整上下文的 Subagent]
```

恢复的 Subagent 保留完整对话历史，从停止的地方继续。

---

## 禁用特定 Subagent

在设置的 `deny` 数组中添加 `Task(subagent-name)` 格式禁用：

```json
{
  "permissions": {
    "deny": ["Task(Explore)", "Task(my-custom-agent)"]
  }
}
```

或使用 CLI 参数：

```bash
claude --disallowedTools "Task(Explore)"
```

---

## 下一步
[Agent Skills创建可复用的知识模块
](../skills/)[Hooks 系统在事件触发时自动执行脚本
](../hooks/)[MCP 服务器连接外部工具和数据源
](../mcp-servers/)
