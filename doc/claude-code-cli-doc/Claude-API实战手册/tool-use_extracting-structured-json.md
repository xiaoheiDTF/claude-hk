# Cookbook / Tool-Use / Extracting-Structured-Json

> 来源: claudecn.com

# 结构化 JSON 抽取（用 Tool Use 做合约）

这份 notebook 用“工具的 `input_schema`”来约束抽取字段（摘要、实体、情感、分类等），把结构化输出从“提示词祈祷”升级为“可验证的合约”。

建议边跑边看 response.content，理解 tool_use/tool_result 的对应关系。

- 对应 notebook：tool_use/extracting_structured_json.ipynb
## 读的时候重点看

- 用工具 schema 当输出合约（优先于纯 JSON mode）
- 必填字段与类型尽量交给 schema + 应用校验兜底
- 遇到 unknown keys/异常形状要防御式处理
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=tool_use/extracting_structured_json.ipynb
```

## 相关内容

- JSON mode：../output-control/json-mode
