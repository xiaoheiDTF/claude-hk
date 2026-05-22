# Cookbook / Output-Control / Prompt-Caching

> 来源: claudecn.com

# Prompt caching（提示词缓存）

这份 notebook 演示 Claude API 的 prompt caching：对比 baseline 与 cache 命中，并解释适用条件。

实操时先把输出格式与停止条件写清楚，再考虑缓存/吞吐等性能优化。

- 对应 notebook：misc/prompt_caching.ipynb
## 读的时候重点看

- 哪些内容适合作为可缓存前缀（稳定上下文）
- 如何量化收益（tokens/延迟）与复杂度
- 与检索结合（文档上下文作为缓存块）
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=misc/prompt_caching.ipynb
```
