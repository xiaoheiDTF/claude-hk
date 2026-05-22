# Claude-Code / Advanced / Mcp-Servers

> 来源: claudecn.com

# MCP 服务器

MCP（Model Context Protocol）是 AI 工具集成的开放标准。通过 MCP 服务器，Claude Code 可以连接数据库、API、浏览器、外部工具等数百种数据源。

## MCP 能做什么

连接 MCP 服务器后，可以要求 Claude：

- 从 Issue 跟踪器实现功能：“添加 JIRA issue ENG-4521 描述的功能并创建 GitHub PR”
- 分析监控数据：“检查 Sentry 和 Statsig 查看 ENG-4521 功能的使用情况”
- 查询数据库：“根据 PostgreSQL 数据库找出 10 个使用过 ENG-4521 功能的随机用户邮箱”
- 集成设计稿：“根据 Slack 中发布的新 Figma 设计更新标准邮件模板”
- 自动化工作流：“创建 Gmail 草稿邀请这 10 个用户参加新功能反馈会”
---

## 安装 MCP 服务器

MCP 服务器支持三种传输方式：

### 远程 HTTP 服务器（推荐）

云服务的推荐方式：

```bash
# 基本语法
claude mcp add --transport http <name> <url>

# 示例：连接 Notion
claude mcp add --transport http docs https://your-mcp-server.example.com/mcp

# 带认证头
claude mcp add --transport http secure-api https://api.example.com/mcp \
  --header "Authorization: Bearer your-token"
```

### 远程 SSE 服务器

SSE（Server-Sent Events）传输已弃用，优先使用 HTTP。

```bash
claude mcp add --transport sse server https://your-mcp-server.example.com/sse
```

### 本地 stdio 服务器
在本地运行的进程，适合需要系统访问的工具：

```bash
# 基本语法
claude mcp add [options] <name> -- <command> [args...]

# 示例：添加 Airtable 服务器
claude mcp add --transport stdio --env AIRTABLE_API_KEY=YOUR_KEY airtable \
  -- npx -y airtable-mcp-server
```

**参数顺序**：所有选项（`--transport`、`--env`、`--scope`、`--header`）必须在服务器名称**之前**。`--` 分隔服务器名称和传递给 MCP 服务器的命令参数。

---

## 管理服务器

```bash
# 列出所有配置的服务器
claude mcp list

# 获取特定服务器详情
claude mcp get github

# 移除服务器
claude mcp remove github

# 在 Claude Code 中检查服务器状态
> /mcp
```

---

## 上下文成本与启用数量（经验建议）
每启用一个 MCP 服务器，通常都会带来额外的工具描述与提示词开销。实际使用中，如果你同时打开太多 MCP，主对话的可用上下文会被明显挤占，表现为更容易“健忘/跑偏”。

建议做法：

- 先把启用的 MCP 控制在一个小集合里（例如不超过 10 个），用到再加，确认收益后再长期启用。
- 能用项目级 .mcp.json 固化的，就不要靠个人的 ~/.claude.json 口口相传。
- 所有密钥用占位符或环境变量注入，避免把真实密钥写进可提交文件。
## MCP Tool Search（工具搜索）

当你配置了很多 MCP 服务器时，所有工具定义（tool schemas / descriptions）会占用大量上下文窗口，导致主对话更容易“被挤爆”。Claude Code 的 **MCP Tool Search** 会在需要时再加载工具，而不是把所有工具一次性预加载到上下文里。

### 工作机制（你会感知到什么）

在触发阈值后：

- MCP 工具定义不再全部预加载
- Claude 会通过搜索来定位“应该用哪个 MCP 工具”
- 只把真正用到的工具加载进上下文
- 对你的使用方式基本不变（仍然是“请 Claude 调用工具做事”）
### 配置方式

通过环境变量 `ENABLE_TOOL_SEARCH` 控制：

| 值 | 行为 |
| --- | --- |
| `auto` | 默认：当 MCP 工具定义超过上下文 10% 时启用 |
| `auto:` | 自定义阈值（百分比），例如 `auto:5` |
| `true` | 总是启用 |
| `false` | 禁用（始终预加载所有 MCP 工具） |

```bash
# 使用更低阈值（5%）
ENABLE_TOOL_SEARCH=auto:5 claude

# 完全禁用
ENABLE_TOOL_SEARCH=false claude
```

也可以写到 `settings.json` 的 `env` 里统一生效（适合团队共享）。

Tool Search 需要模型支持 `tool_reference`（例如 Sonnet 4+ / Opus 4+）。Haiku 系列通常不支持该能力。

### 禁用 MCPSearch 工具（可选）
如果你希望强制不使用 Tool Search，可以在权限里显式禁用 `MCPSearch`：

```json
{
  "permissions": {
    "deny": ["MCPSearch"]
  }
}
```

---

## 安装作用域
MCP 服务器可配置在三个不同作用域：

### 本地作用域（默认）

存储在 `~/.claude.json` 的项目路径下，仅在当前项目目录可用：

```bash
# 默认
claude mcp add --transport http server https://your-mcp-server.example.com/mcp

# 显式指定
claude mcp add --transport http server --scope local https://your-mcp-server.example.com/mcp
```

### 项目作用域
存储在项目根目录的 `.mcp.json` 文件，可提交到版本控制，团队共享：

```bash
claude mcp add --transport http server --scope project https://your-mcp-server.example.com/mcp
```

生成的 `.mcp.json`：

```json
{
  "mcpServers": {
    "paypal": {
      "type": "http",
      "url": "https://your-mcp-server.example.com/mcp"
    }
  }
}
```

### 用户作用域
存储在 `~/.claude.json`，跨所有项目可用：

```bash
claude mcp add --transport http server --scope user https://your-mcp-server.example.com/mcp
```

### 作用域优先级
同名服务器的优先级：**本地 > 项目 > 用户**

---

## 环境变量展开

`.mcp.json` 支持环境变量展开：

```json
{
  "mcpServers": {
    "api-server": {
      "type": "http",
      "url": "${API_BASE_URL:-https://api.example.com/mcp}",
      "headers": {
        "Authorization": "Bearer ${API_KEY}"
      }
    }
  }
}
```

支持的语法：

- ${VAR} - 展开为环境变量值
- ${VAR:-default} - 未设置时使用默认值
---

## OAuth 认证

许多云端 MCP 服务器需要认证。Claude Code 支持 OAuth 2.0：

```bash
# 1. 添加需要认证的服务器
claude mcp add --transport http server https://your-mcp-server.example.com/mcp

# 2. 在 Claude Code 中认证
> /mcp
# 在浏览器中完成登录
```

- 认证令牌安全存储并自动刷新
- 在 /mcp 菜单中选择"Clear authentication"撤销访问
---

## 常用 MCP 服务器

第三方 MCP 服务器请自行评估风险。Anthropic 未验证所有服务器的正确性和安全性。

### GitHub

```bash
claude mcp add --transport http server https://your-mcp-server.example.com/mcp/
```

```
> 审查 PR #456 并建议改进
> 创建一个关于我们刚发现的 bug 的 issue
> 显示分配给我的所有待处理 PR
```

### Sentry

```bash
claude mcp add --transport http server https://your-mcp-server.example.com/mcp
```

```
> 过去 24 小时最常见的错误是什么？
> 显示错误 ID abc123 的堆栈跟踪
> 哪个部署引入了这些新错误？
```

### PostgreSQL

```bash
claude mcp add --transport stdio db -- npx -y @bytebase/dbhub \
  --dsn "postgresql://readonly:pass@prod.db.com:5432/analytics"
```

```
> 这个月我们的总收入是多少？
> 显示 orders 表的模式
> 找出 90 天内没有购买的客户
```

### 查找更多
你可以在团队内部维护 MCP server 清单，并为常用系统提供标准化的连接配置；也可以基于 MCP 规范自行构建服务器。

---

## 使用 MCP 资源

MCP 服务器可以暴露资源，使用 @ 提及引用：

```
# 列出可用资源
> @

# 引用特定资源
> 分析 @github:issue://123 并建议修复方案

# 多资源引用
> 比较 @postgres:schema://users 和 @docs:file://database/user-model
```

---

## 把 MCP Prompts 当命令用
部分 MCP 服务器会暴露 prompts（提示模板），它们会以“命令”的形式出现在 Claude Code 里（输入 `/` 时可见），命名格式通常类似：

- /mcp____
这类命令本质上仍然是 MCP 侧提供的 prompts，只是以更方便的方式暴露在交互界面里，便于把“高频提示模板”作为入口复用。

---

## Claude Code 作为 MCP 服务器

可以将 Claude Code 本身作为 MCP 服务器供其他应用连接：

```bash
claude mcp serve
```

在 Claude Desktop 的 `claude_desktop_config.json` 中配置：

```json
{
  "mcpServers": {
    "claude-code": {
      "type": "stdio",
      "command": "claude",
      "args": ["mcp", "serve"],
      "env": {}
    }
  }
}
```

---

## 输出限制
MCP 工具输出超过 10,000 token 时会显示警告。调整限制：

```bash
# 设置更高限制
export MAX_MCP_OUTPUT_TOKENS=50000
claude
```

---

## 从 JSON 添加

```bash
# HTTP 服务器
claude mcp add-json weather-api '{"type":"http","url":"https://api.example.com/mcp","headers":{"Authorization":"Bearer token"}}'

# stdio 服务器
claude mcp add-json local-weather '{"type":"stdio","command":"/path/to/weather-cli","args":["--api-key","abc123"]}'
```

---

## 从 Claude Desktop 导入

```bash
# 导入服务器
claude mcp add-from-claude-desktop

# 选择要导入的服务器
# 验证
claude mcp list
```

仅支持 macOS 和 WSL。

---

## 插件提供的 MCP 服务器

[插件](https://claudecn.com/docs/claude-code/plugins/)可以捆绑 MCP 服务器：

```json
// 插件的 .mcp.json
{
  "database-tools": {
    "command": "${CLAUDE_PLUGIN_ROOT}/servers/db-server",
    "args": ["--config", "${CLAUDE_PLUGIN_ROOT}/config.json"],
    "env": {
      "DB_URL": "${DB_URL}"
    }
  }
}
```

插件 MCP 服务器在启用插件时自动启动。

---

## 调试

```bash
# 启用调试日志
claude --mcp-debug

# 设置启动超时（毫秒）
MCP_TIMEOUT=10000 claude

# 查看服务器状态
> /mcp
```

---

## 延伸阅读

- 站内博客：/blog/claude-skills-vs-mcp-comparison/
- 站内博客：/blog/will-skills-replace-mcp/
- 文档：/docs/claude-code/advanced/skills/
## 下一步
[Agent Skills创建可复用的知识模块
](../skills/)[Subagents创建专用子代理
](../subagents/)[Hooks 系统在事件触发时自动执行脚本
](../hooks/)
