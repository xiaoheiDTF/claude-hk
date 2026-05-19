# Cookbook / Tool-Use / Customer-Service-Agent

> 来源: claudecn.com

# 客服 Agent（客户端工具）

这份 notebook 演示客服工作流里“客户端执行工具”的写法，并给出一条很实用的开发技巧：用**模拟的 tool 输出**先把对话闭环跑通。

建议边跑边看 response.content，理解 tool_use/tool_result 的对应关系。

- 对应 notebook：tool_use/customer_service_agent.ipynb
## 读的时候重点看

- 客户端工具设计要可控（避免隐藏副作用）
- 用 synthetic tool results 做开发期联调
- 工具入参/出参要可审计、可测试
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=tool_use/customer_service_agent.ipynb
```
