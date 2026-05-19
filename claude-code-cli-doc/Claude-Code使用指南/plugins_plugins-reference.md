# Claude-Code / Plugins / Plugins-Reference

> 来源: claudecn.com

# 插件参考

Claude Code 插件系统完整技术规范，包括组件架构、CLI 命令和开发工具。

## 插件组件参考

### 命令 (Commands)

**位置**：`commands/` 目录

**文件格式**：带 frontmatter 的 Markdown

```markdown
---
description: 命令描述
---

# 命令名

命令的详细指令...
```

### 代理 (Agents)
**位置**：`agents/` 目录

**文件格式**：描述代理能力的 Markdown

```markdown
---
description: 代理专长
capabilities: ["task1", "task2"]
---

# Agent Name

详细描述角色和调用时机。
```

### Skills
**位置**：`skills/` 目录，每个 Skill 一个子目录，包含 `SKILL.md`

```
skills/
├── pdf-processor/
│   ├── SKILL.md
│   └── scripts/
└── code-reviewer/
    └── SKILL.md
```

### Hooks
**位置**：`hooks/hooks.json` 或内联在 `plugin.json`

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/scripts/format.sh"
          }
        ]
      }
    ]
  }
}
```

**可用事件**：

| 事件 | 触发时机 |
| --- | --- |
| `PreToolUse` | 工具使用前 |
| `PostToolUse` | 工具使用后 |
| `PermissionRequest` | 权限对话框显示时 |
| `UserPromptSubmit` | 用户提交提示时 |
| `Stop` | Claude 尝试停止时 |
| `SessionStart` | 会话开始时 |
| `SessionEnd` | 会话结束时 |

**钩子类型**：

- command：执行 shell 命令
- prompt：使用 LLM 评估提示
- agent：运行带工具的验证代理
### MCP 服务器

**位置**：`.mcp.json` 或内联在 `plugin.json`

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

### LSP 服务器
**位置**：`.lsp.json` 或内联在 `plugin.json`

```json
{
  "go": {
    "command": "gopls",
    "args": ["serve"],
    "extensionToLanguage": {
      ".go": "go"
    }
  }
}
```

**必须单独安装语言服务器二进制**。LSP 插件只配置连接方式，不包含服务器本身。

---

## 安装范围

| 范围 | 设置文件 | 适用场景 |
| --- | --- | --- |
| `user` | `~/.claude/settings.json` | 个人插件，跨项目可用（默认） |
| `project` | `.claude/settings.json` | 团队插件，通过版本控制共享 |
| `local` | `.claude/settings.local.json` | 项目特定插件，gitignore |
| `managed` | `managed-settings.json` | 企业管理插件（只读） |

---

## plugin.json 清单模式

### 完整示例

```json
{
  "name": "plugin-name",
  "version": "1.2.0",
  "description": "简要描述",
  "author": {
    "name": "作者名",
    "email": "author@example.com"
  },
  "homepage": "https://docs.example.com/plugin",
  "repository": "https://git.example.com/author/plugin",
  "license": "MIT",
  "keywords": ["keyword1", "keyword2"],
  "commands": ["./custom/commands/special.md"],
  "agents": "./custom/agents/",
  "skills": "./custom/skills/",
  "hooks": "./config/hooks.json",
  "mcpServers": "./mcp-config.json",
  "lspServers": "./.lsp.json"
}
```

### 必填字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `name` | string | 唯一标识符（kebab-case，无空格） |

### 元数据字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `version` | string | 语义化版本 |
| `description` | string | 简要说明插件用途 |
| `author` | object | 作者信息 |
| `homepage` | string | 文档 URL |
| `repository` | string | 源码 URL |
| `license` | string | 许可证标识符 |
| `keywords` | array | 发现标签 |

### 组件路径字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `commands` | string/array | 额外命令文件/目录 |
| `agents` | string/array | 额外代理文件 |
| `skills` | string/array | 额外 Skill 目录 |
| `hooks` | string/object | 钩子配置路径或内联配置 |
| `mcpServers` | string/object | MCP 配置路径或内联配置 |
| `lspServers` | string/object | LSP 配置路径或内联配置 |

### 环境变量
**`${CLAUDE_PLUGIN_ROOT}`**：插件目录的绝对路径。在钩子、MCP 服务器和脚本中使用，确保路径正确。

---

## 插件目录结构

```
enterprise-plugin/
├── .claude-plugin/           # 元数据目录
│   └── plugin.json           # 必需：插件清单
├── commands/                 # 默认命令位置
├── agents/                   # 默认代理位置
├── skills/                   # Agent Skills
├── hooks/                    # 钩子配置
│   └── hooks.json
├── .mcp.json                 # MCP 服务器定义
├── .lsp.json                 # LSP 服务器配置
├── scripts/                  # 钩子和工具脚本
├── LICENSE
└── CHANGELOG.md
```

`.claude-plugin/` 目录只包含 `plugin.json`。其他目录（commands/、agents/、skills/、hooks/）必须在插件根目录。

---

## CLI 命令参考

### plugin install

```bash
claude plugin install <plugin> [options]
```

| 选项 | 说明 | 默认值 |
| --- | --- | --- |
| `-s, --scope ` | 安装范围：`user`、`project`、`local` | `user` |

### plugin uninstall

```bash
claude plugin uninstall <plugin> [options]
```

别名：`remove`、`rm`

### plugin enable / disable

```bash
claude plugin enable <plugin> [options]
claude plugin disable <plugin> [options]
```

### plugin update

```bash
claude plugin update <plugin> [options]
```

---

## 调试和开发工具

```bash
claude --debug
```

显示：

- 加载了哪些插件
- 清单文件中的错误
- 命令、代理、钩子注册
- MCP 服务器初始化
### 常见问题

| 问题 | 原因 | 解决方案 |
| --- | --- | --- |
| 插件不加载 | 无效的 `plugin.json` | 检查 JSON 语法 |
| 命令不显示 | 目录结构错误 | `commands/` 应在根目录，不在 `.claude-plugin/` |
| 钩子不触发 | 脚本不可执行 | `chmod +x script.sh` |
| MCP 服务器失败 | 路径问题 | 使用 `${CLAUDE_PLUGIN_ROOT}` |
| LSP 找不到可执行文件 | 语言服务器未安装 | 安装对应的二进制文件 |

---

## 版本管理

遵循语义化版本：

- MAJOR：破坏性更改
- MINOR：新功能（向后兼容）
- PATCH：Bug 修复
最佳实践：

- 从 1.0.0 开始首个稳定版本
- 分发更改前更新 plugin.json 中的版本
- 在 CHANGELOG.md 中记录更改
- 测试时使用预发布版本如 2.0.0-beta.1
---

## 相关文档
[创建插件开发自定义插件
](../create-plugins/)[发现和安装插件从市场安装插件
](../discover-plugins/)[Agent Skills扩展 Claude 能力
](https://claudecn.com/docs/claude-code/advanced/skills/)[Hooks 系统事件处理和自动化
](https://claudecn.com/docs/claude-code/advanced/hooks/)[MCP 服务器外部工具集成
](https://claudecn.com/docs/claude-code/advanced/mcp-servers/)
