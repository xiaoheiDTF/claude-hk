# 文档中心

本项目的文档按"整体→功能→机制"三层组织。

## 文档结构

| 层面 | 目录 | 回答的问题 |
|------|------|-----------|
| 架构设计 | [`redeme/`](redeme/) | 系统架构、流程图、设计原理 |
| 功能介绍 | [`features/`](features/) | 有哪些功能？怎么用？使用场景是什么？（待补充） |
| 功能机制 | [`mechanisms/`](mechanisms/) | 功能背后怎么实现的？设计原理是什么？（待补充） |
| 项目治理 | [`project/`](project/) | 版本日志、贡献指南（待补充） |
| 中文翻译 | [`claude-code-cli-doc/`](claude-code-cli-doc/) | Claude Code CLI 完整中文文档（130+ 篇） |
| 技能输出 | [`otherDoc/`](otherDoc/) | 002-1 技能的归档文档（按日期） |
| 测试脚本 | [`testcode/`](testcode/) | 002-2 技能的 Python 测试脚本 |
| Issue 草稿 | [`issues/`](issues/) | Issue 模板和本地草稿 |

## 架构设计

> 面向开发者：完整的系统架构参考和流程图。

| 文档 | 内容 |
|------|------|
| [Hooks + Skills 架构](redeme/hooks-skills-architecture.md) | 29 个 Hook 事件、24 个 Skill、共享模块、003 Issue 工作流 |
| [Hooks + Skills 流程图集](redeme/hooks-skills-diagrams.md) | 总览图、初始化时序、Skill 生命周期、并发安全模型 |
| [claude_tap_plus 架构](redeme/claude-tap-plus-architecture.md) | Go 后端：代理模式、15 个 API 端点、6 张表、Profile 配置 |
| [claude_tap_plus 流程图集](redeme/claude-tap-plus-diagrams.md) | 数据流、配置优先级、SSE 重组、ER 图 |
| [Dashboard 线框图](redeme/dashboard-wireframe.md) | 仪表盘功能分析、BDD 场景、后端 BDD |
