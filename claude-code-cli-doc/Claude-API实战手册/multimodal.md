# Cookbook / Multimodal

> 来源: claudecn.com

# 多模态：视觉、图表与文档转写

Cookbooks 的多模态（vision）内容覆盖了三类常见任务：**看图理解**、**看图提取结构化信息**、以及**围绕图像的 agentic loop（例如裁剪工具）**。

## 概述

Claude 的视觉能力让您能够以强大的方式处理图像：

- 图像理解：分析并描述图片、图表和文档
- 结构化提取：从表单、发票和视觉内容中提取数据
- 视觉 Agent 工作流：结合视觉与工具完成复杂的多步骤任务
本节提供了 6 个实用 [ notebook](#)，包含可直接复制的代码示例。

## 推荐 Notebook（按任务类型）

### 1) 入门：怎么把图片“喂给”Claude
[Getting started with vision把图片（URL）传给 Claude
](getting-started-with-vision/)

### 2) 视觉提示工程：让结果更稳定
[Best practices for vision多模态提示工程的稳定性套路
](best-practices-for-vision/)

### 3) 专项：图表、幻灯片、表单与转写
[Charts/graphs/slide decks图表与幻灯片的分析流程
](charts-graphs-and-slide-decks/)[Transcribe documents印刷体/手写体/表单 → 结构化输出
](transcribe-documents/)

### 4) Agentic：给 Claude 一个“裁剪工具”
[Crop tool用裁剪工具放大细节再二次分析
](crop-tool/)
这类做法适合图表/文档细节密集的场景：先让 Claude 发现“需要放大看的区域”，再用工具裁剪并二次分析。

### 5) 视觉 + 工具

视觉 + 工具 的 notebook 统一放在工具调用专题里：

- ../tool-use/vision-with-tools
### 6) 子 agent（可选）
[Using Haiku as a sub-agentPDF → 图片 → 子 agent 抽取再汇总
](using-sub-agents/)

## 实战提示

- 先把输入做干净：清晰的图片 > 更长的提示词；必要时用裁剪/分块减少“看不清”。
- 结构化输出优先：转写/表单抽取通常更适合输出 JSON，并在代码侧做校验与重试。
- 注意隐私与合规：图片里常包含敏感信息，落盘/传输前要有脱敏与权限策略。
