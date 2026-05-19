# Cookbook / Tool-Use / Calculator-Tool

> 来源: claudecn.com

# 计算器工具（Calculator tool）

这份 notebook 用一个最小的计算器工具跑通“工具调用闭环”，理解 `tools`、`tool_use` 与真实代码执行的分工。

建议边跑边看 response.content，理解 tool_use/tool_result 的对应关系。

- 对应 notebook：tool_use/calculator_tool.ipynb
## 读的时候重点看

- 工具 schema：name / description / input_schema
- 工具由你的应用执行（不是模型执行）
- 结果回传使用 tool_result，并用 tool_use_id 关联
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=tool_use/calculator_tool.ipynb
```

## 建议下一步看什么

- 需要模型选择工具：../tool-choice
- 需要一次并行多工具：../parallel-tools
