# Cookbook / Integrations-And-Ops / Finetuning-On-Bedrock

> 来源: claudecn.com

# Amazon Bedrock 微调

在 Amazon Bedrock 上微调 Claude 3 Haiku：数据集准备 → 上传 S3 → 启动作业 → 使用与验证。

外部依赖与 Key 往往比较多，先把环境变量、限流与成本盘清楚再动手。

- 对应 notebook：finetuning/finetuning_on_bedrock.ipynb
## 读的时候重点看

- 数据集卫生（质量、隐私、标注）
- 作业生命周期与运维步骤
- 用评测对比微调前后行为
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=finetuning/finetuning_on_bedrock.ipynb
```
