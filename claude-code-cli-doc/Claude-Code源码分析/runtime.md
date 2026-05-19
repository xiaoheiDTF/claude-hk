# Source-Analysis / Runtime

> 来源: claudecn.com

# 运行时流程

从"有哪些目录"切换到"一个请求是怎么穿过去的"。如果架构分析是空间地图，运行时流程就是时间地图。

可视化对应：[Runtime 页面](https://code.claudecn.com/runtime/)

## 核心问题

一次请求进入 Claude Code 之后，系统大致会经过哪些阶段，在哪些地方收紧边界，在哪些地方把结果重新压缩回长期上下文。

## 七个主要阶段

```
tool_use 响应 → 结果回流 → 入口分发 → 上下文组装 → 采样与缓存 → 权限闸门 → 工具执行 → 压缩与记忆回写 → 渲染与通知
```

| 阶段 | 说明 | 代表路径 |
| --- | --- | --- |
| 入口分发 | 决定当前会话从哪个入口进入 | `src/entrypoints/`、`src/cli/` |
| 上下文组装 | 收集工作目录、项目约定、记忆和 token 预算 | `src/context.ts`、`src/utils/claudemd.ts`、`src/memdir/` |
| 采样与缓存 | 调模型、处理重试、缓存与速率限制 | `src/services/api/`、`src/services/rateLimit/` |
| 权限闸门 | 在真正执行动作前收紧 shell、路径和工具权限 | `src/utils/permissions/`、`src/hooks/toolPermission/` |
| 工具执行 | 让文件、终端、MCP、代理、任务等实体参与主循环 | `src/tools/`、`src/Tool.ts` |
| 压缩与记忆回写 | 维护长会话可持续性，避免上下文失控 | `src/services/compact/`、`src/services/SessionMemory/` |
| 渲染与通知 | 把运行时状态转成终端 UI、状态条和通知 | `src/components/`、`src/ink/`、`src/hooks/notifs/` |

## queryLoop：核心心跳

整个运行时的心跳在 `src/query.ts` 第 241-1728 行的 `queryLoop()` 函数中。它不是"调用模型一次就结束"，而是一个持续运行的 `while(true)` 循环。每次迭代：

- 解构跨迭代状态：messages, toolUseContext, autoCompactTracking …
- 组装系统 prompt + 工具列表（经过 feature flag / deny 规则 / MCP 合并三层过滤）
- 调用 API（streaming response）
- 解析响应——模型返回纯文本则退出循环；返回 tool_use 则进入工具执行路径
- 工具执行：权限检查 → Hook → 沙箱 → 并行/串行执行 → 结果回流
- 检查压缩阈值（~93% token 预算），必要时触发 compact
- 将工具结果拼接回 messages，continue 进入下一次迭代
跨迭代状态（`State` 对象，第 268-279 行）包含：`messages`（完整对话历史）、`toolUseContext`（工具执行上下文）、`turnCount`（迭代计数）、`maxOutputTokensRecoveryCount`（max_tokens 重试）、`hasAttemptedReactiveCompact`（反应式压缩标记）等。循环内有 7 个 `continue` 出口和 1 个 `return` 出口。

一次用户请求可能触发数十次循环迭代——这是 Claude Code 区别于普通聊天工具的根本原因。

## 两个关键闭环

### 执行闭环

请求进入后不会直接结束，而是经过"判断 → 工具执行 → 结果回流 → 再判断"的循环。`StreamingToolExecutor`（`src/services/tools/StreamingToolExecutor.ts`）在 API 响应流式到达时就开始并行启动 `isConcurrencySafe` 工具，写操作工具则串行执行，避免文件冲突。

### 记忆闭环

长任务能否持续推进，取决于系统是否能把必要信息压缩回可复用状态。`compact`（五层策略：Micro → TimeBased → APISide → SessionMemory → Full）、`SessionMemory`（9 节结构化摘要模板）和 `memdir`（跨会话持久化）共同构成了这条闭环。

## 两个值得单独展开的层

### 权限治理层

运行时里最容易被低估的部分，不是工具系统，而是执行前的治理系统。源码里至少能看到四类证据在共同工作：

- src/types/permissions.ts 与 src/utils/permissions/PermissionMode.ts：定义模式和对外行为边界
- src/utils/permissions/permissions.ts、src/utils/permissions/permissionSetup.ts：把规则和当前上下文合并成真实可执行边界
- src/hooks/toolPermission/：在真正执行工具前处理交互式判定、协调者判定和 worker 判定
- src/utils/sandbox/sandbox-adapter.ts：把最后一层执行约束落到沙箱侧，而不是只停留在 UI 提示
这说明"权限闸门"不是一个单点函数，而是一条从配置、模式、规则、Hook 到 sandbox 的多段链路。更完整的拆解见 [权限治理](https://claudecn.com/docs/source-analysis/governance/)。

### 记忆层

`compact` 也不只是 token 优化。source map 里可以直接看到这条链路跨越了多组模块：

- src/memdir/paths.ts、src/memdir/memdir.ts、src/memdir/findRelevantMemories.ts
- src/services/SessionMemory/sessionMemory.ts、src/services/SessionMemory/sessionMemoryUtils.ts
- src/services/compact/compact.ts、src/services/compact/sessionMemoryCompact.ts、src/services/compact/autoCompact.ts
也就是说，Claude Code 并不是"每轮重新开始"，而是在持续装载记忆、压缩历史、保留边界，并为下一轮恢复上下文。更完整的拆解见 [记忆系统](https://claudecn.com/docs/source-analysis/memory/)。

### 成本追踪层

`src/cost-tracker.ts`（323 行）是贯穿整个运行时的横切关注点——7 维会话级计量，覆盖 token、时间、代码变更、多模型拆分和成本估算。通过 `bootstrap/state.js` 全局状态原子实现，任何模块都可以读取当前会话成本。

完整的成本追踪架构分析见 [成本追踪](https://claudecn.com/docs/source-analysis/cost-usage/) 专题。

### Hooks 与运行时韧性

`src/hooks/` 目录包含 80+ 个 React hooks 文件，其中相当一部分直接参与运行时韧性——从权限判定、会话恢复到资源监控和退出安全。运行时的错误恢复不依赖单一机制，而是分布在 API 侧重试、反应式压缩、权限降级和会话持久化四个独立节点上。

完整的 Hooks 韧性架构分析见 [Hooks 韧性](https://claudecn.com/docs/source-analysis/hooks-resilience/) 专题。

## 常见误区

- 不要把 commands 当成运行时本体。命令只是触发面。
- 不要把压缩理解成"省 token 的小优化"。它实际上决定长会话能不能继续。
- 不要忽略权限闸门。Claude Code 能走向生产可用，很大程度靠的是这一层。
## 进一步阅读

- 架构地图
- 工具平面
- 命令表面
- 权限治理
- 记忆系统
