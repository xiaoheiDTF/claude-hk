# Cookbook / Output-Control

> 来源: claudecn.com

# 输出控制：JSON、Citations 与缓存

除了“把 Claude 跑起来”，很多生产问题其实发生在输出层：要 **稳定格式**、要 **可追溯引用**、要 **可控成本**、还要 **批处理吞吐**。

## 推荐 Notebook（按常见问题域）

### 1) 稳定结构化输出
[JSON mode让 JSON 输出更稳定的提示工程
](json-mode/)[Structured JSON + tool use用 tools 做 schema 合约
](../tool-use/extracting-structured-json)

### 2) 引用与可追溯（Citations）
[Citations（引用）跨文档类型的可追溯引用
](citations/)

### 3) 成本与延迟：Prompt caching
[Prompt caching可缓存提示词的构造方式
](prompt-caching/)[Speculative prompt caching投机缓存的对比与取舍
](speculative-prompt-caching/)

### 4) 吞吐：Batch Processing
[Message Batches API批处理的吞吐与回收流程
](batch-processing/)

### 5) 安全：内容审核/过滤
[Moderation filter构建内容审核过滤器
](moderation-filter/)

### 6) 复杂长输出（可选）
[Sampling past max tokens极长输出的策略
](sampling-past-max-tokens/)

### 7) Prompt 工程化（可选）
[Metaprompt提示词模板与测试回路
](metaprompt/)

## 经验法则（更偏工程侧）

- 把“格式”当 API 合约：JSON 输出不要只靠提示词，最好配合工具/校验器做强约束与重试。
- 缓存不是银弹：prompt caching 适合“固定前缀 + 多次复用”的场景；先量化命中率再上生产。
- 批处理要有监控：batch 的“排队/失败/重试/结果回收”需要可观测性，否则很难定位吞吐瓶颈。
