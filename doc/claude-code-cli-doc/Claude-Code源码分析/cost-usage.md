# Source-Analysis / Cost-Usage

> 来源: claudecn.com

# 成本追踪

`cost-tracker.ts`（323 行）是一个常被忽略但贯穿整个运行时的横切关注点。它不只是"记录花了多少钱"，而是一个完整的 7 维会话级计量系统。

## 核心问题

AI agent 运行的每一次 API 调用、每一次工具执行、每一次代码变更都有成本。用户需要知道"这次会话花了多少钱"，系统需要知道"当前还能花多少"，运维需要知道"异常消耗出现在哪里"。

成本追踪不是附加功能，而是 agent 可持续运行的基础设施。

## 7 维计量体系

| 追踪维度 | 说明 |
| --- | --- |
| **Token 计量** | 输入 token、输出 token、缓存创建 token、缓存读取 token 分别统计 |
| **时间分解** | 总耗时、API 调用耗时、重试排除后 API 耗时、工具执行耗时独立计算 |
| **代码变更** | 新增行数、删除行数持续累计 |
| **多模型拆分** | 按模型名归类用量，支持 `getUsageForModel()` 按模型查询 |
| **成本估算** | `calculateUSDCost()` 基于模型定价表实时估算美元成本 |
| **Web Search** | 独立追踪 Web 搜索请求次数 |
| **状态持久化** | `setCostStateForRestore()` / `resetCostState()` 支持会话恢复和重置 |

## 架构设计

```
消费端 → cost-tracker.ts → 事件输入 → API 调用 → token 计量 → 工具执行 → 耗时计量 → 代码变更 → 行数累计 → Web 搜索 → 次数统计 → 全局状态原子 → bootstrap/state.js → 多模型分桶 → getUsageForModel() → 成本估算 → calculateUSDCost() → 状态栏显示 → 实时费用 → 会话持久化 → 恢复/重置 → 遥测上报 → logEvent()
```

### 全局状态原子

成本追踪器通过 `bootstrap/state.js` 中的全局状态原子实现。任何模块都可以通过 `getTotalCostUSD()` 等函数读取当前会话成本，而不需要传递计数器引用。这是一种典型的**横切关注点全局化**设计——成本信息需要在 UI 层（状态栏）、运行时层（预算检查）和遥测层（上报）同时可用。

### Token 计数的两种方式

| 方式 | 场景 | 精度 |
| --- | --- | --- |
| **规范方式** | 从 API 响应的 `usage` 字段获取 | 精确 |
| **粗略估算** | 请求发送前预估、图片/文档估算 | 约估 |

粗略估算规则：

- 普通文本：4 字节/token
- JSON 格式：2 字节/token（密集格式需要更保守的估算）
- 图片/文档：保守估算 2,000 token（真实公式：width × height / 750）
### 多模型分桶

当一次会话中涉及多个模型（主模型 + 子代理模型 + YOLO 分类器模型）时，成本追踪器按模型名分桶统计。`getUsageForModel()` 支持按模型查询单独用量。

## 与运行时的关系

成本追踪不是孤立的记录器，而是与运行时多个子系统交互：

| 交互对象 | 关系 |
| --- | --- |
| **自动压缩** | 压缩决策考虑当前 token 消耗率 |
| **Token 预算** | `MAX_TOOL_RESULTS_PER_MESSAGE_CHARS = 200K` 防止单次工具洪泛 |
| **遥测系统** | API 三事件模型（query/success/error）+ TTFT/TTLT 性能指标 |
| **会话恢复** | `setCostStateForRestore()` 在 resume 时恢复累计成本 |
| **状态栏** | 实时显示当前会话美元成本 |

## 可观测性关联

成本追踪是 Claude Code 5 层遥测体系的数据源之一。每次 API 调用结束后，`logEvent()` 会将 token 用量、耗时和成本信息上报到遥测管线，经过 PII 过滤后分流到 Datadog 和内部数据湖。

关键指标：

- TTFT（Time to First Token）：首 token 延迟
- TTLT（Time to Last Token）：末 token 延迟
- 缓存命中率：cacheReadTokens / totalInputTokens
## 对 agent 开发者的启示

| 模式 | 说明 |
| --- | --- |
| **横切计量** | 成本追踪应该像日志一样无处不在，而不是后期补加 |
| **多维分桶** | 不要只记总 token——按模型、按时间、按变更类型分别统计 |
| **保守估算** | 在精确数据不可用时，用保守估算兜底（JSON 2 字节/token） |
| **会话持久化** | 成本状态需要跨 resume 保持，否则用户看到的费用会不连续 |
| **全局可读** | 通过全局状态原子让任何模块都能读取成本，而不是手动传参 |

## 路径证据

| 路径 | 职责 |
| --- | --- |
| `src/cost-tracker.ts` | 成本追踪器核心（323 行） |
| `src/bootstrap/state.js` | 全局状态原子 |
| `src/utils/modelCost.ts` | 模型定价表 |
| `src/services/analytics/` | 遥测管线 |
| `src/components/` | 状态栏显示 |

## 进一步阅读

- 运行时流程 — 成本追踪在 queryLoop 中的位置
- 架构地图 — 横切关注点的系统位置
- 记忆系统 — 压缩与成本的关系
