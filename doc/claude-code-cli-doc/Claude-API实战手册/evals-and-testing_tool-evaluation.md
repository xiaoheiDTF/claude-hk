# Cookbook / Evals-And-Testing / Tool-Evaluation

> 来源: claudecn.com

# Tool Evaluation（工具调用评测）

评测工具调用行为：正确的工具、正确的参数、正确的顺序，以及 agent loop 的整体表现。

先把最小 eval 跑通并能复现，再逐步扩充指标与样本覆盖。

- 对应 notebook：tool_evaluation/tool_evaluation.ipynb
## 读的时候重点看

- 定义“好工具调用”的标准
- 统计失败类型（误用工具、缺失调用、参数错误）
- 将结果反哺到 schema / prompt / 运行时策略
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=tool_evaluation/tool_evaluation.ipynb
```
