# Cookbook / Agent-Patterns / One-Liner-Research-Agent

> 来源: claudecn.com

# One-liner research agent（单行研究代理）

当你想搭一条最小研究链路，而且希望工具边界清晰、来源可追溯时，这个例子很合适。

它的优势不在于“能力很多”，而在于很容易审计：工具范围很窄，输出规则很明确，也方便逐步检查 agent 到底被允许做什么。

## 读的时候重点看

- allowed_tools=["WebSearch"]：窄能力、好审计
- 用 system prompt 约束 citations 与 Sources: 输出区块
- 何时升级为带状态的 SDK client
## 什么时候更适合用它

- 你需要快速做外部研究，但又希望来源可核查
- 你想先搭最小 agent，再逐步增加状态或工具
- 你更关心稳定与可复查，而不是一开始就能力铺满
## 如果你想本地复现

在本地环境已经准备好之后，可以对这个主题对应的示例做结构检查：

```bash
make test-notebooks NOTEBOOK=claude_agent_sdk/00_The_one_liner_research_agent.ipynb
```
