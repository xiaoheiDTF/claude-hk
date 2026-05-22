# Cookbook / Rag-And-Retrieval / Classification

> 来源: claudecn.com

# 分类（Classification）

当检索流程在回答前必须先做判断，比如先选数据源、先选工具、或先决定提示策略时，分类就是很常见的一层。

它的重点不只是“把东西分门别类”，而是把后续路径分对。

## 读的时候重点看

- 标签设计与边界（模糊类最容易炸）
- 需要领域知识时，引入检索上下文再分类
- 用真实分布的测试集做评测
## 什么时候更适合用它

- 系统必须在多个后续分支之间做选择
- 标签会直接决定下一步动作，而不是只做静态归类
- 你能拿到足够真实的边界样本来检验效果
## 如果你想本地复现

在本地 Cookbook 环境已经准备好后，可以对这个主题对应的示例做结构检查：

```bash
make test-notebooks NOTEBOOK=capabilities/classification/guide.ipynb
```
