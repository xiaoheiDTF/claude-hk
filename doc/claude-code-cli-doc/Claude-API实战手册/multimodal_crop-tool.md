# Cookbook / Multimodal / Crop-Tool

> 来源: claudecn.com

# 裁剪工具（Crop tool）

给 Claude 一个裁剪工具：先发现“需要放大看的区域”，再裁剪后二次分析，形成实用的 agentic loop。

图片质量决定上限：看不清就先裁剪/分块，再让模型做分析。

- 对应 notebook：multimodal/crop_tool.ipynb
## 读的时候重点看

- “发现 → 裁剪 → 再分析”闭环
- 裁剪工具设计（坐标、输出）与安全边界
- 需要细节时，裁剪往往比“更长提示词”更有效
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=multimodal/crop_tool.ipynb
```
