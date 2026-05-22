# Claude-Code / Plugins / Create-Plugins

> 来源: claudecn.com

# 创建插件

插件让你可以用自定义斜杠命令、代理、Skills、Hooks 和 MCP 服务器扩展 Claude Code。本指南介绍如何创建自己的插件。

## 何时使用插件 vs 独立配置

| 方式 | 斜杠命令格式 | 适用场景 |
| --- | --- | --- |
| **独立配置** (`.claude/` 目录) | `/hello` | 个人工作流、项目特定定制 |
| **插件** (`.claude-plugin/plugin.json`) | `/plugin-name:hello` | 团队共享、社区分发、跨项目复用 |

建议先在 `.claude/` 目录中快速迭代，准备好分享时再转换为插件。

---

## 快速开始：创建第一个插件

### 第一步：创建插件目录

```bash
mkdir my-first-plugin
```

### 第二步：创建插件清单
清单文件 `.claude-plugin/plugin.json` 定义插件的身份：

```bash
mkdir my-first-plugin/.claude-plugin
```

创建 `my-first-plugin/.claude-plugin/plugin.json`：

```json
{
  "name": "my-first-plugin",
  "description": "学习基础的问候插件",
  "version": "1.0.0",
  "author": {
    "name": "Your Name"
  }
}
```

| 字段 | 用途 |
| --- | --- |
| `name` | 唯一标识符，也是斜杠命令的命名空间（如 `/my-first-plugin:hello`） |
| `description` | 在插件管理器中显示 |
| `version` | 使用语义化版本 |

### 第三步：添加斜杠命令
斜杠命令放在 `commands/` 目录中，文件名即命令名：

```bash
mkdir my-first-plugin/commands
```

创建 `my-first-plugin/commands/hello.md`：

```markdown
---
description: 用友好的消息问候用户
---

# Hello 命令

用热情的方式问候用户，询问今天可以帮助什么。
```

### 第四步：测试插件
使用 `--plugin-dir` 加载开发中的插件：

```bash
claude --plugin-dir ./my-first-plugin
```

启动后，试试新命令：

```
> /my-first-plugin:hello
```

**为什么要命名空间？** 插件命令始终带命名空间前缀（如 `/greet:hello`），防止多个插件的同名命令冲突。

### 第五步：添加命令参数
使用 `$ARGUMENTS` 占位符捕获用户输入：

```markdown
---
description: 用个性化消息问候用户
---

# Hello 命令

用热情的方式问候名为 "$ARGUMENTS" 的用户。
```

测试：

```
> /my-first-plugin:hello 张三
```

---

## 插件结构概览
插件可以包含多种组件：

| 目录 | 用途 |
| --- | --- |
| `.claude-plugin/` | 包含 `plugin.json` 清单文件（**必需**） |
| `commands/` | 自定义斜杠命令 |
| `agents/` | 自定义子代理 |
| `skills/` | Agent Skills |
| `hooks/` | 事件钩子配置 |
| `.mcp.json` | MCP 服务器定义 |
| `.lsp.json` | LSP 服务器配置 |

**常见错误**：不要把 `commands/`、`agents/`、`skills/` 放在 `.claude-plugin/` 目录里。只有 `plugin.json` 放在 `.claude-plugin/` 里，其他目录都在插件根目录。

---

## 添加代理
在 `agents/` 目录创建 Markdown 文件：

```markdown
---
description: 代码质量和安全审查专家
capabilities: ["code-review", "security-check"]
---

# Code Reviewer

详细描述代理的角色、专业领域和调用时机。

## 能力

- 代码质量审查
- 安全漏洞检测
- 最佳实践建议
```

---

## 添加 Skills
在 `skills/` 目录创建子目录，每个 Skill 包含 `SKILL.md`：

```
skills/
└── pdf-processor/
    ├── SKILL.md
    ├── reference.md （可选）
    └── scripts/ （可选）
```

---

## 添加 Hooks
创建 `hooks/hooks.json`：

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/scripts/format-code.sh"
          }
        ]
      }
    ]
  }
}
```

**重要**：使用 `${CLAUDE_PLUGIN_ROOT}` 变量引用插件目录路径。

---

## 添加 MCP 服务器

创建 `.mcp.json`：

```json
{
  "mcpServers": {
    "plugin-database": {
      "command": "${CLAUDE_PLUGIN_ROOT}/servers/db-server",
      "args": ["--config", "${CLAUDE_PLUGIN_ROOT}/config.json"]
    }
  }
}
```

---

## 完整 plugin.json 示例

```json
{
  "name": "enterprise-tools",
  "version": "2.1.0",
  "description": "企业开发工具集",
  "author": {
    "name": "开发团队",
    "email": "dev@company.com"
  },
  "homepage": "https://docs.example.com/plugin",
  "repository": "https://git.example.com/company/plugin",
  "license": "MIT",
  "keywords": ["enterprise", "development"],
  "commands": ["./custom/commands/special.md"],
  "agents": "./custom/agents/",
  "skills": "./custom/skills/",
  "hooks": "./config/hooks.json",
  "mcpServers": "./mcp-config.json"
}
```

---

## 调试插件

```bash
claude --debug
```

查看：

- 加载了哪些插件
- 清单文件中的错误
- 命令、代理、钩子注册情况
- MCP 服务器初始化
---

## 下一步
[发现和安装插件从市场浏览和安装插件
](../discover-plugins/)[插件参考完整技术规范
](../plugins-reference/)
