# Cookbook / Tool-Use / Parallel-Tools

> 来源: claudecn.com

# 并行工具调用（Parallel tool calls）

这份 notebook 演示一次响应里处理多个 `tool_use` 的模式，并引入“batch tool”封装思路（把多次调用打包成一次）。

建议边跑边看 response.content，理解 tool_use/tool_result 的对应关系。

- 对应 notebook：tool_use/parallel_tools.ipynb
## 读的时候重点看

- 遍历 response.content，处理多个 tool_use
- tool_result 必须带正确的 tool_use_id
- batch tool：减少回合数，降低端到端延迟
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=tool_use/parallel_tools.ipynb
```
