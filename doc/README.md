# 文档中心

本项目的文档按"整体→功能→机制"三层组织。

## 文档结构

| 层面 | 目录 | 回答的问题 |
|------|------|-----------|
| 功能介绍 | [`features/`](features/) | 有哪些功能？怎么用？使用场景是什么？ |
| 功能机制 | [`mechanisms/`](mechanisms/) | 功能背后怎么实现的？设计原理是什么？ |
| 项目治理 | [`project/`](project/) | 版本日志、贡献指南 |

## 功能介绍

> 面向使用者：了解项目能做什么、怎么用。

| 文档 | 内容 |
|------|------|
| [Skill 完整清单](features/skill-reference.md) | 13 个内置 Skill 的命令、能力、输出目录 |
| [Skill 使用教程](features/skill-usage.md) | 调用流程、参数、典型场景 |
| [Git 工作流](features/git-workflow.md) | 提交规范、004/005 Skill 使用 |
| [Hooks 使用方式](features/hooks-usage.md) | 29 个事件一览、如何扩展 |
| [如何新增 Skill](features/how-to-add-skill.md) | 6 步教程、目录模板、规范 |

## 功能机制

> 面向开发者：了解实现原理和设计决策。

| 文档 | 内容 |
|------|------|
| [系统架构](mechanisms/architecture.md) | 三层设计、调用链、模块依赖 |
| [目录结构](mechanisms/project-structure.md) | 每个目录/文件的设计意图和职责 |
| [Hooks 管道](mechanisms/hooks-pipeline.md) | 事件注册、调度器模式、JSON 解析降级 |
| [Skill 系统](mechanisms/skill-system.md) | 上下文注入、生命周期、工具边界、并发安全 |
| [初始化流程](mechanisms/initialization.md) | init.sh 5 步骤、幂等性、平台适配 |
| [设计亮点](mechanisms/design-highlights.md) | 两层调度、渐进降级、并发安全、双路日志 |

## 项目治理

| 文档 | 内容 |
|------|------|
| [CHANGELOG](project/CHANGELOG.md) | 版本更新日志 |
| [贡献指南](project/CONTRIBUTING.md) | 如何提 Issue、PR、新增 Skill |
