# Cookbook / Output-Control / Moderation-Filter

> 来源: claudecn.com

# 内容审核过滤器（Moderation filter）

构建内容审核过滤器，并讨论如何通过自定义/示例等方式提升效果。

实操时先把输出格式与停止条件写清楚，再考虑缓存/吞吐等性能优化。

- 对应 notebook：misc/building_moderation_filter.ipynb
## 读的时候重点看

- 策略类别与阈值定义
- “判定”与“执行”分离（避免直接自动处置）
- 用真实数据评估误报/漏报
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=misc/building_moderation_filter.ipynb
```
