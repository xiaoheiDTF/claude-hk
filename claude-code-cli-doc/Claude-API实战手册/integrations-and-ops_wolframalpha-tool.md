# Cookbook / Integrations-And-Ops / Wolframalpha-Tool

> 来源: claudecn.com

# WolframAlpha 工具调用

把 Wolfram Alpha LLM API 作为工具接入，展示外部计算/知识系统的 tool use 模式。

外部依赖与 Key 往往比较多，先把环境变量、限流与成本盘清楚再动手。

- 对应 notebook：third_party/WolframAlpha/using_llm_api.ipynb
## 读的时候重点看

- 工具接口设计（入参/出参）
- 外部 API 的失败处理
- 调用审计、限流与权限边界
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=third_party/WolframAlpha/using_llm_api.ipynb
```
