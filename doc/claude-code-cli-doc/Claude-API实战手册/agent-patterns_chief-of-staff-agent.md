# Cookbook / Agent-Patterns / Chief-Of-Staff-Agent

> 来源: claudecn.com

# Chief of staff agent（参谋长代理）

当一个 agent 不只是要“把事做完”，还要遵守团队的输出风格、权限边界、hooks 规则与计划方式时，这个例子就会很有参考价值。

它很适合用来理解：怎样把个人试验，推进成团队可复用的工作流。

## 读的时候重点看

- 输出风格：settings + setting_sources=["project"]
- hooks 加载：setting_sources=["project","local"]
- 工具权限收紧：allowed_tools=[...]
## 什么时候更适合用它

- 团队希望输出风格更统一
- 本地自动化或 hooks 会直接影响 agent 行为
- 权限与审查边界的重要性已经不低于提示词本身
## 如果你想本地复现

在本地环境已经准备好之后，可以对这个主题对应的示例做结构检查：

```bash
make test-notebooks NOTEBOOK=claude_agent_sdk/01_The_chief_of_staff_agent.ipynb
```
