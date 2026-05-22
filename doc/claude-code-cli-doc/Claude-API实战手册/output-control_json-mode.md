# Cookbook / Output-Control / Json-Mode

> 来源: claudecn.com

# JSON mode（稳定输出 JSON）

这份 notebook 主要讲让 JSON 输出更稳定的提示工程套路，以及何时应该换成“工具 schema 合约”。

实操时先把输出格式与停止条件写清楚，再考虑缓存/吞吐等性能优化。

- 对应 notebook：misc/how_to_enable_json_mode.ipynb
## 读的时候重点看

- 把输出形状当合约（字段、类型、缺失规则）
- 异常 JSON 的处理（重试/修复）
- 需要强约束时用 tools 代替纯提示词
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=misc/how_to_enable_json_mode.ipynb
```

## 相关内容

- 用 tool schema 抽取 JSON：../tool-use/extracting-structured-json
