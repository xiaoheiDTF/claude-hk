# Cookbook / Rag-And-Retrieval / Text-To-Sql

> 来源: claudecn.com

# Text-to-SQL（自然语言转 SQL）

当用户用自然语言提问，但你的真实系统边界仍然是一套数据库时，Text-to-SQL 就是最直接的桥梁。

这里最难的通常不只是“把 SQL 写出来”，而是让模型真正理解 schema、在执行上保持安全，并且有办法判断结果到底对不对。

## 读的时候重点看

- schema grounding：让 SQL 生成“有上下文、有约束”
- 评测：避免“看起来对”但不可执行/不正确
- 执行安全：只读、limit、参数化与权限隔离
## 什么时候更适合用它

- 问题本身对应的是结构化表数据
- 你能提供足够 schema 上下文，又不会暴露敏感信息
- 执行安全和结果质量同样重要
## 如果你想本地复现

在本地 Cookbook 环境已经准备好后，可以对这个主题对应的示例做结构检查：

```bash
make test-notebooks NOTEBOOK=capabilities/text_to_sql/guide.ipynb
```
