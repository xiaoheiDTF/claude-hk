# Cookbook / Agent-Patterns

> 来源: claudecn.com

# Agent 模式：Patterns 与 Claude Agent SDK

如果你想借 Cookbook 学 agent，最有效的阅读方式通常不是“先学工具”，而是分两步读：

- 先看工作流模式，判断任务应该怎么拆、怎么评审、怎么汇总；
- 再看可运行的 agent 示例，把工具、权限、hooks 与外部系统接进去。
## 推荐 Notebook

### 1) 先看 Patterns：把编排套路认清
[Orchestrator-Workers协调者 + 工作者的编排模式
](orchestrator-workers/)[Evaluator-Optimizer用评审回路把质量闭环
](evaluator-optimizer/)[Basic workflows最小多步工作流套路
](basic-workflows/)
你可以把它们当作“多轮工作流的模板”：任务拆分、并行/串行、评审反馈、以及如何把质量回路闭合。

### 2) 再看 Claude Agent SDK：把套路变成可运行的 agent
[One-liner research agentWebSearch + 引用契约
](one-liner-research-agent/)[Chief of staff agent输出风格、hooks、plan mode
](chief-of-staff-agent/)[Observability agent（MCP）Git/GitHub MCP servers + 工具限制
](observability-agent-mcp/)

这些示例往往会涉及外部依赖，例如 MCP server、GitHub token 或本地脚本。建议先把一条最小本地链路跑通，再逐个扩展。

## 怎么读这一组内容更高效

### 1) 先定流程，再谈工具
多数情况下，真正该先回答的问题不是“该装什么能力”，而是：

- 这是单 agent 任务，还是多角色协作任务？
- 质量检查应该插在哪里？
- 哪些步骤适合并行？
- 哪些输出必须保持稳定？
这也是为什么建议先看 Patterns：先把工作流形状定下来，再补工程实现。

### 2) 当流程稳定后，再补工程层细节

等你知道流程怎么走，SDK 示例才会真正有价值。它们回答的是另一类问题：

- 工具权限怎么收紧
- 输出风格怎么稳定
- hooks 和本地自动化怎么接
- 怎样把 agent 连到外部系统
### 3) 可以先抓住这三个代表性例子

#### 最小研究回路

`00_The_one_liner_research_agent.ipynb` 里展示了两种很实用的模式：

- 最小用法：allowed_tools=["WebSearch"]，快速把“研究能力”跑起来
- 系统提示词把“必须给来源”写成硬约束：要求输出 markdown 链接形式的 citations，并在结尾集中到 Sources: 区块
它还演示了调大 `max_buffer_size`（示例设为 `10MB`）来处理更大的输入/产物（例如图片）。

#### 运行协调者

chief-of-staff 这个例子更适合团队场景：当你希望一个 agent 不只是“会做事”，还要在输出风格、权限边界、执行约束上保持一致时，可以重点看它。里面比较关键的点包括：

- 通过 settings='{\"outputStyle\": \"executive\"}'（以及 technical 示例）选择输出风格
- setting_sources=["project"] 才会加载 .claude/output-styles/ 等项目级文件系统配置
- 如果要启用 hooks，需要 setting_sources=["project","local"]（并且 notebook 明确强调必须同时包含两者）
#### 外部系统接入

`02_The_observability_agent.ipynb` 提供了非常“可照抄”的 MCP 连接方式：

- Git MCP server：uv run python -m mcp_server_git --repository 
- GitHub MCP server：通过 Docker 运行 ghcr.io/github/github-mcp-server，并从 GITHUB_TOKEN 注入 GITHUB_PERSONAL_ACCESS_TOKEN
它还强调了一个关键安全点：

- 需要用 disallowed_tools 才能真正限制工具集（否则 agent 可能仍能用 Bash/CLI 绕过 MCP）。
## 推荐阅读顺序

- 先用 Patterns 把“流程怎么走、哪里需要反馈/评测”定下来；
- 再决定工具层：Tool Use / PTC / Context Compaction / Memory；
- 最后再把“工程实践”（hooks、输出风格、权限、可观测性）补齐。
