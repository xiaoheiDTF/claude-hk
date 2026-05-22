# Source-Analysis / Commands

> 来源: claudecn.com

# 命令表面

用户怎样接触 Claude Code 的能力？97 个命令入口构成了系统的操作表面，它们是工具能力向用户暴露的门面——理解命令和工具的区别，是理解这个系统的关键。

可视化对应：[Commands 页面](https://code.claudecn.com/commands/)

**这里的 97 是路径分桶后的入口数。** 当前口径来自 `../src/commands/` 第一段路径归一化，并排除了 `createMovedToPluginCommand`、`init`、`init-verifiers` 三个内部 wiring 入口。因此它表示“有效命令入口面”，而不是所有用户在任意上下文下都能看到的 slash 命令数。

## 命令与工具的关系

```
能力面 → 编排层 → 用户面 → 直接绑定 → 多工具编排 → 服务链 → 独立路径 → 用户 /command → 命令路由 → src/commands/ → 单工具 → 模型选择多工具 → Service 调用 → 内部逻辑 → 工具系统 → src/tools/
```

## 命令 vs 工具

| 维度 | 命令 (Commands) | 工具 (Tools) |
| --- | --- | --- |
| 触发方式 | 用户输入 `/command` | 模型决定调用 |
| 数量 | 97 个 | 43 个 |
| 本质 | 用户入口面 | 系统能力面 |
| 权限 | 部分需要认证 | 统一走权限层 |
| 位置 | `src/commands/` | `src/tools/` |
**关键认知**：看到一个 `/commit` 命令，并不意味着有一个 `CommitTool`。命令更像是把多个工具能力编排在一起的产品入口。同样，很多工具（如 `FileEditTool`）从来不需要用户通过命令显式调用，而是模型在主循环中自动选择。

## 命令全景

| 指标 | 数值 |
| --- | --- |
| 命令总数 | 97 |
| 分组数 | 7 |
| 总文件数 | 189（含分组内部逻辑） |

## 七大分组

### 1. Setup & Config（22 命令，73 files）

安装、认证、模型选择、插件管理、权限设置和环境配置。

| 命令 | 功能 |
| --- | --- |
| `/login` `/logout` | 认证管理 |
| `/config` | 读写配置项 |
| `/model` | 切换模型（Sonnet / Opus） |
| `/mcp` | MCP 服务器管理 |
| `/permissions` | 权限规则管理 |
| `/plugin` `/reload-plugins` | 插件安装与重载 |
| `/theme` | 终端主题切换 |
| `/upgrade` | 版本升级 |
| `/install` | 安装到系统 |
| `/install-github-app` | 安装 GitHub App |
| `/install-slack-app` | 安装 Slack App |
| `/terminalSetup` | 终端配置 |
| `/add-dir` | 添加工作目录 |
| `/remote-env` `/remote-setup` | 远程环境配置 |
| `/sandbox-toggle` | 沙箱开关 |
| `/privacy-settings` | 隐私设置 |
| `/rate-limit-options` | 速率限制选项 |
| `/output-style` | 输出风格 |
| `/oauth-refresh` | OAuth 刷新 |

### 2. Daily Workflow（23 命令，44 files）

面向日常编码、恢复会话、计划、总结和任务推进。

| 命令 | 功能 |
| --- | --- |
| `/plan` | 进入计划模式（只分析不执行） |
| `/compact` | 手动触发上下文压缩 |
| `/memory` | 记忆管理 |
| `/resume` | 恢复历史会话 |
| `/session` | 会话管理 |
| `/summary` | 生成会话总结 |
| `/status` | 查看当前状态 |
| `/tasks` | 任务列表 |
| `/skills` | 技能管理 |
| `/files` | 查看相关文件 |
| `/context` | 上下文信息 |
| `/usage` | 用量查看 |
| `/help` | 帮助信息 |
| `/copy` | 复制到剪贴板 |
| `/clear` | 清除上下文 |
| `/exit` | 退出 |
| `/brief` | 简洁模式 |
| `/hooks` | 钩子管理 |
| `/onboarding` | 新手引导 |
| `/rewind` | 回退操作 |
| `/share` | 分享会话 |
| `/version` | 版本信息 |
| `/voice` | 语音输入 |

### 3. Review & Git（10 命令，18 files）

围绕分支、提交、PR、diff、安全审查和问题管理。

| 命令 | 功能 |
| --- | --- |
| `/review` | 代码审查 |
| `/commit` | Git 提交 |
| `/commit-push-pr` | 提交+推送+创建 PR 一条龙 |
| `/diff` | 查看差异 |
| `/branch` | 分支操作 |
| `/issue` | Issue 管理 |
| `/pr_comments` | PR 评论处理 |
| `/security-review` | 安全审查 |
| `/autofix-pr` | 自动修复 PR |
| `/rename` | 重命名 |

### 4. Debugging & Diagnostics（18 命令，26 files）

问题诊断、指标观察、缓存调试与内部追踪。

| 命令 | 功能 | 备注 |
| --- | --- | --- |
| `/doctor` | 环境诊断 | 公开 |
| `/cost` | 费用查看 | 公开 |
| `/stats` | 使用统计 | 公开 |
| `/env` | 环境变量 | 公开 |
| `/export` | 导出数据 | 公开 |
| `/feedback` | 反馈 | 公开 |
| `/release-notes` | 发布说明 | 公开 |
| `/ctx_viz` | 上下文可视化 | feature-flagged |
| `/debug-tool-call` | 工具调用调试 | feature-flagged |
| `/ant-trace` | 内部追踪 | feature-flagged |
| `/heapdump` | 堆内存快照 | feature-flagged |
| `/break-cache` | 缓存打断 | feature-flagged |
| `/mock-limits` | 模拟限制 | feature-flagged |
| `/reset-limits` | 重置限制 | feature-flagged |
| `/bughunter` | Bug 猎手 | feature-flagged |
| `/passes` | 执行 Pass | feature-flagged |
| `/perf-issue` | 性能问题 | feature-flagged |
| `/insights` | 洞察面板 | feature-flagged |

### 5. Bridge & Remote（7 命令，12 files）

IDE 桥接、跨设备连接与远程控制。

| 命令 | 功能 |
| --- | --- |
| `/ide` | IDE 集成（VS Code / JetBrains） |
| `/bridge` | Bridge 桥接控制 |
| `/bridge-kick` | 断开 Bridge 连接 |
| `/desktop` | 桌面应用连接 |
| `/mobile` | 移动端连接 |
| `/chrome` | Chrome 扩展连接 |
| `/teleport` | Teleport 传送 |

### 6. Experimental Surface（13 命令，24 files）

更像产品表面的探索入口，观察产品方向。

| 命令 | 功能 | 状态 |
| --- | --- | --- |
| `/ultraplan` | 长时间规划（Opus 级模型） | feature-flagged |
| `/advisor` | AI 顾问模式 | feature-flagged |
| `/thinkback` `/thinkback-play` | 思维回溯与回放 | feature-flagged |
| `/fast` | 快速模式（切换到更快模型） | 公开 |
| `/good-claude` | 正面反馈 | 公开 |
| `/stickers` | 贴纸系统 | feature-flagged |
| `/statusline` | 状态行配置 | feature-flagged |
| `/btw` | 顺便一说 | feature-flagged |
| `/color` | 颜色配置 | 公开 |
| `/effort` | 推理 effort 调节 | feature-flagged |
| `/extra-usage` | 额外用量追踪 | feature-flagged |
| `/tag` | 会话标签 | feature-flagged |

### 7. Agents & Extensions（4 命令，7 files）

子代理管理、Vim 模式和内部迁移命令。

| 命令 | 功能 |
| --- | --- |
| `/agents` | 子代理管理 |
| `/vim` | Vim 模式切换 |
| `/keybindings` | 快捷键配置 |
| `/backfill-sessions` | 会话数据回填（内部迁移） |

## 命令到工具映射

命令和工具不是一一对应的。一条命令可能编排多个工具，一个工具也可能被多条命令复用。以下是主要命令与底层工具的映射关系：

| 命令 | 底层工具 / 执行路径 | 映射类型 |
| --- | --- | --- |
| `/plan` | `EnterPlanModeTool` → `ExitPlanModeTool` | 直接绑定 |
| `/compact` | `src/services/compact/compact.ts` | 直接调用服务 |
| `/memory` | `memdir/*`、`SessionMemory/*` | 服务编排 |
| `/resume` | `SessionMemory` + `setCostStateForRestore()` | 多服务协作 |
| `/commit` | `BashTool`（git 命令） + 模型编排 | 工具 + 模型 |
| `/review` | `FileReadTool` + `GrepTool` + `AgentTool`（Verify） | 多工具编排 |
| `/commit-push-pr` | `BashTool` × 3（commit → push → gh pr create） | 工具链 |
| `/diff` | `BashTool`（git diff） | 单工具 |
| `/voice` | `voice.ts` + `voiceStreamSTT.ts` → 模型输入 | 服务链 |
| `/mcp` | `MCPTool` 注册/卸载 + 配置写入 | 配置 + 工具 |
| `/plugin` | `PluginInstallationManager.ts` | 服务调用 |
| `/agents` | `AgentTool` + `SendMessageTool` + `TaskStopTool` | 多工具 |
| `/tasks` | `TaskCreateTool` / `TaskGetTool` / `TaskListTool` | 工具族 |
| `/doctor` | 环境检查脚本（不经过工具系统） | 独立路径 |
| `/cost` | `cost-tracker.ts` 读取 | 状态读取 |
| `/vim` | `src/vim/` 模式切换 | UI 层 |

**读这张表时要注意**：

- “直接绑定"表示命令和工具一一对应
- “多工具编排"表示命令背后是模型根据上下文自动选择多个工具
- “服务链"表示命令触发的是一组服务调用，不经过标准工具系统
- “独立路径"表示命令完全绕过工具和模型，直接执行内部逻辑
这也解释了为什么命令数（97）远多于工具数（43）——命令是面向用户的产品入口，工具是面向模型的能力面；二者之间的映射不是线性的。

## Feature-Flagged 命令

在 97 个命令中，有相当一部分受 feature flag 控制，只有特定条件下才会出现。这些命令的存在是系统演化方向的重要信号。

主要的 feature-flagged 入口集中在：

- 诊断工具：ctx_viz、debug-tool-call、ant-trace、heapdump — 内部调试能力
- 实验表面：ultraplan、advisor、thinkback — 产品方向探索
- 远程连接：bridge、desktop、mobile — 跨界面扩展
## 建议阅读顺序

- 工具平面 — 理解命令背后的能力
- 运行时流程 — 看命令如何进入主循环
- 扩展与信号 — 看实验命令和信号的关系
