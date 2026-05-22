# Source-Analysis / Memory

> 来源: claudecn.com

# 记忆系统

Claude Code 的"记忆"不是一个孤立功能，而是运行时主干的核心组成部分。这里讨论的不是泛泛的"上下文很长"，而是系统如何装载记忆、压缩历史、保留边界，并在下一轮继续工作。

可视化对应：[Runtime 页面](https://code.claudecn.com/runtime/)

## 为什么记忆不是附属功能

如果 Claude Code 只是一次问答工具，那么记忆最多只是一个优化项。

但只要系统要处理多轮任务、子代理、计划审批、长会话恢复，它就必须回答三个问题：

- 哪些信息要在当前轮装进上下文
- 哪些信息要在压缩后保留下来
- 哪些信息要沉淀成下一轮可再次加载的结构
这已经不是"聊天历史长度"的问题，而是一个持续运行系统的状态管理问题。

## 四个子系统

```
压缩 → 运行 → 装载 → 摘要 → 沉淀 → memdir 长期记忆 → 相关性筛选 → CLAUDE.md 项目约定 → 当前轮上下文 → query loop 执行 → SessionMemory 会话状态 → compact 五层策略 → extractMemories
```

| 子系统 | 作用 | 代表证据 |
| --- | --- | --- |
| 记忆目录 | 存放和定位长期或半长期记忆 | `src/memdir/paths.ts`、`src/memdir/memdir.ts` |
| 相关性筛选 | 决定哪些记忆应在当前轮进入上下文 | `src/memdir/memoryScan.ts`、`src/memdir/findRelevantMemories.ts` |
| 会话记忆 | 维护当前对话的可持续状态 | `src/services/SessionMemory/sessionMemory.ts` |
| 压缩与回写 | 在 token 压力下保留可继续执行的边界与摘要 | `src/services/compact/compact.ts`、`src/services/compact/sessionMemoryCompact.ts` |

这四层合在一起，才是 Claude Code 的"长会话能力"。

## 运行时闭环

- 请求开始时，系统先从项目约定、CLAUDE.md 与 memdir 路径装载可用记忆。
- 这些内容经过筛选后进入上下文，而不是全部平铺到 prompt。
- 当对话变长，compact 系列模块会尝试把历史重新整理成更短但仍可恢复的表示。
- SessionMemory 负责让这些结果在当前会话中继续可用。
- 更长期的内容还可能通过 extractMemories 或 teamMemorySync 进入更稳定的沉淀层。
Claude Code 不是简单地"把旧消息往前拼"。它会在运行中不断重写自己未来可见的上下文。

## 为什么 compact 很关键

很多分析会把 compact 理解成"快没 token 了，所以总结一下"。实际上，compact 真正承担的是状态延续职责：

- 它决定哪些内容还能在下一轮被保留
- 它决定恢复时依赖的是原始消息还是摘要边界
- 它决定长任务会不会在中段失去上下文骨架
也正因为如此，`compact.ts`、`sessionMemoryCompact.ts`、`postCompactCleanup.ts` 这一组文件更像 runtime continuity 组件，而不是普通工具函数。

## 记忆层和治理层的关系

这两层不应该分开理解。

治理层决定系统能做什么，记忆层决定系统还能记住什么。前者保证边界，后者保证连续性。没有治理，长会话会失控；没有记忆，长任务会断裂。

从工程视角看，这两层一起构成了 Claude Code 与"单轮工具调用器"之间最重要的差别。

## 对 agent 开发者的启示

| 模式 | 说明 |
| --- | --- |
| **记忆不是全量加载** | 相关性筛选（`findRelevantMemories`）决定哪些记忆进入当前轮，避免上下文膨胀 |
| **压缩是状态延续** | compact 不是简单的摘要，而是决定下一轮能恢复到什么程度的运行时机制 |
| **分层沉淀** | 当前会话 → SessionMemory → memdir → 团队同步，四层各有不同的生命周期 |
| **记忆与治理协同** | 没有治理边界的长记忆会失控，没有记忆的治理系统会断裂 |

## 路径证据

| 路径 | 职责 |
| --- | --- |
| `src/memdir/paths.ts` | 记忆文件路径定义 |
| `src/memdir/memdir.ts` | 记忆目录核心逻辑 |
| `src/memdir/memoryScan.ts` | 记忆扫描与发现 |
| `src/memdir/findRelevantMemories.ts` | 相关性筛选 |
| `src/services/SessionMemory/sessionMemory.ts` | 会话记忆对象 |
| `src/services/compact/compact.ts` | 压缩主逻辑 |
| `src/services/compact/sessionMemoryCompact.ts` | 会话记忆专用压缩 |
| `src/services/extractMemories/` | 记忆抽取 |
| `src/services/teamMemorySync/` | 团队记忆同步 |

## 进一步阅读

- 运行时流程 — 记忆装载和压缩在七阶段中的位置
- 权限治理 — 治理与记忆的协同
- Hooks 韧性 — 记忆恢复相关的韧性机制
- 扩展与信号 — 记忆系统的演进方向
