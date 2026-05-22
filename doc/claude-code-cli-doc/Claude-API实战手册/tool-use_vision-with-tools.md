# Cookbook / Tool-Use / Vision-With-Tools

> 来源: claudecn.com

# 视觉 + 工具（Vision with tools）

把图片输入与 tool use 结合做结构化抽取（示例：营养成分表；图片以 base64 方式传入）。

建议边跑边看 response.content，理解 tool_use/tool_result 的对应关系。

- 对应 notebook：tool_use/vision_with_tools.ipynb
## 读的时候重点看

- 图片 content block（base64）+ 文本指令组合
- 抽取字段用工具 schema 固化成“合约”
- 应用侧校验 + 重试，提升抽取稳定性
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=tool_use/vision_with_tools.ipynb
```

## 相关内容

- 多模态总览：../multimodal
