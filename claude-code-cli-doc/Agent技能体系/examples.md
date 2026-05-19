# Agent-Skills / Examples

> 来源: claudecn.com

# Skills 示例集

本文介绍 Anthropic 官方 Skills 仓库中的[ 开源](#)示例，涵盖文档处理、创意设计、技术开发等多个领域。当前同步快照包含 17 个一级 Skills 目录，均可作为创建自定义 Skills 的参考。

## Skills 仓库结构

Anthropic Skills 仓库采用标准化的组织结构：

```
anthropics/skills/
├── skills/           # 所有 Skills 实现
│   ├── doc-coauthoring/
│   ├── docx/
│   ├── frontend-design/
│   └── ...
├── spec/             # Agent Skills 规范
│   └── agent-skills-spec.md
└── template/         # Skill 模板
    └── SKILL.md
```

**仓库地址**: [github.com/anthropics/skills](https://github.com/anthropics/skills)

## 文档处理 Skills

### doc-coauthoring - 文档协作创作

**新增** (2025-01)：结构化的文档协作创作工作流。

**功能特性**：

- 三阶段工作流：上下文收集、优化结构化、读者测试
- 支持多种文档类型：技术规范、决策文档、项目提案、RFC
- 集成团队工具：Slack、Teams、Google Drive
- 图片 alt-text 自动生成
**工作流程**：

- Context Gathering（上下文收集）收集项目背景和需求
- 提出澄清问题
- 整合团队讨论内容
- Refinement & Structure（优化与结构化）迭代构建每个章节
- 头脑风暴和编辑
- 结构化内容组织
- Reader Testing（读者测试）使用全新 Claude 实例测试
- 发现文档盲点
- 确保独立可读性
**触发条件**：

```
"write a doc", "draft a proposal", "create a spec"
"PRD", "design doc", "decision doc", "RFC"
```

**代码规模**: 375 行指令

**许可证**: Apache 2.0（开源）

### docx - Word 文档处理

**功能特性**：

- DOCX 文件创建和编辑
- 完整的 OOXML 标准支持（ISO-IEC29500-4）
- 样式、表格、图片处理
- 评论系统集成
**技术实现**：

```
skills/docx/
├── SKILL.md              # 主指令
├── docx-js.md            # JavaScript 实现指南
├── ooxml.md              # OOXML 标准文档
├── ooxml/
│   ├── schemas/          # 完整 OOXML 架构（ISO-IEC29500-4）
│   └── scripts/          # 验证脚本
└── scripts/
    ├── document.py       # 文档处理
    ├── templates/        # 评论模板
    └── utilities.py      # 工具函数
```

**许可证**: 源码可用（非[ 开源](#)），供开发者参考

### pdf - PDF 文档处理

**功能特性**：

- PDF 表单填写和字段提取
- 边界框检查和验证
- PDF 转图片
- 表单注释处理
**核心脚本**：

```python
scripts/
├── check_bounding_boxes.py          # 边界框验证
├── extract_form_field_info.py       # 表单字段提取
├── fill_fillable_fields.py          # 填充表单字段
├── convert_pdf_to_images.py         # PDF 转图片
└── fill_pdf_form_with_annotations.py # 注释填写
```

**许可证**: 源码可用（非开源）

### pptx - PowerPoint 文档处理

**功能特性**：

- PPTX 文件创建和编辑
- HTML 到 PowerPoint 转换
- 幻灯片操作和样式管理
- 缩略图生成
**核心工具**：

```python
scripts/
├── html2pptx.js      # HTML 转 PowerPoint
├── inventory.py      # 幻灯片清单
├── rearrange.py      # 调整幻灯片顺序
├── replace.py        # 内容替换
└── thumbnail.py      # 缩略图生成
```

**许可证**: 源码可用（非开源）

### xlsx - Excel 文档处理

**功能特性**：

- XLSX 文件创建和编辑
- 公式重新计算
- 数据处理和验证
**技术实现**：

- 公式引擎（recalc.py）
- 数据验证
- 表格操作
**许可证**: 源码可用（非开源）

## 创意与设计 Skills

### frontend-design - 前端设计

**新增** (2025-01)：创建独特的生产级前端界面，避免通用 AI 美学。

**设计理念**：

- 大胆美学方向: 选择明确的设计概念并精确执行
- 生产级代码: 功能完整、视觉冲击、细节精致
- 创意优先: 避免通用 AI 美学和可预测布局
**设计焦点**：

**排版（Typography）**：

- 独特字体选择（避免 Inter、Roboto、Arial）
- 配对展示字体和正文字体
- 字体传达设计理念
**颜色与主题**：

- CSS 变量保持一致性
- 主导色 + 尖锐强调色
- 承诺一致的美学
**动效（Motion）**：

- 高影响力时刻的精心编排
- 页面加载的交错显示（animation-delay）
- 滚动触发和悬停状态
- CSS-only 或 Motion 库
**空间构图**：

- 不对称布局
- 重叠和对角流动
- 打破网格的元素
- 大量留白或控制密度
**背景与视觉细节**：

- 渐变网格、噪点纹理
- 几何图案、分层透明度
- 戏剧性阴影、装饰边框
- 自定义光标、颗粒覆盖
**触发条件**：

```
"build web components", "create landing page"
"design dashboard", "create React component"
"style interface", "beautify web UI"
```

**输出格式**: HTML/CSS/JS、React/Vue 组件

**代码规模**: 42 行核心指令

**许可证**: Apache 2.0（[ 开源](#)）

### canvas-design - Canvas 设计

**功能特性**：

- Canvas 图形设计
- 81 个字体文件支持
- 54 种独特字体系列
**字体库**：

- Arsenal SC, Big Shoulders, Bricolage Grotesque
- Crimson Pro, IBM Plex (Mono/Serif)
- Instrument Sans/Serif, JetBrains Mono
- National Park, Work Sans, Young Serif
- 等等…
**目录结构**：

```
skills/canvas-design/
├── SKILL.md
└── canvas-fonts/
    ├── ArsenalSC-Regular.ttf
    ├── IBMPlexMono-Bold.ttf
    ├── InstrumentSans-Regular.ttf
    └── ... (81 个字体文件)
```

**许可证**: Apache 2.0（开源）

### algorithmic-art - 算法艺术

**功能特性**：

- 生成式艺术创作
- JavaScript 算法艺术模板
- 交互式艺术查看器
**包含文件**：

```
skills/algorithmic-art/
├── SKILL.md
├── templates/
│   ├── generator_template.js  # 生成器模板
│   └── viewer.html            # HTML 查看器
```

**许可证**: Apache 2.0（开源）

### theme-factory - 主题工厂

**功能特性**：

- 10 种预设设计主题
- 品牌主题创建
- 一致性设计系统
**预设主题**：

- Arctic Frost: 北极霜 - 清冷简洁
- Botanical Garden: 植物园 - 自然有机
- Desert Rose: 沙漠玫瑰 - 温暖柔和
- Forest Canopy: 森林树冠 - 深邃宁静
- Golden Hour: 黄金时刻 - 温暖明亮
- Midnight Galaxy: 午夜银河 - 深邃神秘
- Modern Minimalist: 现代极简 - 精致克制
- Ocean Depths: 海洋深处 - 深邃流动
- Sunset Boulevard: 日落大道 - 渐变活力
- Tech Innovation: 科技创新 - 未来感
**目录结构**：

```
skills/theme-factory/
├── SKILL.md
├── theme-showcase.pdf
└── themes/
    ├── arctic-frost.md
    ├── botanical-garden.md
    └── ... (10 个主题文件)
```

**许可证**: Apache 2.0（[ 开源](#)）

### slack-gif-creator - Slack GIF 创建器

**功能特性**：

- 创建 Slack 动图
- 动画效果库
- GIF 构建器
**2025-01 重构**：

- 代码精简约 85%（从 7564 行减至约 1000 行）
- 删除 13 个冗余动画模板
- 重写核心模块提高可维护性
**核心模块**：

```python
skills/slack-gif-creator/
├── SKILL.md (254 行)
├── core/
│   ├── easing.py          # 缓动函数
│   ├── frame_composer.py  # 帧合成器（176 行，重写）
│   ├── gif_builder.py     # GIF 构建器（103 行变更）
│   └── validators.py      # 验证器（136 行，重写）
└── requirements.txt
```

**许可证**: Apache 2.0（开源）

### brand-guidelines - 品牌指南

**功能特性**：

- 品牌一致性指导
- 设计规范管理
- 品牌资产组织
**许可证**: Apache 2.0（开源）

## 技术开发 Skills

### mcp-builder - MCP 服务器构建器

**功能特性**：

- 创建 Model Context Protocol 服务器
- Node.js 和 Python 实现
- 服务器验证和测试
**2025-01 更新**：

- 最佳实践从 915 行精简到 249 行（-73%）
- Node.js 服务器参考重写 85%
- Python 服务器参考保留 94%
- 更聚焦实用指导
**核心参考**：

```
skills/mcp-builder/
├── SKILL.md (236 行)
└── reference/
    ├── mcp_best_practices.md (249 行，精简版)
    ├── node_mcp_server.md (重写 85%)
    ├── python_mcp_server.md (94% 保留)
    └── evaluation.md
```

**验证工具**：

```
scripts/
├── connections.py         # 连接测试
├── evaluation.py          # 评估脚本
├── example_evaluation.xml # 评估示例
└── requirements.txt
```

**许可证**: Apache 2.0（开源）

### skill-creator - Skill 创建器

**功能特性**：

- 创建自定义 Agent Skills
- Skill 验证和打包
- 快速验证工具
**2025-01 更新**：

- 新增输出模式参考（output-patterns.md, 82 行）
- 新增工作流指导（workflows.md, 28 行）
- 增强打包和验证脚本
**目录结构**：

```
skills/skill-creator/
├── SKILL.md (173 行变更)
├── references/
│   ├── output-patterns.md # 新增：输出模式参考
│   └── workflows.md       # 新增：工作流指导
└── scripts/
    ├── init_skill.py      # 初始化 Skill
    ├── package_skill.py   # 打包 Skill（改进）
    └── quick_validate.py  # 快速验证（60 行变更）
```

**许可证**: Apache 2.0（[ 开源](#)）

### web-artifacts-builder - Web Artifacts 构建器

**功能特性**：

- 创建 Web Artifacts
- 组件打包
- shadcn 组件集成
**工具脚本**：

```
skills/web-artifacts-builder/
├── SKILL.md
└── scripts/
    ├── init-artifact.sh        # 初始化 artifact
    ├── bundle-artifact.sh      # 打包 artifact
    └── shadcn-components.tar.gz # shadcn 组件库
```

**许可证**: Apache 2.0（开源）

### webapp-testing - Web 应用测试

**功能特性**：

- Web 应用自动化测试
- 控制台日志捕获
- 元素发现
- 静态 HTML 自动化
**示例和工具**：

```
skills/webapp-testing/
├── SKILL.md
├── examples/
│   ├── console_logging.py           # 控制台日志
│   ├── element_discovery.py         # 元素发现
│   └── static_html_automation.py    # 静态 HTML 自动化
└── scripts/
    └── with_server.py               # 服务器集成
```

**许可证**: Apache 2.0（开源）

## 企业与通信 Skills

### internal-comms - 内部沟通

**功能特性**：

- 公司通讯撰写
- 常见问题解答
- 第三方更新通知
- 通用沟通模板
**示例模板**：

```
skills/internal-comms/
├── SKILL.md
└── examples/
    ├── company-newsletter.md  # 公司新闻简报
    ├── faq-answers.md         # FAQ 回答
    ├── 3p-updates.md          # 第三方更新
    └── general-comms.md       # 通用沟通
```

**许可证**: Apache 2.0（开源）

## Skill 模块化设计

每个 Skill 遵循标准的模块化结构：

```
skill-name/
├── SKILL.md          # 核心定义（必需）
│   ├── YAML frontmatter (name, description)
│   └── 主体指令（< 5k tokens）
├── LICENSE.txt       # 许可证（可选）
├── references/       # 参考文档（可选）
│   ├── REFERENCE.md
│   └── ADVANCED.md
├── scripts/          # 辅助脚本（可选）
│   └── helper.py
├── templates/        # 模板文件（可选）
│   └── template.ext
└── examples/         # 示例代码（可选）
    └── example.py
```

## 如何使用这些示例

### 1. 浏览源码
访问 [Anthropic Skills 仓库](https://github.com/anthropics/skills) 浏览完整源码：

```bash
git clone https://github.com/anthropics/skills.git
cd skills/skills
ls -la
```

### 2. 学习 Skill 结构
选择一个感兴趣的 Skill，研究其结构：

```bash
cd doc-coauthoring
cat SKILL.md
```

### 3. 复制和修改
使用官方模板创建自己的 Skill：

```bash
cp -r template/ my-custom-skill/
cd my-custom-skill
# 修改 SKILL.md
```

### 4. 参考最佳实践
查看 `skill-creator` 的新增参考文档：

- references/output-patterns.md: 输出模式最佳实践
- references/workflows.md: 工作流设计指导
## 相关资源

### 官方资源

- Skills 仓库: github.com/anthropics/skills
- 官方文档: docs.claude.com
### 学习资源

- 快速开始: 创建第一个 Skill
- 最佳实践: 编写高质量 Skills
- 架构设计: 理解 Skills 技术架构
### 社区资源

- Skills 最佳实践: best-practices
- Partner Skills: 来自合作伙伴的 Skills 示例
## 贡献指南

想要贡献自己的 Skill 到官方仓库？

- Fork 仓库: Fork anthropics/skills 到你的账号
- 创建 Skill: 遵循标准结构创建 Skill
- 测试验证: 确保 Skill 质量和完整性
- 提交 PR: 向官方仓库提交 Pull Request
详细贡献指南请查看仓库的 [CONTRIBUTING.md](https://github.com/anthropics/skills/blob/main/CONTRIBUTING.md)。

## 许可证说明

Skills 仓库中的示例有两种许可证：

**开源 Skills（Apache 2.0）**：

- 创意与设计类 Skills
- 技术开发类 Skills
- 企业通信类 Skills
- 可自由使用、修改和分发
**源码可用（非[ 开源](#)）**：

- 文档处理 Skills（docx、pdf、pptx、xlsx）
- 这些是 Claude.ai 文档能力的核心实现
- 供开发者学习参考，但不能用于商业产品
## 总结

Anthropic Skills 仓库提供了：

- 17 个官方一级 Skills 目录
- 多个领域覆盖: 文档、设计、开发、通信
- 完整的技术实现: 代码、脚本、参考文档
- 标准化结构: 易于学习和复用
- 持续更新: 反映最新最佳实践
这些示例是创建自定义 Agent Skills 的最佳学习资源。

## 下一步
[快速开始创建你的第一个 Skill
](../quickstart/)[最佳实践编写高质量 Skills
](../best-practices/)[架构设计理解 Skills 技术架构
](../architecture/)
