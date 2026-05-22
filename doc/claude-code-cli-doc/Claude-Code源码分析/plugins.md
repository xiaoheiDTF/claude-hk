# Source-Analysis / Plugins

> 来源: claudecn.com

# 插件系统

从技能文件到可分发的扩展包——Plugins 是 Claude Code 扩展架构的顶层容器，解决的不是"如何定义一个能力"，而是"如何发现、信任、安装、更新和卸载一组能力"。

## 核心问题

Skills 让你用 Markdown 定义一个可复用的提示词模板。但当你想把一组 Skills、几个 Hook、一对 MCP 服务器和一套自定义命令打包成一个可分发的产品时，你需要的是 Plugins。

Plugins 回答的是一系列更难的问题：**如何让一千个用户使用同一个插件而不互相干扰？如何在安装前验证它是否安全？如何升级时不破坏已有配置？**

## 子系统全景

| 指标 | 数值 |
| --- | --- |
| **核心服务文件** | 3 个主文件 + Schema 定义 |
| **目录位置** | `src/services/plugins/`、`src/utils/plugins/` |
| **成熟度** | Integrated（已集成到主干） |
| **清单 Schema** | ~1,700 行 Zod 定义 |

## 插件清单架构

插件的一切始于 `plugin.json` 清单文件。其验证 Schema 是 Claude Code 中最大的单个 Schema 定义（`schemas.ts` 约 1,681 行），由 11 个子 Schema 组合：

| 子 Schema | 说明 |
| --- | --- |
| **Metadata** | 名称、版本、作者、关键词、依赖（必填） |
| **Hooks** | Hook 定义（`hooks.json` 或内联） |
| **Commands** | 自定义命令（`commands/*.md`） |
| **Agents** | 自定义 agent 定义 |
| **Skills** | 技能文件（`skills/**/SKILL.md`） |
| **Output Styles** | 输出样式定制 |
| **Channels** | MCP 消息注入 |
| **MCP Servers** | MCP 服务器配置 |
| **LSP Servers** | LSP 服务器配置 |
| **Settings** | 预设值 |
| **User Config** | 安装时提示用户配置 |

除 Metadata 外，所有子 Schema 都使用 `.partial()`——只有 Hook 的插件和提供完整工具链的插件共享同一个格式。

```
plugin.json → PluginManifest → Metadata → name, version, author → Hooks → 生命周期拦截 → Commands → 自定义命令 → Agents → agent 定义 → Skills → SKILL.md 文件 → MCP Servers → 工具服务 → LSP Servers → 语言服务 → Settings → 预设配置
```

## 生命周期管理

| 文件 | 职责 |
| --- | --- |
| `PluginInstallationManager.ts` | 安装、卸载、版本管理 |
| `pluginCliCommands.ts` | 命令行入口（`install` / `uninstall` / `list`） |
| `pluginOperations.ts` | 运行时加载和操作逻辑 |

### 安装源

插件可以从多种来源安装：本地目录、Git 仓库、npm 包、市场（Marketplace）等。安装过程包含验证链：

- 清单验证：Zod Schema 全量校验
- 路径安全：所有文件路径必须以 ./ 开头，不能包含 ..
- 名称保护：不能冒充官方市场名称（anthropic-marketplace 等保留名）
- 版本隔离：不同版本使用独立缓存目录
### 错误处理

插件加载使用 25 种 discriminated union 错误类型。这不是简单的 `try/catch`——每种失败模式都有结构化的错误信息，便于诊断和恢复。

## Plugins vs Skills 对比

| 维度 | Skills | Plugins |
| --- | --- | --- |
| **粒度** | 单个提示词模板 | 一组能力的打包容器 |
| **分发** | 项目级 `.claude/skills/` | 跨项目可安装包 |
| **生命周期** | 发现 → 调用 | 安装 → 加载 → 运行 → 更新 → 卸载 |
| **信任边界** | 项目内信任 | 需要验证签名和来源 |
| **组合能力** | 单一技能 | 技能 + 命令 + Hook + MCP + LSP |
| **定义格式** | SKILL.md（YAML + Markdown） | plugin.json（Zod Schema 验证） |

## 关键设计决策

### 渐进式组合

插件不要求实现所有 11 种组件。`.partial()` 设计让最简单的插件只需要一个 `metadata` 加一个 `hooks` 配置，而最复杂的可以提供完整的工具链。

### 市场名称保留

```
ALLOWED_OFFICIAL_MARKETPLACE_NAMES:
  claude-code-marketplace, claude-code-plugins,
  claude-plugins-official, anthropic-marketplace,
  anthropic-plugins, agent-skills, ...
```

保留名机制 + `inline`/`builtin` 两个特殊名保护，防止第三方插件冒充官方来源。

### 版本化缓存隔离

不同插件版本使用独立的缓存目录，避免升级时的状态污染。这对于有状态的 MCP 服务器和 LSP 服务器尤其重要。

## 对 agent 开发者的启示

| 模式 | 说明 |
| --- | --- |
| **容器化扩展** | 不要让用户手动拼装 skills + hooks + 配置，提供一个统一的安装入口 |
| **Schema 即文档** | 1,700 行 Zod Schema 本身就是最精确的插件格式文档 |
| **失败关闭** | 25 种错误类型意味着每种失败都有明确的处理路径，而不是统一的 `Error` |
| **名称治理** | 生态系统从第一天就需要名称保护机制 |
| **渐进式复杂度** | 让简单用例简单（一个 hook），让复杂用例可能（完整工具链） |

## 路径证据

| 路径 | 职责 |
| --- | --- |
| `src/services/plugins/` | 插件服务目录 |
| `src/services/plugins/PluginInstallationManager.ts` | 安装管理器 |
| `src/services/plugins/pluginCliCommands.ts` | 命令行入口 |
| `src/services/plugins/pluginOperations.ts` | 运行时操作 |
| `src/utils/plugins/schemas.ts` | 清单 Schema 定义（~1,681 行） |
| `src/utils/plugins/` | 插件工具函数 |

## 进一步阅读

- 工具平面 — 插件注册的工具如何融入工具体系
- 权限治理 — 插件的信任边界
- 扩展与信号 — 插件在扩展织物中的位置
- 记忆系统 — 插件与记忆系统的交互
