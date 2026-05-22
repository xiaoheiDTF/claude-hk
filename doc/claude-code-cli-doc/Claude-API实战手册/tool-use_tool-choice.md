# Cookbook / Tool-Use / Tool-Choice

> 来源: claudecn.com

# 工具选择（Auto / Any / 强制）

这份 notebook 主要讲 `tool_choice` 的常见模式，并强调：生产效果往往取决于“提示词契约”而不仅是参数本身。

建议边跑边看 response.content，理解 tool_use/tool_result 的对应关系。

- 对应 notebook：tool_use/tool_choice.ipynb
## 读的时候重点看

- tool_choice={"type":"auto"}：按需调用工具
- 强制某个工具：更确定，但更容易错过更合适的工具
- any 的取舍：自由更大、风险也更高
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=tool_use/tool_choice.ipynb
```
