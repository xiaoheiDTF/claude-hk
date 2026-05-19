# Cookbook / Multimodal / Best-Practices-For-Vision

> 来源: claudecn.com

# 视觉最佳实践（Best practices for vision）

这份 notebook 主要讲更稳的多模态提示工程套路（visual prompting、few-shot、多图输入等）。

图片质量决定上限：看不清就先裁剪/分块，再让模型做分析。

- 对应 notebook：multimodal/best_practices_for_vision.ipynb
## 读的时候重点看

- few-shot 让抽取更一致
- 多图输入（对比、序列）
- 先定义输出合约（结构、约束、拒答边界）
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=multimodal/best_practices_for_vision.ipynb
```
