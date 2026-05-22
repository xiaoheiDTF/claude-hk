# Cookbook / Output-Control / Speculative-Prompt-Caching

> 来源: claudecn.com

# Speculative prompt caching（投机缓存）

探索投机缓存的模式，并对比标准缓存的收益与风险。

实操时先把输出格式与停止条件写清楚，再考虑缓存/吞吐等性能优化。

- 对应 notebook：misc/speculative_prompt_caching.ipynb
## 读的时候重点看

- 何时投机有效（高复用、分支可预测）
- 失败模式（命中率低、浪费 tokens）
- 用真实流量/数据评估取舍
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=misc/speculative_prompt_caching.ipynb
```
