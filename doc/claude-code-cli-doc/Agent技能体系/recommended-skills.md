# Agent-Skills / Recommended-Skills

> 来源: claudecn.com

# Claude Skills 实用指南

Skills 是 Claude 动态加载的指令包——一个文件夹里放着 `SKILL.md`、脚本和参考资料，Claude 在识别到相关任务时自动读取并遵循。这套机制让同一个 Claude 在不同场景下表现出截然不同的专业水准。

截至 2026-04-07 的同步快照，Anthropic 在 [官方仓库](https://github.com/anthropics/skills) 中维护 17 个一级 Skills 目录，按用途可归为四类能力外加一个元技能。

## 创意与设计

### frontend-design

解决 AI 生成前端"千篇一律"的问题。它要求 Claude 在写代码前先确定美学方向——极简主义、复古未来风、Brutalist、有机自然风、奢华精致——然后在排版、配色、动效、空间构图上全部围绕这个方向执行。明确禁用 Inter、Roboto、Arial 等高频字体，强制使用有辨识度的字体搭配。支持 HTML/CSS/JS、React、Vue 等主流框架输出。

**适用场景：** 构建网页、落地页、仪表盘、React 组件，或任何需要提升 UI 美感的前端界面。

```bash
/install anthropics/skills/frontend-design
```

### canvas-design
用代码在 PDF 或 PNG 上生成视觉作品。工作流分两步：先输出一份"设计哲学"文档（.md），定义美学运动的核心理念；再按照这份哲学在画布上表达，生成原创视觉设计。严格避免复制已有艺术家作品。

**适用场景：** 海报设计、封面制作、数据可视化图形、品牌视觉素材。

```bash
/install anthropics/skills/canvas-design
```

### algorithmic-art
通过 p5.js 生成算法艺术：几何图案、流场（Flow Field）、粒子系统和分形。输出三类文件：算法哲学文档（.md）、交互式查看器（.html）和生成算法（.js）。使用种子随机数确保可复现，支持交互式参数调整。

**适用场景：** 生成式艺术创作、程序化视觉素材、技术团队品牌元素。

```bash
/install anthropics/skills/algorithmic-art
```

### theme-factory
提供 10 套预设的专业主题系统，每套包含精心配色和字体搭配。可以应用到幻灯片、文档、报告、HTML 落地页等任何制品上，也支持根据输入关键词即时生成新主题。

**适用场景：** 快速为 UI 组件、演示文稿、营销页面统一视觉风格。

```bash
/install anthropics/skills/theme-factory
```

## 文档处理
官方仓库中的四个文档 Skills 驱动着 Claude.ai 的文档创建功能，属于 source-available 许可。不装时 Claude 每次临时写脚本处理格式，结果不稳定；装上后相当于给了一套经过生产验证的标准流程。

### docx

创建、读取、编辑 Word 文档。底层将 .docx 视为 ZIP/XML 结构操作，支持目录生成、页眉页脚、图片插入、批注追踪、查找替换等高级功能。生产级输出要求零格式错误。

**适用场景：** 技术报告、备忘录、合同模板、项目文档自动生成。

```bash
/install anthropics/skills/docx
```

### xlsx
处理 Excel 和 CSV/TSV 文件的完整流水线。要求所有公式零错误（无 #REF!、#DIV/0! 等），使用专业字体，支持图表生成、条件格式、数据透视。能处理格式混乱的原始数据文件。

**适用场景：** 数据清洗、财务建模、报表生成、跨格式转换。

```bash
/install anthropics/skills/xlsx
```

### pdf
PDF 全生命周期处理：文本提取、合并拆分、页面旋转、水印添加、表单填写、加密解密、图片提取、扫描件 OCR。基于 Python 工具链（pypdf、pymupdf 等）实现。

**适用场景：** 批量 PDF 处理、合同签署流程、扫描文件数字化。

```bash
/install anthropics/skills/pdf
```

### pptx
读取、创建和编辑 PowerPoint 演示文稿。支持从模板创建、从零创建两种模式，处理母版布局、演讲者备注、注释等元素。可搭配 `markitdown` 提取 .pptx 中的文本内容。

**适用场景：** 项目汇报、融资路演、培训课件、会议演示自动生成。

```bash
/install anthropics/skills/pptx
```

这四个 Skill 可以叠加使用：从 PDF 提取数据 → 在 Excel 中分析 → 生成 Word 报告 → 做成 PPT 演示。

## 开发与技术

### mcp-builder

指导 Claude 构建高质量的 MCP Server（Model Context Protocol）。覆盖从 API 深度研究、Schema 设计、工具实现到评估测试的完整四阶段流程。推荐 TypeScript + Streamable HTTP 技术栈，内附 Python 和 TypeScript 两套完整参考实现。强调工具命名可发现性、上下文管理、可执行的错误消息，以及工具注解（readOnlyHint、destructiveHint 等）。

**适用场景：** 为 Claude 接入外部 API/服务，构建自定义 MCP Server，扩展 Claude 的工具生态。

```bash
/install anthropics/skills/mcp-builder
```

### claude-api
使用 Claude API 或 Agent SDK 构建 LLM 应用。自动检测项目语言（支持 Python / TypeScript / Java / Go / Ruby / C# / PHP / cURL），加载对应的代码示例和 SDK 用法。内置模型选择决策树、Adaptive Thinking 配置（Opus 4.6 / Sonnet 4.6 推荐方式）、Prompt Caching 前缀稳定性设计、Compaction 长会话方案，以及 Tool Runner 自动循环。

**适用场景：** 构建聊天机器人、多步工作流、自定义 Agent、批量处理管线。

```bash
/install anthropics/skills/claude-api
```

### webapp-testing
用 Playwright 测试本地 Web 应用。提供 `with_server.py` 脚本管理服务器生命周期，支持多服务器并行启动。核心模式是"先侦察再操作"——等待 `networkidle` 后截图识别选择器，再执行交互。包含常见模式示例：元素发现、静态 HTML 自动化、控制台日志捕获。

**适用场景：** 前端功能验证、UI 行为调试、浏览器截图对比、自动化回归测试。

```bash
/install anthropics/skills/webapp-testing
```

### web-artifacts-builder
在 Claude.ai 中构建复杂的多组件 Web Artifact。技术栈为 React 18 + TypeScript + Vite + Tailwind CSS + shadcn/ui，通过 `init-artifact.sh` 初始化项目，`bundle-artifact.sh` 打包为单个 HTML 文件。用于需要状态管理、路由或 shadcn/ui 组件的复杂 Artifact。

**适用场景：** 交互式原型、数据仪表盘、内部工具的快速 MVP。

```bash
/install anthropics/skills/web-artifacts-builder
```

## 企业与协作

### brand-guidelines
加载 Anthropic 官方品牌色彩和字体规范，将其应用到任何制品上——幻灯片、文档、报告、HTML 页面。包含完整的色彩系统和字体层级定义。也可作为模板，指导你为自己的品牌创建类似的规范 Skill。

**适用场景：** 品牌视觉一致性维护、设计规范文档生成。

```bash
/install anthropics/skills/brand-guidelines
```

### internal-comms
撰写各类内部沟通文档：3P 更新（进展/计划/问题）、全员邮件、项目周报、变更通知、事故报告、FAQ。根据受众层级自动调整语气和信息密度。提供结构化模板和最佳实践参考。

**适用场景：** 管理层汇报、跨部门通知、项目状态更新、事故复盘。

```bash
/install anthropics/skills/internal-comms
```

### doc-coauthoring
结构化的文档协同撰写工作流。分三个阶段推进：**上下文收集**（提问 + 信息倾倒） → **迭代打磨**（逐章节头脑风暴 → 筛选 → 起草 → 精修）→ **读者测试**（用无上下文的 Claude 验证文档是否自洽）。维护文档结构、交叉引用和术语一致性。

**适用场景：** 技术白皮书、产品 PRD、设计决策文档、RFC 等需要严谨结构的长文档。

```bash
/install anthropics/skills/doc-coauthoring
```

### slack-gif-creator
创建适合 Slack 聊天的动画 GIF。针对 Slack 的尺寸限制优化（Emoji GIF 128×128、消息 GIF 480×480），提供验证工具和动画概念模板。

**适用场景：** 团队自定义表情、项目里程碑庆祝动画、趣味沟通素材。

```bash
/install anthropics/skills/slack-gif-creator
```

## 元技能

### skill-creator
构建属于你自己的 Skill。完整工作流包括：意图捕获 → 用户访谈 → SKILL.md 草稿编写 → 测试用例生成 → 并行运行对比（有 Skill vs 无 Skill）→ 定量评估与基准测试 → 人工审核（通过内置 HTML 查看器）→ 迭代优化 → 触发描述优化（通过 `run_loop.py` 自动化）。支持渐进式披露架构：SKILL.md < 500 行，大型参考资料按需加载。

**适用场景：** 封装团队代码规范、部署流程、运维检查清单、个人工作流——这些通用 Skills 覆盖不到的领域。

```bash
/install anthropics/skills/skill-creator
```

## 安装方式

### Claude Code（推荐）
逐个安装——在 Claude Code 中输入：

```bash
/install anthropics/skills/frontend-design
/install anthropics/skills/mcp-builder
/install anthropics/skills/skill-creator
/install anthropics/skills/docx
# ... 其他 Skill 同理
```

### Claude.ai
付费用户可直接使用所有官方 Skills，无需手动安装。自定义 Skills 通过上传 `.skill` 文件启用。

### Claude API

通过 [Skills API](https://docs.claude.com/en/api/skills-guide) 在 API 调用中附加 Skills。

## 加载原理

Skills 采用三级渐进式披露，不会因为安装过多而显著增加日常消耗：

| 级别 | 加载时机 | Token 量级 |
| --- | --- | --- |
| 元数据 | 每次会话启动 | ~100 tokens / Skill |
| SKILL.md 正文 | 任务匹配时 | < 5K tokens |
| 附带资源 | 按需读取 | 无上限 |

触发机制基于 `description` 字段的语义匹配——Claude 判断当前任务是否需要某个 Skill 的专业能力，只有匹配时才加载完整指令。

## 下一步
[快速开始创建你的第一个自定义 Skill
](../quickstart/)[官方示例集17 个 Skills 完整源码
](../examples/)[最佳实践编写高质量 Skills 的设计原则
](../best-practices/)
