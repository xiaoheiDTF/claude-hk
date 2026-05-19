# Cookbook / Tool-Use / Session-Memory-Compaction

> 来源: claudecn.com

# 会话记忆压缩（Session memory compaction）

这份 notebook 讲的是**对话类应用**里“手动、可控、提前准备”的会话记忆管理：在后台持续生成一份精炼的 session memory，这样当上下文接近阈值时可以做到**几乎瞬时压缩**，避免用户等待。

- 对应 notebook：misc/session_memory_compaction.ipynb
## 适用场景

- 你在做长对话产品（编码助手、客服、写作/协作工具）。
- 你希望压缩是“无感”的：不出现明显的总结等待过程。
- 你需要比“自动压缩”更强的可控性（保留哪些信息、丢弃哪些信息、何时刷新）。
## 建议重点看

- session memory 提示词怎么写：保留目标、约束、关键决策、已确认事实
- 后台刷新（threading）让 compaction 随时可用
- 用 prompt caching 降低后台刷新成本
## 相关 notebook

- 面向 agent workflow 的自动压缩：tool_use/automatic-context-compaction.ipynb
- 记忆与安全边界的整体方法：tool_use/memory_cookbook.ipynb
## 本地运行

```bash
make test-notebooks NOTEBOOK=misc/session_memory_compaction.ipynb
```
