# Cookbook / Integrations-And-Ops / Elevenlabs-Voice-Assistant

> 来源: claudecn.com

# 低延迟语音助手（ElevenLabs）

把 STT/TTS 与 Claude 组合成低延迟语音助手的示例。

外部依赖与 Key 往往比较多，先把环境变量、限流与成本盘清楚再动手。

- 对应 notebook：third_party/ElevenLabs/low_latency_stt_claude_tts.ipynb
## 读的时候重点看

- 流式与延迟预算
- STT/TTS 的重试与容错
- 语音输出的安全控制
## 怎么在本地跑

```bash
make test-notebooks NOTEBOOK=third_party/ElevenLabs/low_latency_stt_claude_tts.ipynb
```
