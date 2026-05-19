# Cookbook / Rag-And-Retrieval / Sql-Queries

> 来源: claudecn.com

# SQL 查询（实战配方）

如果你已经明确 SQL 就是正确接口，这一页更适合拿来解决“怎么把它用稳”的问题。

和更抽象的 Text-to-SQL 讨论相比，这个示例更偏实战：怎样给 schema 上下文、怎样把执行边界收紧、怎样让生成结果更适合进入真实流程。

## 读的时候重点看

- 提供 schema 上下文时的安全边界
- 执行侧 guardrails：只读、limit、参数化
- 让 SQL 输出可自动化（更确定、更可验证）
## 什么时候更适合用它

- 团队本身已经大量使用 SQL 做分析或报表
- 你现在更缺的是实践性安全措施，而不是新奇能力
- 生成的查询会进入 脚本、报表或自动化任务链路
## 如果你想本地复现

在本地 Cookbook 环境已经准备好后，可以对这个主题对应的示例做结构检查：

```bash
make test-notebooks NOTEBOOK=misc/how_to_make_sql_queries.ipynb
```
