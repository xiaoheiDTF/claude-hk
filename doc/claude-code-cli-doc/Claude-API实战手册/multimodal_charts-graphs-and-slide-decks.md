# Cookbook / Multimodal / Charts-Graphs-And-Slide-Decks

> 来源: claudecn.com

# 图表与幻灯片（Charts/graphs/slide decks）

图表/图形与 PPT 的阅读流程：包含摄取、调用 API、以及如何组织问题让模型更“可解释”。

图片质量决定上限：看不清就先裁剪/分块，再让模型做分析。

- 对应 notebook：multimodal/reading_charts_graphs_powerpoints.ipynb
## 读的时候重点看

- 明确读图步骤（坐标轴、单位、序列）
- 先抽取事实再生成叙述
- 多页幻灯片：逐页抽取 vs 全局汇总
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=multimodal/reading_charts_graphs_powerpoints.ipynb
```
