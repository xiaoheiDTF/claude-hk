# Cookbook / Output-Control / Batch-Processing

> 来源: claudecn.com

# 批处理（Message Batches API）

高吞吐的批处理流程：创建 batch、监控进度、回收结果。

实操时先把输出格式与停止条件写清楚，再考虑缓存/吞吐等性能优化。

- 对应 notebook：misc/batch_processing.ipynb
## 读的时候重点看

- 生命周期：create → monitor → collect
- 失败处理（部分失败、重试、回收）
- 可观测性（成本、延迟、成功率）
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=misc/batch_processing.ipynb
```
