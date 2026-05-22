# Plugins 参考

> 来源: https://code.claude.com/docs/zh-CN/plugins-reference

本文档提供 Claude Code 插件系统的完整技术规范，包括组件架构、CLI 命令和开发工具。

---

## Plugin 组件类型

### 1. Skills

Plugins 向 Claude Code 添加 skills，创建可由您或 Claude 调用的 `/name` 快捷方式。

**位置**：插件根目录中的 `skills/` 或 `commands/` 目录
**文件格式**：Skills 是包含 `SKILL.md` 的目录；commands 是简单的 markdown 文件

**结构**：
```
skills/
├── pdf-processor/
│   ├── SKILL.md
│   ├── reference.md (可选)
│   └── scripts/ (可选)
└── code-reviewer/
    └── SKILL.md
```

**集成行为**：
- 安装插件时会自动发现 Skills 和 commands
- Claude 可以根据任务上下文自动调用它们
- Skills 可以在 SKILL.md 旁边包含支持文件

### 2. Agents

Plugins 可以为特定任务提供专门的 subagents。

**位置**：插件根目录中的 `agents/` 目录
**文件格式**：描述 agent 功能的 Markdown 文件

**Frontmatter 字段**：
- `name`、`description`、`model`、`effort`、`maxTurns`
- `tools`、`disallowedTools`、`skills`、`memory`、`background`
- `isolation`（唯一有效值是 `"worktree"`）

> Plugin agents 不支持 `hooks`、`mcpServers` 和 `permissionMode`（出于安全原因）。

### 3. Hooks

Plugins 可以提供事件处理程序。

**位置**：插件根目录中的 `hooks/hooks.json`，或在 plugin.json 中内联

**Hook 类型**：`command`、`http`、`mcp_tool`、`prompt`、`agent`

### 4. MCP Servers

Plugins 可以捆绑 Model Context Protocol (MCP) servers。

**位置**：插件根目录中的 `.mcp.json`，或在 plugin.json 中内联

**配置示例**：
```json
{
  "mcpServers": {
    "plugin-database": {
      "command": "${CLAUDE_PLUGIN_ROOT}/servers/db-server",
      "args": ["--config", "${CLAUDE_PLUGIN_ROOT}/config.json"],
      "env": {
        "DB_PATH": "${CLAUDE_PLUGIN_ROOT}/data"
      }
    }
  }
}
```

### 5. LSP Servers

Plugins 可以提供 Language Server Protocol (LSP) servers，提供实时代码智能。

**位置**：插件根目录中的 `.lsp.json`，或在 `plugin.json` 中内联

**可用 LSP Plugins**：

| Plugin | 语言服务器 | 安装命令 |
|--------|-----------|---------|
| `pyright-lsp` | Pyright (Python) | `pip install pyright` 或 `npm install -g pyright` |
| `typescript-lsp` | TypeScript Language Server | `npm install -g typescript-language-server typescript` |
| `rust-analyzer-lsp` | rust-analyzer | 参阅 rust-analyzer 安装 |

### 6. Monitors（实验性）

Plugins 可以声明后台 monitors，在 plugin 激活时自动启动。

**位置**：插件根目录中的 `monitors/monitors.json`，或在 plugin.json 中内联

**字段**：
- `name`（必需）：唯一标识符
- `command`（必需）：持久后台进程运行的 shell 命令
- `description`（必需）：简短摘要
- `when`（可选）：控制 monitor 何时启动。`"always"` 或 `"on-skill-invoke:<skill-name>"`

### 7. Themes（实验性）

Plugins 可以提供颜色主题。

**位置**：`themes/` 中的 JSON 文件

```json
{
  "name": "Dracula",
  "base": "dark",
  "overrides": {
    "claude": "#bd93f9",
    "error": "#ff5555",
    "success": "#50fa7b"
  }
}
```

---

## Plugin 安装范围

| 范围 | 设置文件 | 用例 |
|------|---------|------|
| `user` | `~/.claude/settings.json` | 在所有项目中可用的个人 plugins（默认） |
| `project` | `.claude/settings.json` | 通过版本控制共享的团队 plugins |
| `local` | `.claude/settings.local.json` | 项目特定的 plugins，gitignored |
| `managed` | Managed settings | 托管 plugins（只读，仅更新） |

---

## Plugin CLI 命令

### 安装

```bash
# 安装到用户范围（默认）
claude plugin install formatter@my-marketplace

# 安装到项目范围（与团队共享）
claude plugin install formatter@my-marketplace --scope project

# 安装到本地范围（gitignored）
claude plugin install formatter@my-marketplace --scope local
```

### 卸载

```bash
claude plugin uninstall <plugin> [options]
```

选项：
- `-s, --scope <scope>`：从范围卸载
- `--keep-data`：保留插件的持久数据目录
- `--prune`：同时删除其他 plugin 不需要的自动安装依赖项
- `-y, --yes`：跳过确认提示

### 修剪依赖

```bash
claude plugin prune [options]
```

选项：
- `-s, --scope <scope>`
- `--dry-run`：列出将被删除的内容而不实际删除
- `-y, --yes`：跳过确认提示

> 要在一个步骤中删除 plugin 并清理其依赖项，请运行 `claude plugin uninstall <plugin> --prune`。

### 启用/禁用

```bash
claude plugin enable <plugin> [-s <scope>]
claude plugin disable <plugin> [-s <scope>]
```

### 更新

```bash
claude plugin update <plugin> [-s <scope>]
```

### 列出

```bash
claude plugin list [--json] [--available]
```

### 标记发布

```bash
claude plugin tag [--push] [--dry-run] [-f, --force]
```

---

## Plugin 清单架构（plugin.json）

### 必需字段

| 字段 | 类型 | 描述 | 示例 |
|------|------|------|------|
| `name` | string | 唯一标识符（kebab-case，无空格） | `"deployment-tools"` |

### 元数据字段

| 字段 | 类型 | 描述 |
|------|------|------|
| `$schema` | string | 用于编辑器自动完成和验证的 JSON Schema URL |
| `version` | string | 语义版本。设置后将 plugin 固定到该版本 |
| `description` | string | plugin 目的的简要说明 |
| `author` | object | 作者信息 |
| `homepage` | string | 文档 URL |
| `repository` | string | 源代码 URL |
| `license` | string | 许可证标识符 |
| `keywords` | array | 发现标签 |

### 组件路径字段

| 字段 | 类型 | 描述 |
|------|------|------|
| `skills` | string\|array | 自定义 skill 目录 |
| `commands` | string\|array | 自定义平面 `.md` skill 文件或目录 |
| `agents` | string\|array | 自定义 agent 文件 |
| `hooks` | string\|array\|object | Hook 配置路径或内联配置 |
| `mcpServers` | string\|array\|object | MCP 配置路径或内联配置 |
| `outputStyles` | string\|array | 自定义输出样式文件/目录 |
| `lspServers` | string\|array\|object | LSP 配置 |
| `experimental.themes` | string\|array | 颜色主题文件/目录 |
| `experimental.monitors` | string\|array | 后台 Monitor 配置 |

### 用户配置

`userConfig` 字段声明了 Claude Code 在启用 plugin 时提示用户的值：

```json
{
  "userConfig": {
    "api_endpoint": {
      "type": "string",
      "title": "API endpoint",
      "description": "Your team's API endpoint"
    },
    "api_token": {
      "type": "string",
      "title": "API token",
      "description": "API authentication token",
      "sensitive": true
    }
  }
}
```

**字段选项**：
- `type`（必需）：`string`、`number`、`boolean`、`directory`、`file`
- `title`（必需）：配置对话框中显示的标签
- `description`（必需）：字段下方的帮助文本
- `sensitive`：如果为 `true`，掩盖输入并存储在安全存储中
- `required`、`default`、`multiple`、`min`/`max`

每个值都可用于在 MCP 和 LSP server 配置、hook 命令和 monitor 命令中作为 `${user_config.KEY}` 进行替换。

---

## 环境变量

| 变量 | 说明 |
|------|------|
| `${CLAUDE_PLUGIN_ROOT}` | plugin 安装目录的绝对路径。使用此路径引用与 plugin 捆绑的脚本、二进制文件和配置文件 |
| `${CLAUDE_PLUGIN_DATA}` | 用于 plugin 状态的持久目录，在更新后保留。使用此目录用于已安装的依赖项、生成的代码、缓存 |

### 持久数据目录技巧

数据目录解析为 `~/.claude/plugins/data/{id}/`。

**推荐模式**：将捆绑的清单与数据目录中的副本进行比较，并在它们不同时重新安装：

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "diff -q \"${CLAUDE_PLUGIN_ROOT}/package.json\" \"${CLAUDE_PLUGIN_DATA}/package.json\" >/dev/null 2>&1 || (cd \"${CLAUDE_PLUGIN_DATA}\" && cp \"${CLAUDE_PLUGIN_ROOT}/package.json\" . && npm install) || rm -f \"${CLAUDE_PLUGIN_DATA}/package.json\""
          }
        ]
      }
    ]
  }
}
```

---

## 目录结构

```
enterprise-plugin/
├── .claude-plugin/           # 元数据目录（可选）
│   └── plugin.json             # plugin 清单
├── skills/                   # Skills
│   ├── code-reviewer/
│   │   └── SKILL.md
│   └── pdf-processor/
│       ├── SKILL.md
│       └── scripts/
├── commands/                 # Skills 作为平面 .md 文件
├── agents/                   # Subagent 定义
├── output-styles/            # 输出样式定义
├── themes/                   # 颜色主题定义
├── monitors/                 # 后台 monitor 配置
├── hooks/                    # Hook 配置
├── bin/                      # 添加到 PATH 的可执行文件
├── settings.json            # plugin 的默认设置
├── .mcp.json                # MCP server 定义
├── .lsp.json                # LSP server 配置
├── scripts/                 # Hook 和实用脚本
├── LICENSE                  # 许可证文件
└── CHANGELOG.md             # 版本历史
```

> plugin 根目录中的 `CLAUDE.md` 文件不会作为项目上下文加载。Plugins 通过 skills、agents 和 hooks 来贡献上下文。

---

## 版本管理

| 方法 | 如何操作 | 最适合 |
|------|---------|--------|
| **显式版本** | 在 `plugin.json` 中设置 `"version": "2.1.0"` | 具有稳定发布周期的已发布 plugin |
| **提交 SHA 版本** | 从 `plugin.json` 和市场条目中省略 `version` | 正在积极开发的内部或团队 plugin |

版本解析优先级：
1. `plugin.json` 中的 `version` 字段
2. `marketplace.json` 中的 `version` 字段
3. Git 提交 SHA
4. `unknown`

---

## 常见问题排查

| 问题 | 原因 | 解决方案 |
|------|------|---------|
| Plugin 未加载 | 无效的 `plugin.json` | 运行 `claude plugin validate` 检查语法和架构错误 |
| Skills 未出现 | 目录结构错误 | 确保 `skills/` 或 `commands/` 在根目录 |
| Hooks 未触发 | 脚本不可执行 | 运行 `chmod +x script.sh` |
| MCP server 失败 | 缺少 `${CLAUDE_PLUGIN_ROOT}` | 对所有 plugin 路径使用变量 |
| 路径错误 | 使用了绝对路径 | 所有路径必须是相对的，并以 `./` 开头 |
| LSP `Executable not found in $PATH` | 语言服务器未安装 | 安装二进制文件 |

### 路径遍历限制

已安装的 plugins 无法引用其目录外的文件。要访问外部文件，在 plugin 目录中创建指向外部文件的符号链接：

```bash
ln -s /path/to/shared-utils ./shared-utils
```
