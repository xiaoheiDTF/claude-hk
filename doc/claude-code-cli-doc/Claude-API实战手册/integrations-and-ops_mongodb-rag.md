# Cookbook / Integrations-And-Ops / Mongodb-Rag

> 来源: claudecn.com

# MongoDB RAG（数据库集成）

使用 MongoDB 构建 RAG 系统的示例。

外部依赖与 Key 往往比较多，先把环境变量、限流与成本盘清楚再动手。

- 对应 notebook：third_party/MongoDB/rag_using_mongodb.ipynb
## 读的时候重点看

- 面向检索的数据建模
- 查询与排序策略
- 部署与索引（吞吐、稳定性）
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=third_party/MongoDB/rag_using_mongodb.ipynb
```
