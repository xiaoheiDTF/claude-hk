# Cookbook / Multimodal / Transcribe-Documents

> 来源: claudecn.com

# 文档转写（Transcribe documents）

覆盖印刷体、手写体、表单等文档转写，并包含“非结构化 → JSON”的输出策略。

图片质量决定上限：看不清就先裁剪/分块，再让模型做分析。

- 对应 notebook：multimodal/how_to_transcribe_text.ipynb
## 读的时候重点看

- 先定抽取目标（字段）再转写，避免无效成本
- 表单类优先输出 JSON，并在应用侧校验
- 扫描质量差时先裁剪/分块
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=multimodal/how_to_transcribe_text.ipynb
```
