# Source-Analysis / Hooks-Resilience

> 来源: claudecn.com

# Hooks 韧性

80+ 个 React hooks 中有相当一部分直接参与运行时韧性——从权限判定、会话恢复到资源监控和退出安全，它们构成了分布在多个节点上的错误恢复网络。

## 核心问题

AI agent 的运行时韧性不能靠单一的 try/catch 兜底。当 API 返回 max_tokens、当上下文超限、当终端失焦、当 IDE 连接断开时——系统需要在多个独立节点上同时具备恢复能力，而不是在某一个地方集中处理所有异常。

## 韧性类别

| 类别 | 代表 Hook | 韧性职责 |
| --- | --- | --- |
| **权限判定** | `toolPermission/handlers/` | 三套并行处理器：`interactiveHandler`（交互式确认）、`coordinatorHandler`（编排模式）、`swarmWorkerHandler`（Worker 模式），按运行时身份选择 |
| **会话恢复** | `useSessionBackgrounding.ts` | 检测终端失焦/后台，触发会话保存与恢复 |
| **连接健康** | `useIdeConnectionStatus.ts`、`useDirectConnect.ts` | IDE 连接断开时提供降级路径 |
| **资源监控** | `useMemoryUsage.ts`、`useTerminalSize.ts` | 内存压力和终端尺寸变化的实时响应 |
| **任务看护** | `useScheduledTasks.ts`、`useTaskListWatcher.ts` | 后台任务状态轮询与异常通知 |
| **退出安全** | `useExitOnCtrlCD.ts` | 防止意外退出导致的状态丢失 |

## 分布式恢复架构

```
任意节点失败 → 其他节点仍可运转 → 独立恢复 → 资源侧监控 → useMemoryUsage → 内存压力 → useTerminalSize → 终端尺寸 → useExitOnCtrlCD → 退出安全 → 会话侧持久化 → useSessionBackgrounding → 终端后台化 → useIdeConnectionStatus → IDE 连接健康 → 权限侧降级 → interactiveHandler → 交互式确认 → coordinatorHandler → 编排模式 → swarmWorkerHandler → Worker 模式 → API 侧恢复 → max_tokens 重试 → maxOutputTokensRecoveryCount → 最多 3 次 → 反应式压缩 → hasAttemptedReactiveCompact
```

**核心设计**：四个恢复节点相互独立。API 侧重试失败不影响会话持久化；权限降级不影响资源监控；任意一个节点失败，其他节点仍能维持系统基本运转。

## 关键恢复机制

### API 侧：max_tokens 恢复

当 API 返回 `max_tokens`（模型输出被截断）时，`queryLoop` 通过 `maxOutputTokensRecoveryCount` 计数器控制重试。最多 3 次重试后放弃——这是一个**熔断器**模式，防止无限重试消耗 token。

### API 侧：反应式压缩

当上下文接近预算上限时，`hasAttemptedReactiveCompact` 标记确保反应式压缩只尝试一次。如果压缩后仍然超限，系统会进入更激进的恢复路径（丢弃最旧的 API 轮次组）。

### 权限侧：三套处理器

权限判定不是单一函数，而是根据运行时身份选择不同处理器：

- 交互式：弹出确认对话框，等待用户决定
- 编排模式：Lead Agent 代替用户做权限决策
- Worker 模式：Worker 的权限由 Lead Agent 预设
### 会话侧：后台化保护

`useSessionBackgrounding` 检测终端失焦（用户切换到其他窗口）或进入后台。触发后自动保存会话状态，确保即使进程被 OS 杀掉，下次启动时仍能恢复。

### 退出侧：级联超时

`useExitOnCtrlCD` 不是简单的 `process.exit()`。退出流程是级联的：

- 终端 UI 清理
- 运行中的工具中断
- Hook 回调（SessionEnd，1.5 秒超时）
- 遥测数据排空
- 5 秒失败保险——如果上述步骤卡住，强制退出
## Hooks 系统概览

Claude Code 的 Hooks 不仅是 UI 状态管理——26 种事件类型覆盖了从会话启动到工具执行到会话结束的完整生命周期：

| 事件组 | 代表事件 | 用途 |
| --- | --- | --- |
| **工具执行** | `PreToolUse`、`PostToolUse`、`PostToolUseFailure` | 工具执行前后的拦截和处理 |
| **权限** | `PermissionRequest`、`PermissionDenied` | 权限决策的 Hook 点 |
| **会话** | `SessionStart`、`SessionEnd` | 会话生命周期 |
| **用户交互** | `UserPromptSubmit`、`Stop` | 用户输入和停止信号 |
| **文件变更** | `FileChanged` | 文件系统变化通知 |

### Hook 执行模型

- 异步生成器：Hook 回调通过异步生成器执行
- 超时保护：默认 10 分钟超时，SessionEnd 为 1.5 秒
- 信任门控：交互模式下所有 Hook 都需要信任对话框确认
- 退出码语义：0 = 允许、2 = 阻塞错误（入队为任务通知）
## 对 agent 开发者的启示

| 模式 | 说明 |
| --- | --- |
| **[ 分布式](#)恢复** | 不要把所有异常处理集中在一个 try/catch 中——在 API 层、权限层、会话层、资源层分别设置恢复点 |
| **熔断器** | 重试必须有上限（max_tokens 最多 3 次），防止失控循环消耗资源 |
| **级联退出** | 退出不是 `process.exit(0)`，而是有序清理——但也要有"最后手段"超时 |
| **后台化保护** | agent 进程随时可能被 OS 杀掉，会话状态需要持续保存而不是退出时才写入 |
| **身份感知** | 同一个权限决策在不同运行时身份（交互/编排/Worker）下走完全不同的路径 |

## 路径证据

| 路径 | 职责 |
| --- | --- |
| `src/hooks/toolPermission/handlers/` | 三套权限处理器 |
| `src/hooks/useSessionBackgrounding.ts` | 会话后台化 |
| `src/hooks/useIdeConnectionStatus.ts` | IDE 连接健康 |
| `src/hooks/useMemoryUsage.ts` | 内存监控 |
| `src/hooks/useScheduledTasks.ts` | 任务调度 |
| `src/hooks/useExitOnCtrlCD.ts` | 退出安全 |
| `src/query.ts` 第 268-279 行 | 跨迭代恢复状态 |

## 进一步阅读

- 运行时流程 — Hooks 在 queryLoop 中的位置
- 权限治理 — 权限处理器的完整链路
- 记忆系统 — 会话恢复与记忆保持
