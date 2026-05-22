# Source-Analysis / Signals

> 来源: claudecn.com

# 扩展与信号

Claude Code 下一步可能往哪走？10 个能力信号从路径结构中被识别出来，它们不是营销功能列表，而是从真实代码中直接读到的系统外沿。按成熟度分成四级，避免把雏形误读为即将上线的功能。

可视化对应：[Signals 页面](https://code.claudecn.com/features/)

## 成熟度判定标准

| 等级 | 定义 | 判定依据 |
| --- | --- | --- |
| **Integrated** | 已集成到主干 | 有独立服务/工具目录、有命令入口、与其他子系统有明确依赖 |
| **Emerging** | 正在浮现 | 有独立目录和服务文件、但命令入口或 UI 尚未完整 |
| **Partial** | 部分完成 | 有目录和组件、但从入口和依赖看仍像局部挂载 |
| **Surface** | 表面信号 | 仅有命令入口或单文件、未形成独立子系统 |

## 信号全景

已成熟为独立专题的信号有独立页面，其余保留在这里作为观察索引。

### Integrated（6 个信号）

| 信号 | 置信度 | 说明 | 深入阅读 |
| --- | --- | --- | --- |
| **Bridge & Mailbox** | 高 | 跨界面连接能力，`bridge` 目录 + `mailbox` 上下文 + 多端命令 | [MCP 与 Bridge](https://claudecn.com/docs/source-analysis/mcp-bridge/) |
| **AutoDream & Memory Hygiene** | 高 | 后台记忆整理，与 memdir 和 session memory 形成闭环 | [记忆系统](https://claudecn.com/docs/source-analysis/memory/) |
| **Cron & Remote Triggers** | 高 | `ScheduleCronTool` 和 `RemoteTriggerTool` 提供定时调度和远程触发 | [工具平面](https://claudecn.com/docs/source-analysis/tools/) |
| **Verification Agent Lane** | 高 | `AgentTool` 的 `built-in/` 包含验证、规划、探索三种代理 | [Coordinator 编排](https://claudecn.com/docs/source-analysis/coordinator/) |
| **[ Computer](#) Use** | 高 | 15 个专用文件的桌面自动化子系统 | [Computer Use](https://claudecn.com/docs/source-analysis/computer-use/) |
| **Plugins** | 高 | ~1,700 行 Schema 的可安装扩展系统 | [插件系统](https://claudecn.com/docs/source-analysis/plugins/) |

### Emerging（3 个信号）

| 信号 | 置信度 | 说明 | 深入阅读 |
| --- | --- | --- | --- |
| **Team Memory Sync** | 高 | `teamMemorySync` 服务 + `teamMemPaths` 工具，跨成员记忆共享 | [记忆系统](https://claudecn.com/docs/source-analysis/memory/) |
| **Voice Surface** | 中 | 原生音频 + STT 流 + OAuth 闸门，Push-to-talk 语音输入 | [Voice 语音](https://claudecn.com/docs/source-analysis/voice/) |
| **Coordinator Mode** | 高 | 369 行的多 Worker 编排系统，四阶段工作流 | [Coordinator 编排](https://claudecn.com/docs/source-analysis/coordinator/) |

### Partial（1 个信号）

#### Buddy Companion

**置信度**：中

`buddy` 目录已经有 prompt、sprite、component 等完整组件（确定性抽卡、ASCII 动画引擎、AI 观察者），但从命令入口看仍像局部挂载的能力，而不是主干功能。

**路径证据**：`src/buddy/`

### Surface（3 个信号）

#### Context Viz & Thinkback

**置信度**：中

`ctx_viz`、`thinkback`、`thinkback-play` 暴露了更偏研究和回溯的表面入口。让用户可以可视化上下文分布和回放思考过程。

**路径证据**：`src/commands/ctx_viz`、`src/commands/thinkback`、`src/commands/thinkback-play`

#### UltraPlan Overlay

**置信度**：中

`ultraplan` 命令与计划模式工具并列出现，可能允许在 Opus 级模型上运行长达 30 分钟的规划会话。

**路径证据**：`src/commands/ultraplan`、`src/tools/EnterPlanModeTool/`、`src/tools/ExitPlanModeTool/`

#### Statusline Setup

**置信度**：中

`statusline` 命令和内建 `statuslineSetup` agent 表明终端状态条被视为独立产品面。

**路径证据**：`src/commands/statusline`、`src/tools/AgentTool/built-in/statuslineSetup.ts`

## 信号关系图

```
Partial → Emerging → Integrated → Surface → Context Viz → UltraPlan → Statusline Setup → Bridge & Mailbox → AutoDream → Cron & Triggers → Verification Agent → Computer Use → Plugins → Team Memory Sync → Voice Surface → Coordinator Mode → Buddy Companion
```

## 如何使用这些信号

- Integrated 信号 已经拥有独立专题页——点击上方表格中的链接深入学习
- Emerging 信号 值得持续观察——下一个版本可能会有显著变化
- Partial 和 Surface 信号 不要过度解读——它们可能被废弃，也可能突然成熟
## 建议阅读顺序

- Computer Use — 桌面自动化子系统
- 插件系统 — 可安装扩展的完整工程
- Coordinator 编排 — 多 Worker 编排
- Voice 语音 — 语音输入通道
- MCP 与 Bridge — 扩展织物的两条核心通道
