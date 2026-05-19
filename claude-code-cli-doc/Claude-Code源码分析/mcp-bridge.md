# Source-Analysis / Mcp-Bridge

> 来源: claudecn.com

# MCP 与 Bridge

MCP 集成和 Bridge 跨界面连接构成了 Claude Code 扩展织物的两条核心通道——一条向外接入外部工具能力，一条向内连接多端界面。

## 核心问题

Agent 系统如何在不修改核心代码的情况下接入外部能力？如何让同一个 agent 同时支持 CLI、IDE、Desktop 和 Mobile？MCP 和 Bridge 分别回答了这两个问题。

## MCP 集成

### 传输层

Claude Code 支持 6 种 MCP 传输方式：

| 传输 | 说明 |
| --- | --- |
| **stdio** | 标准输入输出，本地进程通信 |
| **SSE** | Server-Sent Events，HTTP 长连接 |
| **HTTP** | 标准 HTTP 请求/响应 |
| **WebSocket** | 全双工通信 |
| **SDK** | Anthropic SDK 原生集成 |
| **claude.ai proxy** | 通过 claude.ai 中转 |

### 配置范围

MCP 服务器可以在 7 个配置范围中注册：

| 范围 | 生效范围 | 典型用途 |
| --- | --- | --- |
| 全局用户配置 | 所有项目 | 通用工具（搜索、浏览器） |
| 项目 `.mcp.json` | 单个项目 | 项目专用工具 |
| CLAUDE.md 声明 | 项目/本地 | 轻量级工具声明 |
| 插件内嵌 | 插件范围 | 插件附带的 MCP 服务器 |
| 环境变量 | 会话级 | 临时覆盖 |
| SDK 模式 | 程序化使用 | SDK 客户端提供 |
| IDE 配置 | IDE 范围 | VS Code / JetBrains 集成 |

### 工具命名与注册

MCP 工具以 `mcp__<server>__<tool>` 模式命名，动态注册到工具系统中。这意味着 MCP 工具与内建工具共用：

- 相同的权限检查流程
- 相同的执行编排（并行/串行分区）
- 相同的结果预算机制
- 相同的 Hook 拦截点
这种"统一工具面"的设计让外部工具无法绕过治理体系。

### OAuth 2.0 + PKCE

远程 MCP 服务器的认证采用 OAuth 2.0 + PKCE 流程，支持：

- 授权码模式（适合有浏览器的环境）
- 设备码模式（适合纯 CLI 环境）
- Token 缓存与刷新
## Bridge 跨界面连接

### 多端架构

Bridge 不是实验功能，而是已集成的跨界面连接能力。它让同一个 Claude Code 实例能够被多种界面访问：

| 连接端 | 命令入口 | 用途 |
| --- | --- | --- |
| **IDE** | `src/commands/ide` | VS Code、JetBrains 等编辑器 |
| **Desktop** | `src/commands/desktop` | Electron 桌面应用 |
| **Mobile** | `src/commands/mobile` | 移动端控制 |
| **Chrome** | `src/commands/chrome` | 浏览器扩展 |

### 邮箱机制

Bridge 通过 Mailbox 实现跨界面消息传递：

- src/utils/mailbox.ts — 消息邮箱核心
- src/context/mailbox.tsx — 邮箱 UI 上下文
消息邮箱是异步的——发送方不需要等待接收方在线。这使得 CLI 可以在后台运行，而 IDE 界面随时连接/断开。

```
界面通道 → MCP 通道 → Claude Code 核心 → Agent Loop → Bridge 服务 → src/bridge/ → Mailbox → 消息队列 → stdio MCP Server → HTTP MCP Server → claude.ai Proxy → CLI 终端 → VS Code / JetBrains → Desktop App → Mobile
```

## 两条通道的交汇

MCP 和 Bridge 在扩展织物中的角色不同但互补：

| 维度 | MCP | Bridge |
| --- | --- | --- |
| **方向** | 向外：接入外部能力 | 向内：连接多端界面 |
| **注册时机** | 配置时或运行时发现 | 进程启动时 |
| **通信模式** | 请求/响应 | 消息邮箱（异步） |
| **治理** | 统一工具面 | 独立连接管理 |

两者在一些场景中交汇：IDE 可以通过 Bridge 连接到 Claude Code，而 Claude Code 又通过 MCP 连接到 IDE 提供的 LSP 服务。

## 对 agent 开发者的启示

| 模式 | 说明 |
| --- | --- |
| **统一工具面** | 外部工具（MCP）和内建工具共用同一套治理，不设后门 |
| **多传输适配** | 不要假设用户只有一种网络环境——支持从 stdio 到 WebSocket 的多种传输 |
| **异步邮箱** | 跨界面通信不要求双方同时在线——消息队列是更稳健的选择 |
| **配置分层** | 全局 → 项目 → 插件 → 会话，让用户在合适的范围覆盖配置 |

## 路径证据

| 路径 | 职责 |
| --- | --- |
| `src/services/mcp/` | MCP 集成核心 |
| `src/bridge/` | Bridge 服务核心 |
| `src/utils/mailbox.ts` | 消息邮箱机制 |
| `src/context/mailbox.tsx` | 邮箱 UI 上下文 |
| `src/commands/bridge` | Bridge 命令入口 |
| `src/commands/ide` | IDE 集成命令 |

## 进一步阅读

- 架构地图 — 扩展织物在六层结构中的位置
- 工具平面 — MCP 工具的注册和治理
- 插件系统 — 插件如何内嵌 MCP 服务器
- 扩展与信号 — Bridge 的成熟度判定
