# Cookbook / Agent-Patterns / Orchestrator-Workers

> 来源: claudecn.com

# Orchestrator-Workers（编排-工作者）

当一个任务已经大到不适合靠单轮提示一次做完，但又能自然拆成几个子任务时，这个模式就很有价值。一个角色负责拆解与协调，其他角色负责完成边界明确的小任务，再由前者统一汇总。

它的核心价值不在于“agent 变多了”，而在于职责更清晰、提示更短、检查点更明确。

## 读的时候重点看

- 拆解：哪些该在 orchestrator 做、哪些交给 worker
- 步骤接口（输入/输出）要当成合约
- 在哪里插入评审/评测关卡
## 什么时候更适合用它

- 任务能拆成彼此相对独立的部分
- 每个 worker 都能产出小而可检查的结果
- 你希望利用并行能力，但又不想失去最后的统一控制
## 如果你想本地复现

在本地环境已经跑通的前提下，可以对这个主题对应的示例做结构检查：

```bash
make test-notebooks NOTEBOOK=patterns/agents/orchestrator_workers.ipynb
```
