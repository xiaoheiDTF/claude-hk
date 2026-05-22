# Cookbook / Rag-And-Retrieval / Contextual-Retrieval

> 来源: claudecn.com

# 上下文化检索（Contextual retrieval）

当基础检索已经能找到“大致相关”的片段，但这些片段一旦脱离全文语境就说不清意思时，就该考虑上下文化检索了。

它的关键做法，是先给每个 chunk 补上一层轻量的定位上下文，让检索命中的不再只是“词很像”的片段，而是“语义位置也更对”的片段，再配合缓存和混合检索提高稳定性。

通常更合理的顺序是：先确认基础 RAG 方向成立，再进入这一层优化，而不是一开始就把复杂度拉满。

## 读的时候重点看

- chunk 定位（situating context）提示词：短、面向检索
- prompt caching 的用法（cache_control: {type: "ephemeral"}）
- 混合检索：BM25（词法）+ 语义检索
## 什么时候更适合用它

- 相关片段能搜到，但上下文不够，回答仍然发虚
- 相似片段很多，模型容易混淆不同段落的含义
- 基础 RAG 已经可用，但在真实文档上还不够稳
## 如果你想本地复现

在本地 Cookbook 环境已经准备好之后，可以对这个主题对应的示例做结构检查：

```bash
make test-notebooks NOTEBOOK=capabilities/contextual-embeddings/guide.ipynb
```

混合 BM25 章节使用 Elasticsearch（示例连接 `http://localhost:9200`）。
