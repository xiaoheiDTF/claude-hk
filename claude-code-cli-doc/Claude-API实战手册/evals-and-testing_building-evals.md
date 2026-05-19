# Cookbook / Evals-And-Testing / Building-Evals

> 来源: claudecn.com

# 构建 Evals（评测体系）

这份 notebook 主要讲评测的基本要素（代码打分、人工打分、模型打分）以及如何把它们接入迭代回路。

先把最小 eval 跑通并能复现，再逐步扩充指标与样本覆盖。

- 对应 notebook：misc/building_evals.ipynb
## 读的时候重点看

- 指标要能映射到产品目标
- 数据集构建与打分逻辑分离
- 让 eval 可在 CI 中跑（快、可定位）
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=misc/building_evals.ipynb
```
