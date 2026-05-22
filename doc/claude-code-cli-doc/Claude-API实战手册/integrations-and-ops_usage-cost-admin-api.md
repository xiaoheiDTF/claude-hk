# Cookbook / Integrations-And-Ops / Usage-Cost-Admin-Api

> 来源: claudecn.com

# 用量与成本 Admin API

这份 notebook 用量/成本追踪脚手架：分组、过滤、分页、导出，便于接入监控与告警。

外部依赖与 Key 往往比较多，先把环境变量、限流与成本盘清楚再动手。

- 对应 notebook：observability/usage_cost_api.ipynb
## 读的时候重点看

- 查询模式（过滤、分页）
- 导出格式（BI/告警/报表）
- 把原始用量转成团队级 guardrails
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=observability/usage_cost_api.ipynb
```
