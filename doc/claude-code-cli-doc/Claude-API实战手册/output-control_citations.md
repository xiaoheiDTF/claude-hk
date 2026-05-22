# Cookbook / Output-Control / Citations

> 来源: claudecn.com

# Citations（可追溯引用）

给输出附上可追溯引用，并展示不同文档类型下的引用组织方式。

实操时先把输出格式与停止条件写清楚，再考虑缓存/吞吐等性能优化。

- 对应 notebook：misc/using_citations.ipynb
## 读的时候重点看

- 输出结构要便于消费 citations
- 检索输入与引用要对齐（避免“漂浮引用”）
- 切分策略决定引用的颗粒度与可用性
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=misc/using_citations.ipynb
```
