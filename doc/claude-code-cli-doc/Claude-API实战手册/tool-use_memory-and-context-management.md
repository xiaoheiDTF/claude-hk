# Cookbook / Tool-Use / Memory-And-Context-Management

> 来源: claudecn.com

# 记忆与上下文管理（Memory & context）

这份 notebook 主要讲长运行 agent 的记忆策略与安全边界，覆盖 context editing、最佳实践与常见风险。

建议边跑边看 response.content，理解 tool_use/tool_result 的对应关系。

- 对应 notebook：tool_use/memory_cookbook.ipynb
## 读的时候重点看

- 什么该持久化、什么该摘要、什么该丢弃
- 安全边界：避免把密钥/隐私写入“记忆”
- 不膨胀上下文的长流程稳定性策略
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=tool_use/memory_cookbook.ipynb
```
