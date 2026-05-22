# Cookbook / Rag-And-Retrieval / Retrieval-Augmented-Generation

> 来源: claudecn.com

# 检索增强生成（RAG）

当模型只靠当前提示已经回答不到位，而你又需要在运行时补充外部知识时，RAG 往往是最先该考虑的方案。

这个示例真正有价值的地方，不只是“把检索接上去”，而是它展示了怎样从一个最小可用基线开始，再一步步引入更好的检索组织、排序和评测，把“感觉更好”变成“可以比较的提升”。

如果你刚开始接触检索系统，这通常比一上来就做上下文化检索、混合检索或知识图谱更适合作为起点。

## 读的时候重点看

- 最小向量库封装（embeddings、缓存、落盘）
- 每一步升级如何改变召回/精度取舍
- 引入 eval 让改动可度量、可回归
## 什么时候更适合用它

- 回答依赖提示里放不下的外部知识
- 参考资料变化比提示词本身更频繁
- 你想先建立可比较的基线，再进入更复杂的检索设计
## 如果你想本地复现

在本地 Cookbook 环境已经跑通后，可以对这个主题对应的示例做结构检查：

```bash
make test-notebooks NOTEBOOK=capabilities/retrieval_augmented_generation/guide.ipynb
```

这个示例可能依赖额外的密钥或服务，例如 embeddings 提供方。真正接入业务数据前，先把 setup 部分看完。
