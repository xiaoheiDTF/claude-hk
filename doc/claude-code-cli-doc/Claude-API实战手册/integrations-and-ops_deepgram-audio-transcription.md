# Cookbook / Integrations-And-Ops / Deepgram-Audio-Transcription

> 来源: claudecn.com

# Deepgram 音频转写

这份 notebook 用 Deepgram 做音频转写，并让 Claude 生成采访问题（示例工作流）。

外部依赖与 Key 往往比较多，先把环境变量、限流与成本盘清楚再动手。

- 对应 notebook：third_party/Deepgram/prerecorded_audio.ipynb
## 读的时候重点看

- 分工：转写 vs 总结/生成
- 时间戳与说话人处理
- 让输出更可行动（问题清单、摘要）
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=third_party/Deepgram/prerecorded_audio.ipynb
```
