# Cookbook / Tool-Use / Automatic-Context-Compaction

> 来源: claudecn.com

# 自动上下文压缩（Automatic context compaction）

这份 notebook 用 beta 的 tool runner + `compaction_control` 给长流程兜底，让对话历史在超阈值后自动压缩并插入摘要。

建议边跑边看 response.content，理解 tool_use/tool_result 的对应关系。

- 对应 notebook：tool_use/automatic-context-compaction.ipynb
## 读的时候重点看

- 何时压缩（token 阈值）
- 压缩后消息结构怎么变化（摘要 message）
- 长工单/长会话中保持行为稳定
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=tool_use/automatic-context-compaction.ipynb
```
