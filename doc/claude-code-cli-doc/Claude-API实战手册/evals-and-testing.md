# Cookbook / Evals-And-Testing

> 来源: claudecn.com

# 质量与评测：Evals、测试数据与 Tool Evaluation

Cookbooks 的一个重要价值在于：不仅给“能跑的示例”，还给“怎么评测/怎么对齐质量”的方法论与可运行[ 脚本](#)。

## 推荐 Notebook

### 1) 构建 Evals（从概念到落地）
[Building evals评测要素：代码/人工/模型打分
](building-evals/)

### 2) 生成测试用例（合成数据）
[Generate synthetic test cases为提示词模板生成测试数据
](generate-test-cases/)

### 3) Tool Evaluation（专门评测工具调用）
[Tool evaluation评测工具调用行为与 agent loop
](tool-evaluation/)

### 4) Agent 里的质量回路（可选）
Evaluator-Optimizer 归档在 Agent 模式专题里：

- ../agent-patterns/evaluator-optimizer
## 与仓库“质量门禁”的关系

如果你在团队里要复用/二次开发这些 notebook，建议把质量收敛到两条线：

- 结构与可复现：make test-notebooks（快、适合 CI）
- 真实执行与回归：make test-notebooks-exec（慢、适合定期/抽样）
先让“最小评测”跑通，再逐步把指标、样本覆盖、以及失败诊断做细，通常比一次性追求大而全更稳。
