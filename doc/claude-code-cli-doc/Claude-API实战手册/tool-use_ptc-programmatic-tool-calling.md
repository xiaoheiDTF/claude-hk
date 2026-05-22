# Cookbook / Tool-Use / Ptc-Programmatic-Tool-Calling

> 来源: claudecn.com

# PTC：程序化工具调用

这份 notebook 介绍 PTC（Programmatic Tool Calling）：在受控的 code execution 环境里，让代码来编排与调用工具。

建议边跑边看 response.content，理解 tool_use/tool_result 的对应关系。

- 对应 notebook：tool_use/programmatic_tool_calling_ptc.ipynb
## 读的时候重点看

- 传统 tool loop 与 PTC 的性能/成本权衡
- 限制可调用者（示例里用 allowed_callers）
- 把 code execution 当“受权限控制的能力”来审计与隔离
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=tool_use/programmatic_tool_calling_ptc.ipynb
```
