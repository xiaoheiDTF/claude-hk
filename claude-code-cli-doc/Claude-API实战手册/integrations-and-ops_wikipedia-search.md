# Cookbook / Integrations-And-Ops / Wikipedia-Search

> 来源: claudecn.com

# Wikipedia 迭代检索

在 Wikipedia 上做“检索 → 阅读 → 细化”的迭代流程，适合作为轻量外部知识源。

外部依赖与 Key 往往比较多，先把环境变量、限流与成本盘清楚再动手。

- 对应 notebook：third_party/Wikipedia/wikipedia-search-cookbook.ipynb
## 读的时候重点看

- 迭代 loop：search → read → refine
- 引用与检索段落对齐
- 何时升级为可索引的 RAG 系统
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=third_party/Wikipedia/wikipedia-search-cookbook.ipynb
```
