# Cookbook / Multimodal / Getting-Started-With-Vision

> 来源: claudecn.com

# 视觉入门（Getting started with vision）

最简单的图片输入方式（示例使用图片 URL），用来跑通“图像 + 指令”的最小闭环。

图片质量决定上限：看不清就先裁剪/分块，再让模型做分析。

- 对应 notebook：multimodal/getting_started_with_vision.ipynb
## 读的时候重点看

- image content block 的结构
- 图片与指令尽量收敛到同一个目标
- 何时从 URL 切到 base64（见 vision + tools）
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=multimodal/getting_started_with_vision.ipynb
```

## 相关内容

- 视觉 + 工具（base64）：../tool-use/vision-with-tools
