# Claude-Code / Workflows / Docs-And-Codemaps

> 来源: claudecn.com

# 文档同步与 Codemaps：让文档跟着代码走

团队里最浪费时间的文档问题通常是两类：

- 文档写了，但很快过期
- 文档没写，知识只在某个人脑子里
一个更稳的思路是把“文档”当成可自动维护的产物：一部分从代码与配置中抽取（单一事实源），一部分用 codemap 的方式把架构结构“压缩成低 token 的地图”。

重要：下面讲的是“方法”，不是 Claude Code 的内置功能。你需要把它落地为团队的自定义命令（`.claude/commands/`）或专用 agent（`.claude/agents/`）。

## 1) 文档同步：把事实源钉在 package.json 与 .env.example

社区仓库的 `/update-docs` 命令强调一个原则：

Single source of truth: `package.json` and `.env.example`

你可以把文档分成两类：

- 事实型文档（强烈建议自动生成/半自动同步） 脚本与开发命令（来自 package.json scripts）
- 环境变量说明（来自 .env.example）
- 贡献指南/开发流程（来自实际仓库约定）
- 解释型文档（需要人工写，但可以由 Claude 辅助）为什么这样设计（ADR/设计说明）
- 关键模块的边界与数据流
- 常见故障与排查路径
一个可执行的 `/update-docs` 工作流可以包括：

- 生成脚本参考表（ scripts → 表格）
- 提取环境变量清单（.env.example → 说明与格式）
- 生成 docs/CONTRIB.md（开发流程、测试流程、常用命令）
- 生成 docs/RUNBOOK.md（部署、监控、常见故障、回滚）
- 列出“90 天未更新的文档”供人工复核（不要自动删）
## 2) Codemaps：用“结构地图”降低上下文成本

当仓库变大后，最难的是让 Claude（以及新同事）快速理解“哪里是入口、模块怎么串”。社区仓库的 `/update-codemaps` 思路是：

- 扫描 imports/exports/dependencies
- 生成 token-lean codemaps（只保留结构与入口，不写实现细节）
- 按模块输出多张地图（architecture/backend/frontend/data）
- 计算与上一版的差异百分比
- 变化超过阈值（示例里是 30%）时先请求人工确认再覆盖更新
这种“地图”的价值在于：你可以把它作为 Claude Code 会话的稳定上下文（比直接塞一堆源码更省 token）。

## 3) 用专用 agent 把它跑成团队的“日常维护”

社区仓库提供了 `doc-updater` agent 的角色定义：它的职责是“更新 codemaps 与文档，并验证文档与仓库现状一致”。你可以把它作为固定流程：

- 每次大版本合并前跑一次 codemaps + docs sync
- 每次发布前跑一次 runbook 与脚本表更新
- 每次架构变化后补 ADR，并在 codemap 顶部刷新时间戳
## 4) 落地建议：不要制造“随机文档碎片”

文档最怕碎：到处都是小 `.md`，没人知道哪个才是最新。社区仓库甚至用 Hook 去阻止“随手写一堆 md”。团队落地建议：

- 入口固定：README / docs/INDEX / codemaps/INDEX
- 解释型文档写在少量固定位置（例如 docs/ADR/、docs/GUIDES/）
- 事实型文档尽量自动同步（ scripts/env/codemaps）
## 下一步

- 想把团队配置工程化：从 团队 Starter Kit 开始
- 想把文档维护变成一条闭环：结合 团队质量门禁
## 参考

无
