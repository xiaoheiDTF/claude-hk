# Cookbook / Agent-Patterns / Evaluator-Optimizer

> 来源: claudecn.com

# Evaluator-Optimizer（评审-优化回路）

当“先产出一版”不难，但“稳定地把质量拉到可用水平”很难时，这个模式会比单次提示更有效。一个角色负责生成，另一个角色负责判断和指出改进方向，循环直到结果过线。

它的价值在于把质量控制从“隐含期待”变成“显式步骤”。

## 读的时候重点看

- judge 与 generate 分离
- 停止条件（避免无限循环）
- 如何把评审回路变成可回归的 eval（见评测专题）
## 什么时候更适合用它

- 任务结果经常“差一点”，但问题点是可描述的
- 你比起“一次写对”，更容易先定义什么叫变好
- 你希望把人工评审逐步沉淀成可重复的评测机制
## 如果你想本地复现

在本地环境已经跑通后，可以对这个主题对应的示例做结构检查：

```bash
make test-notebooks NOTEBOOK=patterns/agents/evaluator_optimizer.ipynb
```

## 相关内容

- 评测专题：../evals-and-testing
