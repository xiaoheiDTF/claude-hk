# Cookbook / Integrations-And-Ops / Llamaindex-Basic-Rag

> 来源: claudecn.com

# LlamaIndex 基础 RAG

这份 notebook 用 LlamaIndex 搭建 RAG pipeline，适合已经在项目里使用 LlamaIndex 的团队。

外部依赖与 Key 往往比较多，先把环境变量、限流与成本盘清楚再动手。

- 对应 notebook：third_party/LlamaIndex/Basic_RAG_With_LlamaIndex.ipynb
## 读的时候重点看

- 与你现有检索栈的映射关系
- 如何插入 rerank 与 eval
- chunking 与 metadata 对检索质量的影响
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=third_party/LlamaIndex/Basic_RAG_With_LlamaIndex.ipynb
```
