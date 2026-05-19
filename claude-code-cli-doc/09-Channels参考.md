# Channels 参考

> 来源: https://code.claude.com/docs/zh-CN/channels-reference

Channel 是一个 MCP 服务器，它将事件推送到 Claude Code 会话中，以便 Claude 可以对终端外发生的事情做出反应。

---

## Channel 概述

### 使用场景

- **聊天平台**（Telegram、Discord）：在本地运行并轮询平台的 API 以获取新消息
- **Webhooks**（CI、监控）：在本地 HTTP 端口上侦听，外部系统 POST 到该端口

### 单向 vs 双向

- **单向频道**：转发警报、webhooks 或监控事件供 Claude 处理
- **双向频道**：也公开回复工具，以便 Claude 可以发送消息回复
- **中继权限提示**：具有受信任发送者路径的频道可以选择加入中继权限提示，以便您可以远程批准或拒绝工具使用

---

## 服务器要求

1. 声明 `claude/channel` 能力，以便 Claude Code 注册通知侦听器
2. 当发生某事时发出 `notifications/claude/channel` 事件
3. 通过 stdio transport 连接（Claude Code 将您的服务器作为子进程生成）

### 运行时要求

- `@modelcontextprotocol/sdk` 包
- Node.js 兼容的运行时（Bun、Node、Deno）

---

## 研究预览期间测试

在研究预览期间，每个频道都必须在批准的允许列表上才能注册。使用开发标志绕过：

```bash
# 测试您正在开发的插件
claude --dangerously-load-development-channels plugin:yourplugin@yourmarketplace

# 测试裸 .mcp.json 服务器（尚无插件包装器）
claude --dangerously-load-development-channels server:webhook
```

> 绕过是按条目的。将此标志与 `--channels` 结合不会将绕过扩展到 `--channels` 条目。

---

## 服务器选项

频道在 `Server` 构造函数中设置这些选项：

| 字段 | 类型 | 描述 |
|------|------|------|
| `capabilities.experimental['claude/channel']` | `object` | **必需**。始终为 `{}`。存在注册通知侦听器。 |
| `capabilities.experimental['claude/channel/permission']` | `object` | 可选。声明此频道可以接收权限中继请求。 |
| `capabilities.tools` | `object` | 仅双向。始终为 `{}`。标准 MCP 工具能力。 |
| `instructions` | `string` | 推荐。添加到 Claude 的系统提示。告诉 Claude 期望什么事件、`<channel>` 标签属性的含义、是否回复等。 |

### 单向频道示例

```javascript
import { Server } from '@modelcontextprotocol/sdk/server/index.js'

const mcp = new Server(
  { name: 'your-channel', version: '0.0.1' },
  {
    capabilities: {
      experimental: { 'claude/channel': {} },
      // 对于单向频道省略 tools
    },
    instructions: 'Messages arrive as <channel source="your-channel" ...>.',
  },
)
```

### 双向频道示例

```javascript
const mcp = new Server(
  { name: 'your-channel', version: '0.0.1' },
  {
    capabilities: {
      experimental: { 'claude/channel': {} },
      tools: {},  // 双向必需
    },
    instructions: 'Messages arrive as <channel source="your-channel" ...>. Reply with the reply tool.',
  },
)
```

---

## 通知格式

您的服务器使用两个参数发出 `notifications/claude/channel`：

| 字段 | 类型 | 描述 |
|------|------|------|
| `content` | `string` | 事件主体。作为 `<channel>` 标签的主体传递。 |
| `meta` | `Record<string, string>` | 可选。每个条目成为 `<channel>` 标签上的属性。键必须是标识符（仅字母、数字和下划线）。 |

### 推送事件示例

```javascript
await mcp.notification({
  method: 'notifications/claude/channel',
  params: {
    content: 'build failed on main: https://ci.example.com/run/1234',
    meta: { severity: 'high', run_id: '1234' },
  },
})
```

Claude 接收到的格式：

```xml
<channel source="your-channel" severity="high" run_id="1234">
build failed on main: https://ci.example.com/run/1234
</channel>
```

---

## 公开回复工具（双向频道）

双向频道需要三个组件：

1. `Server` 构造函数能力中的 `tools: {}` 条目
2. 定义工具架构并实现发送逻辑的工具处理程序
3. `instructions` 字符串，告诉 Claude 何时以及如何调用工具

### 回复工具示例

```javascript
mcp.setRequestHandler(ListToolsRequestSchema, async () => ({
  tools: [{
    name: 'reply',
    description: 'Send a message back over this channel',
    inputSchema: {
      type: 'object',
      properties: {
        chat_id: { type: 'string', description: 'The conversation to reply in' },
        text: { type: 'string', description: 'The message to send' },
      },
      required: ['chat_id', 'text'],
    },
  }],
}))

mcp.setRequestHandler(CallToolRequestSchema, async req => {
  if (req.params.name === 'reply') {
    const { chat_id, text } = req.params.arguments
    // 发送消息的逻辑
    return { content: [{ type: 'text', text: 'sent' }] }
  }
  throw new Error(`unknown tool: ${req.params.name}`)
})
```

---

## 门控入站消息（安全关键）

未门控的频道是提示注入向量。任何可以到达您的端点的人都可以在 Claude 前面放置文本。

### 发送者检查示例

```javascript
const allowed = new Set(loadAllowlist())

// 在您的消息处理程序中，在发出之前：
if (!allowed.has(message.from.id)) {
  return  // 静默删除
}
await mcp.notification({ ... })
```

> 根据发送者的身份而不是聊天或房间身份进行门控。在群组聊天中，`message.from.id` 和 `message.chat.id` 不同。根据房间进行门控会让允许列表中的任何人向会话注入消息。

---

## 中继权限提示

双向频道可以选择加入以在并行接收工具批准提示，并将其中继到您的另一台设备。

### 工作原理

1. Claude Code 生成一个短请求 ID 并通知您的服务器
2. 您的服务器将提示和 ID 转发到您的聊天应用
3. 远程用户使用该 ID 回复是或否
4. 您的入站处理程序将回复解析为判决

### 启用权限中继

需要三个组件：

1. `Server` 构造函数中的 `'claude/channel/permission': {}` 能力
2. `notifications/claude/channel/permission_request` 的通知处理程序
3. 入站消息处理程序中识别 `yes <id>` 或 `no <id>` 的检查

### 权限请求字段

| 字段 | 描述 |
|------|------|
| `request_id` | 从 `a`-`z` 中抽取的五个小写字母，不包括 `l` |
| `tool_name` | Claude 想要使用的工具的名称 |
| `description` | 人类可读摘要 |
| `input_preview` | 工具的参数作为 JSON 字符串，截断为 200 个字符 |

### 判决格式

您的服务器发送回的判决：

```javascript
await mcp.notification({
  method: 'notifications/claude/channel/permission',
  params: {
    request_id: 'abcde',
    behavior: 'allow',  // 或 'deny'
  },
})
```

> 本地终端对话保持打开。如果终端上的某人在远程判决到达之前回答，该答案将被应用，待处理的远程请求将被删除。

---

## 完整 Webhook 接收器示例

这是一个完整的双向 webhook 接收器，包含回复工具、发送者门控和权限中继：

```typescript
#!/usr/bin/env bun
import { Server } from '@modelcontextprotocol/sdk/server/index.js'
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js'
import { ListToolsRequestSchema, CallToolRequestSchema } from '@modelcontextprotocol/sdk/types.js'
import { z } from 'zod'

// SSE 监听器
const listeners = new Set<(chunk: string) => void>()
function send(text: string) {
  const chunk = text.split('\n').map(l => `data: ${l}\n`).join('') + '\n'
  for (const emit of listeners) emit(chunk)
}

// 发送者允许列表
const allowed = new Set(['dev'])

const mcp = new Server(
  { name: 'webhook', version: '0.0.1' },
  {
    capabilities: {
      experimental: {
        'claude/channel': {},
        'claude/channel/permission': {},
      },
      tools: {},
    },
    instructions: 'Messages arrive as <channel source="webhook" chat_id="...">. Reply with the reply tool.',
  },
)

// 回复工具
mcp.setRequestHandler(ListToolsRequestSchema, async () => ({
  tools: [{
    name: 'reply',
    description: 'Send a message back over this channel',
    inputSchema: {
      type: 'object',
      properties: {
        chat_id: { type: 'string' },
        text: { type: 'string' },
      },
      required: ['chat_id', 'text'],
    },
  }],
}))

mcp.setRequestHandler(CallToolRequestSchema, async req => {
  if (req.params.name === 'reply') {
    const { chat_id, text } = req.params.arguments
    send(`Reply to ${chat_id}: ${text}`)
    return { content: [{ type: 'text', text: 'sent' }] }
  }
  throw new Error(`unknown tool: ${req.params.name}`)
})

// 权限中继
const PermissionRequestSchema = z.object({
  method: z.literal('notifications/claude/channel/permission_request'),
  params: z.object({
    request_id: z.string(),
    tool_name: z.string(),
    description: z.string(),
    input_preview: z.string(),
  }),
})

mcp.setNotificationHandler(PermissionRequestSchema, async ({ params }) => {
  send(
    `Claude wants to run ${params.tool_name}: ${params.description}\n\n` +
    `Reply "yes ${params.request_id}" or "no ${params.request_id}"`,
  )
})

await mcp.connect(new StdioServerTransport())

// HTTP 服务器
const PERMISSION_REPLY_RE = /^\s*(y|yes|n|no)\s+([a-km-z]{5})\s*$/i
let nextId = 1

Bun.serve({
  port: 8788,
  hostname: '127.0.0.1',
  idleTimeout: 0,
  async fetch(req) {
    const url = new URL(req.url)

    // GET /events：SSE 流
    if (req.method === 'GET' && url.pathname === '/events') {
      const stream = new ReadableStream({
        start(ctrl) {
          ctrl.enqueue(': connected\n\n')
          const emit = (chunk: string) => ctrl.enqueue(chunk)
          listeners.add(emit)
          req.signal.addEventListener('abort', () => listeners.delete(emit))
        },
      })
      return new Response(stream, {
        headers: { 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache' },
      })
    }

    // 入站处理
    const body = await req.text()
    const sender = req.headers.get('X-Sender') ?? ''
    if (!allowed.has(sender)) return new Response('forbidden', { status: 403 })

    // 检查判决格式
    const m = PERMISSION_REPLY_RE.exec(body)
    if (m) {
      await mcp.notification({
        method: 'notifications/claude/channel/permission',
        params: {
          request_id: m[2].toLowerCase(),
          behavior: m[1].toLowerCase().startsWith('y') ? 'allow' : 'deny',
        },
      })
      return new Response('verdict recorded')
    }

    // 正常聊天事件
    const chat_id = String(nextId++)
    await mcp.notification({
      method: 'notifications/claude/channel',
      params: { content: body, meta: { chat_id, path: url.pathname } },
    })
    return new Response('ok')
  },
})
```

### 测试步骤

1. 启动 Claude Code 会话：
```bash
claude --dangerously-load-development-channels server:webhook
```

2. 观看出站端：
```bash
curl -N localhost:8788/events
```

3. 发送消息：
```bash
curl -d "list the files in this directory" -H "X-Sender: dev" localhost:8788
```

4. 远程批准工具：
```bash
curl -d "yes <id>" -H "X-Sender: dev" localhost:8788
```

---

## 打包为 Plugin

要将频道可安装和可共享，将其包装在 plugin 中并发布到市场。

- 用户使用 `/plugin install` 安装它
- 使用 `--channels plugin:<name>@<marketplace>` 按会话启用它
- 发布到您自己的市场的频道仍然需要 `--dangerously-load-development-channels` 来运行
- 要添加到官方允许列表，将其提交到官方市场
- 在 Team 和 Enterprise 计划上，管理员可以将其包含在组织自己的 `allowedChannelPlugins` 列表中
