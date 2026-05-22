# Agent-Skills / Architecture

> 来源: claudecn.com

# Skills 架构设计

本文深入探讨 Agent Skills 的技术架构和设计理念，帮助你理解 Skills 如何高效地扩展 Claude 的能力。

## 核心设计理念

Agent Skills 采用**渐进式披露（Progressive Disclosure）**架构，这是一种现代软件工程中的"懒加载"机制，确保 Claude 只在需要时加载必要内容，避免上下文窗口浪费。

### 设计目标

- 效率优先：最小化 token 消耗
- 按需加载：只加载相关 Skills 的详细内容
- 模块化：Skills 之间相互独立，可组合使用
- 可扩展：支持无限数量的 Skills 而不影响性能
## 三层渐进加载架构

Skills 的内容分为三个层级，每层在不同时机加载：

```
+-------------------------+
|   Level 1: Metadata     |  ← Claude 启动时加载：100 tokens/skill
+-------------------------+
|   Level 2: Instructions |  ← 请求匹配时加载：<5k tokens
+-------------------------+
|   Level 3: Resources    |  ← 执行时按需加载：实际无限
+-------------------------+
```

### Level 1: 元数据（Metadata）
**加载时机**：Claude 启动时，始终加载

**内容**：

- Skill 名称（name）
- 简短描述（description）
- 可选标签和分类
**成本**：~100 tokens per Skill

**作用**：

- 帮助 Claude 发现可用的 Skills
- 快速判断是否与用户请求相关
- 轻量级设计允许安装大量 Skills
**示例**：

```yaml
---
name: "PPT Generator"
description: "Creates PowerPoint presentations based on user descriptions. Use when users ask to create slides or presentations."
tags: ["document", "presentation", "office"]
---
```

### Level 2: 指令（Instructions）
**加载时机**：当 Claude 判断 Skill 与请求相关时

**内容**：

- 详细的使用指南
- 工作流程步骤
- 最佳实践建议
- 输入输出示例
**成本**：<5k tokens（建议保持简洁）

**作用**：

- 告诉 Claude 如何正确使用这个 Skill
- 提供上下文和领域知识
- 指导 Claude 的行为和决策
**示例**：

```markdown
## 使用指南

### 工作流程
1. 分析用户的演示主题和目标受众
2. 确定幻灯片数量和结构
3. 为每张幻灯片生成标题和要点
4. 使用适当的布局和视觉元素
5. 生成最终的 PPTX 文件

### 最佳实践
- 保持每张幻灯片内容简洁
- 使用一致的视觉风格
- 为复杂概念提供图表

### 示例
输入："创建一个关于 AI 伦理的 5 页演示"
输出：包含引言、三个主要观点和结论的专业演示文稿
```

### Level 3: 资源（Resources）
**加载时机**：执行具体任务时，按需加载

**内容**：

- 可执行脚本：Python、Bash 等脚本文件
- 模板文件：PPTX 模板、文档模板等
- 参考文档：API 文档、数据库模式等
- 示例数据：测试数据、配置文件等
**成本**：实际无限（文件内容不直接进入上下文）

**访问方式**：

- Claude 通过 bash 命令读取文件：cat REFERENCE.md
- 执行脚本：python scripts/generate_ppt.py
- 只有命令的输出进入上下文窗口
**目录结构示例**：

```
my-skill/
├── SKILL.md              # Level 1+2: 元数据和指令
├── REFERENCE.md          # Level 3: API 参考文档
├── FORMS.md             # Level 3: 表单模板说明
├── scripts/
│   ├── generate.py      # Level 3: 生成脚本
│   └── validate.py      # Level 3: 验证脚本
└── templates/
    ├── basic.pptx       # Level 3: 演示模板
    └── professional.pptx # Level 3: 专业模板
```

## 虚拟机环境
Skills 在 Claude 的代码执行容器中运行，这个环境提供：

### 文件系统访问

- Skills 可以读取和写入文件
- 支持标准的文件操作命令
- 文件在会话期间持久存在
### 代码执行能力

- 运行 Python、Bash 等脚本
- 使用预安装的包和库
- 执行确定性计算任务
### 安全限制

为了安全，代码执行环境有以下限制：

- 无网络访问：无法进行外部 API 调用
- 无运行时包安装：只能使用预装包
- 资源限制：CPU 和内存使用受限
查看[代码执行工具文档](https://docs.claude.com/en/docs/agents-and-tools/tool-use/code-execution-tool)了解可用包列表。

## 上下文工程视角

从上下文工程的角度看，Skills 架构体现了几个重要原则：

### 1. 清晰的界限

Skills 通过 SKILL、FORMS、REFERENCE、scripts 和虚拟机环境构成了完整的上下文生命周期：

- 定义：SKILL.md 定义能力和用法
- 传输：通过文件系统传输知识
- 执行：在虚拟机中执行具体任务
### 2. 最小化上下文污染

- 元数据始终加载，但成本很低
- 指令只在需要时加载
- 资源通过文件系统访问，不占用上下文
### 3. 确定性与灵活性平衡

- 确定性任务：使用脚本确保一致输出
- 创意任务：使用自然语言指导允许灵活发挥
## 多 Skill 协同

Claude 可以同时使用多个 Skills 完成复杂任务：

### 自动编排

Claude 会根据任务需求：

- 识别相关的 Skills
- 确定使用顺序
- 协调 Skills 之间的输入输出
### 示例工作流

**任务**：“分析销售数据并创建季度报告演示”

**Skill 编排**：

- Excel Skill：读取和分析销售数据
- Excel Skill：生成图表和统计
- PowerPoint Skill：创建演示框架
- PowerPoint Skill：插入图表和关键发现
### 组合优势

- 每个 Skill 专注于特定任务
- Skills 之间通过文件共享数据
- Claude 负责整体协调和决策
## 与 MCP 的对比

| 特性 | Agent Skills | MCP |
| --- | --- | --- |
| **上下文成本** | 极低（渐进加载） | 高（全量加载） |
| **加载方式** | 按需加载指令和资源 | 启动时加载完整定义 |
| **设计理念** | 上下文工程 | 提示词工程 |
| **执行环境** | 虚拟机 + 文件系统 | 外部 API 调用 |
| **可靠性** | 可执行代码确保确定性 | 依赖 API 响应 |
| **可移植性** | 跨 Claude 平台（API、Code、Web） | 需要配置 MCP 服务器 |

## 性能优化策略

### 元数据优化

- 保持 name 和 description 简洁明确
- 使用动词短语描述功能
- 包含触发条件提示
### 指令优化

- 控制 SKILL.md 在 500 行以内
- 使用检查列表而非长篇描述
- 提供清晰的输入输出示例
- 假设 Claude 已知基础概念
### 资源优化

- 将详细文档放入单独的 REFERENCE.md
- 使用脚本处理复杂逻辑
- 模板文件保持合理大小
- 避免深层目录嵌套
## 架构演进

Skills 架构的设计深度借鉴了：

- Manus 的上下文工程实践
- 现代软件工程 的懒加载模式
- 微服务架构 的模块化思想
这种设计代表了从"提示词工程"到"上下文工程"的演进，为 AI 能力扩展提供了更科学的范式。

## 下一步
[快速开始创建你的第一个 Skill
](../quickstart/)[最佳实践编写高效的 Skills
](../best-practices/)[Claude Code在 Claude Code 中使用 Skills
](https://claudecn.com/docs/claude-code/)

## 参考资源

- 官方文档：Skills Overview
- Skills GitHub 仓库
- 代码执行工具文档
