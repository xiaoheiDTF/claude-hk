# Cookbook / Rag-And-Retrieval / Summarization

> 来源: claudecn.com

# 摘要（Summarization）

当原始材料太长、太乱，或者结构差异太大，没法直接稳定地进入后续检索或决策环节时，摘要往往就是第一层整理。

好的摘要不只是把文本变短，而是形成一份稳定的信息合约：保留什么、舍弃什么、下游还能依赖哪些结构。

## 读的时候重点看

- 摘要格式做成合约（长度、结构、禁止项）
- 用 eval 捕捉幻觉/遗漏
- 何时把 summary 写进索引（summary-index）
## 什么时候更适合用它

- 原始文档太长，或者结构很不统一
- 后续步骤需要更规整的输入表示
- 你希望摘要本身也能被检查、比较和复用
## 如果你想本地复现

在本地 Cookbook 环境已经准备好后，可以对这个主题对应的示例做结构检查：

```bash
make test-notebooks NOTEBOOK=capabilities/summarization/guide.ipynb
```
