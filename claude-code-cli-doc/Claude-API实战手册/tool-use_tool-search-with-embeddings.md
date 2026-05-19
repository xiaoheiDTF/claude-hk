# Cookbook / Tool-Use / Tool-Search-With-Embeddings

> 来源: claudecn.com

# Tool Search（用 embeddings 扩展到海量工具）

当工具数量从几十增长到上千时，用 embeddings 做“工具检索”先筛选，再进入 tool loop。

建议边跑边看 response.content，理解 tool_use/tool_result 的对应关系。

- 对应 notebook：tool_use/tool_search_with_embeddings.ipynb
## 读的时候重点看

- 构建 tool catalog（名称/描述）
- embeddings + 近邻检索先 shortlist
- 最终执行层仍要保持确定性与可审计
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=tool_use/tool_search_with_embeddings.ipynb
```
