# Cookbook / Integrations-And-Ops / Pinecone-Rag

> 来源: claudecn.com

# Pinecone RAG（向量库集成）

使用 Pinecone 作为向量库的 RAG 端到端示例。

外部依赖与 Key 往往比较多，先把环境变量、限流与成本盘清楚再动手。

- 对应 notebook：third_party/Pinecone/rag_using_pinecone.ipynb
## 读的时候重点看

- 数据摄取与索引策略
- 检索与回答合成的组合方式
- 运维视角：成本、延迟、刷新策略
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=third_party/Pinecone/rag_using_pinecone.ipynb
```
