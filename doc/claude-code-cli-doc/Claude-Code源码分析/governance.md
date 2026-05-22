# Source-Analysis / Governance

> 来源: claudecn.com

# 权限治理

Claude Code 的权限系统不是一个简单的"允许/拒绝"开关，而是一套贯穿模式、规则、Hook 和沙箱四个层次的运行时边界系统。理解这套治理链路，比理解任何单个工具都重要——它决定了系统能否从"会调用工具的 CLI"进化成可进入真实工作流的执行平台。

可视化对应：[Runtime 页面](https://code.claudecn.com/runtime/)

## 为什么权限治理值得单独成页

如果只看 `src/tools/`，很容易得出一个错误结论：Claude Code 的能力主要来自"工具很多"。

真正更重要的事实是，工具只是动作面。是否允许执行、是否必须询问、是否还能落到沙箱边界，决定了这个系统有没有工程可控性。

## 四层治理结构

```
放行 → 询问 → 拒绝 → 允许 → 拒绝 → 工具调用请求 → 模式层 → PermissionMode → 规则层 → allow / deny / ask → 判定层 → toolPermission Hook → 执行层 → Sandbox 约束 → 用户确认 → 拒绝执行 → 实际执行
```

| 层 | 作用 | 代表证据 |
| --- | --- | --- |
| 模式层 | 定义当前会话的总体姿态，例如默认、plan、acceptEdits、bypassPermissions | `src/types/permissions.ts`、`src/utils/permissions/PermissionMode.ts` |
| 规则层 | 把 allow、deny、ask、additionalDirectories 等规则并入当前上下文 | `src/utils/permissions/permissions.ts`、`src/utils/permissions/permissionSetup.ts` |
| 判定层 | 在工具调用前做交互式、协调式或 worker 级别判定 | `src/hooks/toolPermission/` |
| 执行层 | 对 shell、文件系统、网络等动作施加最后约束 | `src/utils/sandbox/sandbox-adapter.ts`、`src/utils/permissions/filesystem.ts` |

这四层不是串联的"前端提示 → 后端执行"老式结构，而是会在一次请求中多次回流的运行时边界系统。

## 一个动作是如何被治理的

- 当前会话先带着 PermissionMode 与规则集合进入主循环。
- 工具调用前，src/hooks/toolPermission/ 会读取当前 toolPermissionContext，决定是放行、拒绝还是询问。
- 如果需要询问，系统还会触发 PermissionRequest、PermissionDenied 这类 Hook 事件，而不是只在 UI 里弹一个临时提示。
- 即使模式或规则允许，最终动作仍可能被 sandbox 或文件系统边界收紧。
这就是为什么 `acceptEdits` 或 `bypassPermissions` 并不等于系统完全失去结构边界。模式只是上层姿态，底层仍有真实执行约束。

## 配置面也属于治理系统的一部分

在 `settings` 侧，源码能看到治理入口并不只有 `permissions.*`，还包括 `sandbox.*`。这两者解决的是不同层次的问题：

- permissions.* 决定谁可以尝试做事
- sandbox.* 决定即使允许尝试，执行环境还能保留哪些硬边界
这两个面共同组成了运行时治理，而不是互相替代。

## 治理边界跨通道穿透

在桥接相关逻辑里，`tool_call` 负载携带 `permission_mode` 与 `allowed_domains`。这意味着治理边界不会在本地工具处停止，而是穿透到扩展或桥接面——权限是跨执行通道的公共协议，不是某个模块的内部细节。

## 对 agent 开发者的启示

| 模式 | 说明 |
| --- | --- |
| **先收紧再放开** | 默认最严格模式，让用户主动选择放宽——而不是从"全部放行"开始再打补丁 |
| **权限不止是 prompt** | 规则匹配、Hook 拦截、沙箱约束三层叠加，任何一层都可以独立拦截 |
| **治理跨通道** | 权限模型随 tool_call 负载穿透到 Bridge 和 MCP，不局限于本地工具 |
| **配置与运行时分离** | `permissions.*` 管"谁可以尝试"，`sandbox.*` 管"即使允许后还能做到什么程度" |

## 路径证据

| 路径 | 职责 |
| --- | --- |
| `src/types/permissions.ts` | 类型层定义 permission mode 与对外暴露边界 |
| `src/utils/permissions/PermissionMode.ts` | 模式到用户可见语义的映射 |
| `src/utils/permissions/permissions.ts` | 规则解析成运行时可判定结构 |
| `src/utils/permissions/permissionSetup.ts` | 当前会话上下文接入治理系统 |
| `src/hooks/toolPermission/` | 工具执行前的权限判定与交互 |
| `src/utils/sandbox/sandbox-adapter.ts` | 最终执行环境限制 |
| `src/utils/settings/permissionValidation.ts` | 配置入口的治理 |

## 进一步阅读

- 运行时流程 — 权限闸门在七阶段中的位置
- 记忆系统 — 治理与连续性的交叉
- 工具平面 — 被治理的工具全景
- 扩展与信号 — 治理如何延伸到扩展面
