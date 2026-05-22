# Cookbook / Agent-Patterns / Observability-Agent-Mcp

> 来源: claudecn.com

# Observability agent（MCP 可观测性代理）

当 agent 需要真正连到外部系统，而且你必须把接入过程做得可观察、可审查、可控时，这个例子就很关键。

它更适合已经越过“提示词怎么写”阶段、开始处理真实权限与系统边界的人。

## 读的时候重点看

- MCP server wiring：git 用 uv，GitHub 用 Docker
- Token：GITHUB_TOKEN → GITHUB_PERSONAL_ACCESS_TOKEN
- 真正限制工具集需要 disallowed_tools（避免 agent 回退到 Bash）
## 什么时候更适合用它

- agent 需要读写或观察外部系统
- 你必须清楚限定权限边界
- 你当前更关心运行安全，而不是继续叠加能力点
## 如果你想本地复现

在本地环境已经准备好之后，可以对这个主题对应的示例做结构检查：

```bash
make test-notebooks NOTEBOOK=claude_agent_sdk/02_The_observability_agent.ipynb
```
