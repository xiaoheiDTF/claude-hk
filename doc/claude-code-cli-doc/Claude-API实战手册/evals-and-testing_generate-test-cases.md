# Cookbook / Evals-And-Testing / Generate-Test-Cases

> 来源: claudecn.com

# 生成合成测试用例（Synthetic test cases）

为提示词模板生成合成测试数据，用于快速补齐覆盖面并支持回归测试。

先把最小 eval 跑通并能复现，再逐步扩充指标与样本覆盖。

- 对应 notebook：misc/generate_test_cases.ipynb
## 读的时候重点看

- 把提示词模板当成“可测试的代码”
- 覆盖 edge cases，而不是只生成“正常样例”
- 作为回归数据集持续运行
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=misc/generate_test_cases.ipynb
```
