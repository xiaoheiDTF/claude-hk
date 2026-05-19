# Cookbook / Output-Control / Metaprompt

> 来源: claudecn.com

# Metaprompt（提示词工程化）

这份 notebook 用“像写代码一样写提示词”的工作流：模板化、测试回路、系统化迭代。

实操时先把输出格式与停止条件写清楚，再考虑缓存/吞吐等性能优化。

- 对应 notebook：misc/metaprompt.ipynb
## 读的时候重点看

- 提示词版本化与可测试性
- 用合成测试用例验证改动
- 模板尽量小且可组合
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=misc/metaprompt.ipynb
```
