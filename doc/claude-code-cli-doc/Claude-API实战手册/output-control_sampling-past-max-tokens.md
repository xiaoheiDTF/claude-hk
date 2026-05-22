# Cookbook / Output-Control / Sampling-Past-Max-Tokens

> 来源: claudecn.com

# 超过 max tokens 的长输出（Sampling past max tokens）

这份 notebook 主要讲超长输出的生成策略：分段生成、续写提示、以及如何保持结构一致。

实操时先把输出格式与停止条件写清楚，再考虑缓存/吞吐等性能优化。

- 对应 notebook：misc/sampling_past_max_tokens.ipynb
## 读的时候重点看

- 分段计划（大纲 → 各章节）
- 续写提示要保持结构合约
- 安全停止条件（避免无止境生成）
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=misc/sampling_past_max_tokens.ipynb
```
