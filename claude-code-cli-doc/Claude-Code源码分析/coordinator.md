# Source-Analysis / Coordinator

> 来源: claudecn.com

# Coordinator 编排

从单 agent 到多 Worker 编排——`coordinatorMode.ts`（369 行）揭示了 Claude Code 如何通过 Lead Agent 分解任务、并行 Worker 执行、独立验证构建多代理工作流。

## 核心问题

单个 Agent Loop 的上下文窗口是有限资源。当任务规模超过单次对话所能承载的信息量时——例如"调查 bug 根因 → 修复 → 跑测试 → 写 PR"——单 agent 要么被迫塞满中间结果，要么不断做压缩丢失细节。更本质的问题是：**单 agent 无法并行**，而[ 软件](#)工程任务天然适合分治。

## 子系统全景

| 指标 | 数值 |
| --- | --- |
| **核心文件** | `coordinatorMode.ts`（369 行） |
| **目录位置** | `src/coordinator/` |
| **成熟度** | Emerging（正在浮现） |
| **激活方式** | 环境变量 `CLAUDE_CODE_COORDINATOR_MODE` + Feature Flag `COORDINATOR_MODE` |

## 四阶段工作流

Coordinator Mode 将复杂任务分解为四个有序阶段：

```
Research → 调研 → Synthesis → 综合 → Implementation → 实现 → Verification → 验证
```

| 阶段 | 说明 | Worker 特征 |
| --- | --- | --- |
| **Research** | 调研问题、收集上下文 | 只读，可自由并行 |
| **Synthesis** | 综合调研结果、形成方案 | 聚合型，通常串行 |
| **Implementation** | 按方案实现代码变更 | 写操作，按文件范围串行 |
| **Verification** | 独立验证实现结果 | **不携带实现上下文** |

### 独立验证

验证阶段的设计是 Coordinator 最值得关注的决策：验证 Worker 不携带实现 Worker 的上下文。这不是偷懒——而是有意的**强制独立视角**。如果验证者看到了实现过程中的推理链，它更容易被说服"这应该是对的"。独立验证要求验证者从头审视结果。

## 核心组件

| 组件 | 职责 |
| --- | --- |
| `isCoordinatorMode()` | 环境变量 + Feature Flag 双重检查 |
| `matchSessionMode()` | 恢复会话时自动匹配 coordinator/normal 模式，防止模式漂移 |
| `getCoordinatorUserContext()` | 为 Worker 注入可用工具列表、MCP 服务器列表、Scratchpad 目录 |
| `getCoordinatorSystemPrompt()` | 369 行的完整 System Prompt，定义角色、工具、工作流和 Worker 管理协议 |

## Worker 管理

```
AgentTool → AgentTool → SendMessageTool → TaskStopTool → 结果 → 结果 → AgentTool → AgentTool → 代码变更 → PASS/FAIL → Lead Agent → Coordinator → Worker 1 → Research → Worker 2 → Research → Worker 3 → Implementation → Worker 4 → Verification
```

Worker 通过三个工具管理：

| 工具 | 职责 |
| --- | --- |
| `AgentTool` | 生成新 Worker |
| `SendMessageTool` | 向运行中的 Worker 发送消息/继续指令 |
| `TaskStopTool` | 停止 Worker |

### 并行策略

- 只读任务：可自由并行（多个 Research Worker 同时调研不同方面）
- 写操作：按文件范围串行（避免两个 Worker 同时修改同一文件）
- 验证：在所有实现完成后启动，独立上下文
### Scratchpad 共享

通过 `tengu_scratch` Feature Flag 控制的跨 Worker 持久化目录。Worker 可以将中间结果写入 Scratchpad，其他 Worker 可以读取——这是 Worker 之间唯一的显式通信通道（除了通过 Lead Agent 转发）。

## 与多 Agent 体系的关系

Claude Code 提供三种递进的多 agent 模式：

| 模式 | 复杂度 | 上下文继承 | 用途 |
| --- | --- | --- | --- |
| **子 Agent** | 低 | 全新对话 | 单任务委派 |
| **Fork** | 中 | 后台执行 | 并行探索 |
| **Coordinator** | 高 | Worker 隔离 | 多阶段工作流 |

Coordinator 是最重量级的模式，它不是简单的"多开几个 agent"，而是一套完整的任务分解、分配、执行和验证框架。

### 与 Teams 的区别

Coordinator Mode 是进程内编排——所有 Worker 在同一个 Claude Code 进程中运行。Teams（第 20b 章）则是跨进程协作——多个独立的 Claude Code 实例通过 UDS（Unix Domain Socket）通信。

## 权限处理

Coordinator 模式有自己的权限处理器 `coordinatorHandler.ts`，与标准的 `interactiveHandler`（交互式确认）和 `swarmWorkerHandler`（Worker 模式）并行存在。按运行时身份自动选择。

## 对 agent 开发者的启示

| 模式 | 说明 |
| --- | --- |
| **独立验证** | 实现者和验证者必须隔离上下文——不让实现者自己给自己打分 |
| **分治并行** | 只读任务并行、写操作串行——这是文件系统一致性的最小约束 |
| **四阶段分解** | Research → Synthesis → Implementation → Verification 是通用的复杂任务处理范式 |
| **模式漂移防护** | `matchSessionMode()` 在会话恢复时自动匹配模式，防止 coordinator 会话以 normal 模式恢复 |
| **Scratchpad 通信** | Worker 间的数据交换不通过共享内存，而是通过文件系统——简单但可靠 |

## 路径证据

| 路径 | 职责 |
| --- | --- |
| `src/coordinator/coordinatorMode.ts` | 编排模式核心（369 行） |
| `src/tools/AgentTool/` | Worker 生成与管理（15 个文件） |
| `src/tools/SendMessageTool/` | Worker 继续通信 |
| `src/tools/TaskStopTool/` | Worker 停止 |
| `src/hooks/toolPermission/handlers/coordinatorHandler.ts` | 编排模式权限处理 |

## 进一步阅读

- 工具平面 — AgentTool 的详细工作原理
- 权限治理 — 多模式权限处理器
- 扩展与信号 — Coordinator 的成熟度判定
- 记忆系统 — Worker 间的记忆隔离
