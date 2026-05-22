# Cookbook / Multimodal / Using-Sub-Agents

> 来源: claudecn.com

# 子代理抽取（Using sub-agents）

这份 notebook 演示子代理工作流：先生成更聚焦的抽取提示词，把 PDF 转成图片做抽取，最后再汇总输出。

图片质量决定上限：看不清就先裁剪/分块，再让模型做分析。

- 对应 notebook：multimodal/using_sub_agents.ipynb
## 读的时候重点看

- 子代理分工：抽取 vs 汇总
- PDF → 图片，提升抽取鲁棒性
- 中间产物要可审计、可复现
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=multimodal/using_sub_agents.ipynb
```
