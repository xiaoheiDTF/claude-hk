# Source-Analysis / Architecture

> 来源: claudecn.com

# 架构地图

一个最基础但也最容易被误读的问题：Claude Code 这个系统到底由哪些层组成？这里不照搬物理目录树，而是把发布包里暴露出来的路径重新组织成更适合理解的结构层。

**Treemap 一览代码重心**：[Architecture 可视化](https://code.claudecn.com/architecture/) 提供交互式 Treemap——面积正比于文件数，颜色区分功能角色。悬停查看详情，直观感受代码分布。

**这里有两种不同统计口径。** `47` 来自 `../src/` 顶层首段分桶；六层里的 `33 / 216 / 510 / 222 / 46 / 24` 则是为了帮助理解而做的语义聚类路径计数，不等于真实物理目录层级。

## 分析对象

- 版本：@anthropic-ai/claude-code 2.1.88
- 源码文件：1902 个用户源码文件（从 cli.js.map 的 ../src/ 路径统计）
- 顶层模块：47 个顶层首段分桶（混合目录与单文件入口）
## 一手证据

| 证据 | 用途 | 代表内容 |
| --- | --- | --- |
| `package.json` | 确认发布边界 | 版本 2.1.88、bin 入口 `cli.js`、Node ≥ 18 |
| `cli.js.map` | 恢复源码路径 | `../src/` 下 47 个顶层模块的分布和文件数 |
| `sdk-tools.d.ts` | 验证工具层 | 工具 schema、输入输出类型定义 |
| `vendor/` | 识别运行时能力 | `ripgrep`（代码搜索）、`audio-capture`（语音输入） |

## 六层结构

### 第 1 层：入口与运行时核心（33 files）
这是系统真正开始运转的地方。CLI 入口接住请求后，经过模式判定和环境初始化，进入 QueryEngine 驱动的主循环。

**代表路径**：

- src/entrypoints/ — 入口分发，区分交互式 REPL、单次执行 -p、管道模式、MCP 服务器、SDK 模式
- src/cli/ — CLI 参数解析、版本检查、快速路径分发
- src/query* — QueryEngine 核心，submitMessage() → query() → queryLoop(){ while(true) } 主循环
- src/context.ts — 运行态上下文组装，工作目录、CLAUDE.md、token 预算
**核心机制**：QueryEngine 是整个系统的心跳。它不是简单的"发一次请求等一次结果"，而是一个 `while(true)` 循环——每次 API 调用后如果模型返回工具调用，循环继续；只有当模型返回纯文本（无工具调用）时才退出。

### 第 2 层：工具与治理边界（216 files）

工具不是简单的函数集合，而是一套带权限、路径约束、执行模式和外部连接能力的治理面。这一层是 Claude Code 区别于"会写 shell 的聊天框"的关键。

**代表路径**：

- src/tools/ — 43 个工具首段实体，包含 shared、testing、utils 等基础设施桶
- src/Tool.ts — 工具基类，定义注册、schema 声明、执行接口和输出格式
- src/utils/permissions/ — 权限加载与规则匹配，7 种模式 × 6 个规则源 × 3 种结果
- src/hooks/toolPermission/ — 工具执行前的权限 hook 拦截
- src/utils/sandbox/ — BashTool 专属沙箱，macOS sandbox-exec / Linux bubblewrap + seccomp
**治理闭环**：工具调用 → 权限规则匹配 → Hook 拦截（PreToolUse） → 沙箱约束（仅 Bash） → 执行 → Hook 回调（PostToolUse）。任何一层都可以拒绝执行。

### 第 3 层：终端 UI 与交互层（510 files）

用户看到的是命令行，但内部并不是"输出一段文本"那么简单。这是文件数最多的一层，由 React + Ink 自定义终端渲染引擎驱动。

**代表路径**：

- src/components/ — 终端 UI 组件，消息渲染、状态栏、对话框、进度指示器
- src/ink/ — 自定义 Ink 引擎，ConcurrentRoot 并发渲染、Yoga Flex 布局、Cell 压缩
- src/keybindings/ — 17 个键绑定上下文、100+ 默认快捷键
- src/vim/ — 完整 Vim 模式：Normal / Insert / Visual，含 motions + operators + text objects
- src/state/ — 全局状态管理，驱动 UI 响应式更新
**为什么这层最重**：Claude Code 不只是"后端 agent 加个打印层"。状态栏实时显示模型/模式/CWD/上下文/费用/Vim 状态；@ 文件自动补全、Deep Link 协议、VS Code 桥接都属于这层。

### 第 4 层：扩展织物（222 files）

这层负责让系统接入外部能力。MCP、skills、bridge、plugins、hooks 构成一张"扩展织物"，决定了 Claude Code 为什么可以不断长出新能力。

**代表路径**：

- src/services/mcp/ — MCP 集成，6 种传输（stdio / SSE / HTTP / WebSocket / SDK / claude.ai proxy）、7 个配置范围、OAuth 2.0 + PKCE
- src/skills/ — 技能系统，SKILL.md 格式（YAML 前置 + Markdown）、3 级渐进式发现（Discovery → Invocation → Fork）
- src/utils/plugins/ — 插件系统，7 种组件（commands / agents / skills / hooks / output-styles / MCP servers / LSP）、7 种安装源
- src/bridge/ — 跨界面桥接，IDE / Desktop / Mobile / Chrome 连接
- src/hooks/ — 钩子系统，27 个生命周期事件、4 种类型（command / prompt / agent / HTTP）
**扩展深度**：MCP 工具命名遵循 `mcp__<server>__<tool>` 模式，动态注册到工具系统中，与内建工具共用权限检查和执行路径。

### 第 5 层：记忆与恢复（46 files）

长会话能否稳定持续，不取决于模型一次回答得多好，而取决于系统如何压缩、回写、恢复和继承上下文。

**代表路径**：

- src/services/compact/ — 五层压缩策略：Micro → TimeBased → APISide → SessionMemory → Full
- src/services/SessionMemory/ — 会话记忆，压缩后保留 files / skills / plan / hooks 等结构化摘要
- src/memdir/ — 记忆目录，持久化存储、跨会话继承
- src/tasks/ — 任务状态管理
- src/migrations/ — 数据迁移，处理跨版本格式变化
**压缩触发**：当 token 使用率达到预算的 ~93% 时触发自动压缩（effective-13K 阈值），9 节摘要模板保证压缩后的上下文仍然可用。

### 第 6 层：实验外沿（24 files）

还有一些路径并不一定属于当前主干能力，但它们透露了系统正在探索的方向。

**代表路径**：

- src/buddy/ — 终端桌宠系统：确定性抽卡、ASCII 动画引擎、AI 观察者
- src/services/autoDream/ — 后台记忆整理：会话间自动梦境处理
- src/services/teamMemorySync/ — 团队记忆同步：跨成员知识共享
- src/services/voice/ + src/voice/ — 语音表面：语音输入与命令
- src/remote/ — 远程控制能力
- src/coordinator/ — 协调器模式：Lead Agent 分解任务、并行 Worker
## 跨层依赖关系

```
入口与运行时核心 → 33 files → 工具与治理边界 → 216 files → 终端 UI 与交互 → 510 files → 扩展织物 → 222 files → 记忆与恢复 → 46 files → 实验外沿 → 24 files
```

说明：实线表示主干依赖，虚线表示实验性依赖。入口层驱动工具层和记忆层；工具层接入扩展织物；UI 层观察运行时核心；实验外沿通过扩展织物和记忆层与主干连接。

## 记忆与恢复层补充说明

第 5 层最容易被写成“神奇记忆系统”。更准确的理解是：Claude Code 把**当前会话连续性、跨会话记忆、压缩恢复材料、后台整理**拆成不同工件，它们解决的是不同时间尺度的问题。

### 四类连续性工件

| 类别 | 生命周期 | 主要路径 | 作用 |
| --- | --- | --- | --- |
| **Session Memory** | 当前长会话 | `src/services/SessionMemory/` | 在会话中维护结构化工作摘要，优先服务 compact 与 resume |
| **Memory Directory** | 跨会话 | `src/memdir/` | 保存项目、用户、反馈、参考等持久记忆 |
| **Compact Artifacts** | 压缩边界前后 | `src/services/compact/`、`src/attachments.ts` | 在上下文被压缩后重新注入关键材料，维持连续性 |
| **Background Consolidation** | 慢周期后台 | `src/services/autoDream/`、`src/services/teamMemorySync/` | 做记忆整理、合并、同步与慢周期治理 |

这里刻意不使用未经 2.1.88 证据稳定支持的命名。真正重要的是：**不要把长会话摘要、跨会话记忆和后台整理混成同一个“memory”概念。**

### 五层压缩策略

当 token 使用率达到预算的 ~93% 时（effective-13K 阈值），触发自动压缩。压缩不是简单的截断，而是分层降级：

- Micro Compaction — 移除冗余的工具调用结果
- Time-Based — 按时间衰减压缩早期对话
- API-Side — 利用 API 的 prompt caching 降低实际传输量
- SessionMemory — 用 9 节摘要模板（files、skills、plan、hooks 等）压缩为结构化摘要
- Full Compaction — 极端情况下的全量重写
这个设计的启示：长会话 agent 的稳定性不取决于模型多聪明，而取决于上下文治理和连续性工件分工是否足够精细。

## 常见误读

- 误读一：commands 就是系统核心 → 实际上它更像暴露层，真正的能力分布在工具、服务和运行时
- 误读二：看到 510 个 UI 文件就认为系统"前端重" → UI 层的复杂度来自自研渲染引擎（Ink ConcurrentRoot + Yoga 布局 + Cell 压缩 + 差量更新），这是终端 agent 的核心竞争力
- 误读三：把 vendor/ 当成无关资源 → 它说明发布包已经把 ripgrep（代码搜索核心）和 audio-capture（语音入口）固化到分发层
- 误读四：24 个实验文件说明系统不稳定 → 主干 5 层共 1027 files 都是 high confidence，实验层是独立探索
## 建议阅读顺序

- 运行时流程 — 从空间切换到时间视角
- 工具平面 — 看清能力如何组织
- 权限治理 — 理解执行边界
- 扩展与信号 — 看系统往哪里长
