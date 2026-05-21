---
title: '[Docs] 工程化重构 README.md 并建立三层文档体系（整体/功能/机制）'
labels: documentation,enhancement
priority: P1
status: draft
created: 2026-05-21
---

## 背景

当前 `README.md` 内容过于冗长（175行），同时混入了**整体介绍**、**功能说明**、**机制实现**三类内容，导致：
1. 新用户难以快速了解项目价值和上手方式
2. 功能介绍和底层机制混在一起，查找困难
3. 缺少独立的文档承载"功能怎么用"和"机制怎么实现"
4. `doc/` 目录目前仅作为 Skill 运行时输出，未承担文档职责

## 目标

建立**三层文档体系**，将内容按"整体→功能→机制"分层：

| 层面 | 文档 | 回答的问题 |
|------|------|-----------|
| **整体** | `README.md` | 这是什么项目？能做什么？怎么快速上手？ |
| **功能介绍** | `docs/features/` | 有哪些功能？怎么用？使用场景是什么？ |
| **功能机制** | `docs/mechanisms/` | 功能背后怎么实现的？设计原理是什么？怎么扩展？ |

---

## 任务一：整体层面 — README.md 工程化重构

### 当前问题
- 175行，内容过多，包含大量功能机制和架构细节
- 缺少项目徽章、CHANGELOG 链接、LICENSE 声明
- 快速开始过于简略，安装/配置步骤不清晰

### 改造方案（参考顶级开源项目最佳实践）

目标：控制在 **80~100 行**，聚焦"是什么、能做什么、怎么快速用"

```markdown
# claude-hk

[项目徽章区] — build状态、license、release版本、stars 等 (shields.io)

## 一句话简介
Claude Code 配置与文档项目，提供完整的 Hooks 管道、Skill 技能系统和自动化初始化流程。

## 功能概览（精简版，只列核心能力，详情链接到 docs/features/）
- ⚡ **Hooks 管道** — 29 个生命周期事件自动调度
- 🔧 **Skill 系统** — 可扩展的技能插件体系，热加载注入
- 🚀 **自动初始化** — 一键配置 Python、UTF-8、目录结构
- 📦 **Git 工作流** — 规范化 commit + push，中文提交信息
- 📚 **中文文档** — Claude Code CLI 完整中文翻译文档

## 快速开始
1. 克隆仓库 → 用 Claude Code 打开 → 自动触发初始化
2. 输入 `/技能名` 调用内置 Skill
3. 查看 [功能文档](docs/features/) 了解全部能力

## 安装要求
- Claude Code CLI
- Git / Bash
- Python 3.x（可选，无环境时自动下载嵌入版）

## 文档导航
| 层面 | 目录 | 内容 |
|------|------|------|
| 功能介绍 | [`docs/features/`](docs/features/) | 全部功能的使用说明 |
| 功能机制 | [`docs/mechanisms/`](docs/mechanisms/) | 设计原理与实现细节 |

## 许可证 & 鸣谢
```

### 从 README 迁出的内容

| 现有内容 | 迁出目标 |
|---------|---------|
| 项目结构详解 | `docs/mechanisms/project-structure.md` |
| 架构 / 三层设计 | `docs/mechanisms/architecture.md` |
| Hooks 管道详解 | `docs/mechanisms/hooks-pipeline.md` |
| 初始化流程 | `docs/mechanisms/initialization.md` |
| Skill 系统实现 | `docs/mechanisms/skill-system.md` |
| 内置 Skill 列表 | `docs/features/skill-reference.md` |
| Skill 使用教程 | `docs/features/skill-usage.md` |
| Git 提交规范 | `docs/features/git-workflow.md` |
| 新增 Skill 步骤 | `docs/features/how-to-add-skill.md` |
| 设计亮点 | `docs/mechanisms/design-highlights.md` |

---

## 任务二：功能介绍层面 — `docs/features/`

回答：**有哪些功能？怎么用？**

### 目录结构

```
docs/features/
├── README.md                 # 功能总览索引
├── skill-reference.md        # 内置 Skill 完整清单（5个Skill的能力、命令、输出目录）
├── skill-usage.md            # Skill 调用教程（如何触发、参数、典型场景）
├── git-workflow.md           # Git 提交规范与 004/005 Skill 使用
├── hooks-usage.md            # Hooks 事件的使用方式（开发者视角）
└── how-to-add-skill.md       # 如何新增一个 Skill（步骤教程）
```

### 各文档要点

#### `docs/features/README.md`
- 功能总览：所有功能按用户场景分类
- 快速导航到各功能详细文档

#### `docs/features/skill-reference.md`
- 5 个内置 Skill 的完整对照表
- 每个 Skill：命令、功能、输出目录、使用示例

#### `docs/features/skill-usage.md`
- 调用 Skill 的完整流程（输入 `/技能名` → 上下文注入 → 执行 → 清理）
- 各 Skill 的典型使用场景示例
- 常见问题（如 Skill 未注册、命令不识别等）

#### `docs/features/git-workflow.md`
- 004-git-push 和 005-git-commit 的使用步骤
- commit 格式规范（type + 主描述 + 子描述）
- 分组 add 的优先级规则
- 示例：一个完整的提交流程

#### `docs/features/hooks-usage.md`
- 29 个生命周期事件一览表（编号、名称、触发时机）
- 哪些事件目前有业务逻辑，哪些是纯日志
- 如何给某个事件添加自定义脚本

#### `docs/features/how-to-add-skill.md`
- 步骤 1~6 的详细教程
- 目录结构模板
- `SKILL.md` frontmatter 规范
- `on_load.sh` 编写要点
- 注册到 `registry.conf` 的格式

---

## 任务三：功能机制层面 — `docs/mechanisms/`

回答：**功能背后怎么实现的？设计原理是什么？**

### 目录结构

```
docs/mechanisms/
├── README.md                 # 机制总览索引
├── architecture.md           # 系统架构总览（三层设计、调用链）
├── project-structure.md      # 目录结构与文件职责解析
├── hooks-pipeline.md         # Hooks 管道实现机制
├── skill-system.md           # Skill 系统实现机制
├── initialization.md         # 初始化流程实现（init.sh + 01-session-start）
└── design-highlights.md      # 设计亮点与技术决策
```

### 各文档要点

#### `docs/mechanisms/README.md`
- 各机制文档的导航与阅读顺序建议
- 一张总览图/表，展示各机制之间的关系

#### `docs/mechanisms/architecture.md`
- 三层架构图：settings.json → Hooks 管道 → Skills 系统
- 数据/控制流走向
- 模块间依赖关系

#### `docs/mechanisms/project-structure.md`
- 每个一级目录的设计意图
- 关键文件职责说明
- 运行时生成的文件 vs 版本控制的文件

#### `docs/mechanisms/hooks-pipeline.md`
- `settings.json` 如何注册 29 个事件
- `base.sh` 调度器模式（调度 vs 业务分离）
- 公共基础设施：`json_get()` 三级降级、结构化日志、`hook_output()`
- 扩展机制：新增事件脚本无需改配置

#### `docs/mechanisms/skill-system.md`
- 完整调用链：用户输入 → skill-inject.sh → registry.conf → on_load.sh → 上下文注入
- 生命周期管理：`active_add` / `active_remove`
- 自动注册机制：`skill-register.sh` 扫描目录
- 并发安全：`lock.sh` mkdir 原子锁 + 僵尸锁清理
- 双路日志：全局日志 + 模块日志

#### `docs/mechanisms/initialization.md`
- 首次运行：`init.sh` 的 5 个步骤
- 每次会话：`01-session-start/base.sh` 的巡检逻辑
- 幂等性设计：标记文件、.bashrc 标记块
- 平台适配：Windows 嵌入版 Python 下载逻辑

#### `docs/mechanisms/design-highlights.md`
- 两层调度（调度器 + 业务脚本）
- 渐进降级（JSON: jq → Python → sed；Python: 嵌入版 → 系统 → 下载）
- 并发安全与幂等初始化
- 双路日志设计

---

## 补充：项目治理文档

除了三层核心文档，建议补充开源项目标配：

```
docs/
├── features/                 # 【功能介绍层面】
├── mechanisms/               # 【功能机制层面】
└── project/                  # 【项目治理】
    ├── CHANGELOG.md          # 版本更新日志（Keep a Changelog 格式）
    ├── CONTRIBUTING.md       # 贡献指南（如何提 Issue、PR、新增 Skill）
    └── SECURITY.md           # 安全策略与漏洞报告方式
```

---

## 验收标准

- [ ] README.md 精简至 100 行以内，聚焦整体介绍 + 快速开始 + 文档导航
- [ ] `docs/features/` 建立完成，覆盖所有功能的使用说明
- [ ] `docs/mechanisms/` 建立完成，覆盖所有机制的实现原理
- [ ] 现有 README 的 175 行内容**全部迁移**到对应文档，不丢失信息
- [ ] 三层文档之间有清晰的交叉引用（README → features/mechanisms，features ↔ mechanisms）
- [ ] 新增 `CHANGELOG.md`、`CONTRIBUTING.md`
- [ ] `docs/README.md` 作为文档总入口，提供三层结构的导航索引

## 相关参考

- [Keep a Changelog](https://keepachangelog.com/)
- [Shields.io](https://shields.io/)
- [GitHub Docs: 关于自述文件](https://docs.github.com/zh/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-readmes)

---

## 发布记录

- Issue #16: https://github.com/xiaoheiDTF/claude-hk/issues/16 (发布于 2026-05-21 22:41)
