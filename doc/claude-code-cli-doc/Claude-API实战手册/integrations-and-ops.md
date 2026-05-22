# Cookbook / Integrations-And-Ops

> 来源: claudecn.com

# 集成与运维：第三方、微调与用量成本

Cookbooks 里一部分内容专门围绕"怎么把 Claude 放进你的系统里"：**第三方生态集成**、**云平台能力**（例如 Bedrock）、以及**用量/成本的可观测性**。

## 概述

在生产环境中构建 Claude 应用需要的不只是 API 调用：

- 集成服务：连接数据库（Pinecone、MongoDB）、搜索服务（Wikipedia）和专业工具（Wolfram Alpha、Deepgram）
- 云平台：在 AWS Bedrock 上部署并使用微调能力
- 运维监控：追踪用量、监控成本，为生产工作负载设置可观测性
## 第三方集成（third_party）

优先从你实际在用的组件入手：
[Pinecone RAG基于 Pinecone 的 RAG 示例
](pinecone-rag/)[MongoDB RAG基于 MongoDB 的 RAG 示例
](mongodb-rag/)[LlamaIndex basic RAG用 LlamaIndex 搭 RAG pipeline
](llamaindex-basic-rag/)[Wikipedia search迭代式检索 Wikipedia
](wikipedia-search/)[Wolfram Alpha tool把 WolframAlpha 当作工具
](wolframalpha-tool/)[ElevenLabs 语音助手低延迟语音助手（STT/TTS）
](elevenlabs-voice-assistant/)[Deepgram 音频转写音频转写 + 生成采访问题
](deepgram-audio-transcription/)

## 微调（Fine-Tuning）
[Finetuning on Bedrock在 Amazon Bedrock 上微调 Claude 3 Haiku
](finetuning-on-bedrock/)

微调与数据集相关的示例往往涉及敏感数据：在把数据上传到云端前，先对齐合规、脱敏、权限与留存策略。

## 用量与成本（Observability）
[Usage & cost Admin API用量/成本追踪脚手架
](usage-cost-admin-api/)
这份 notebook 适合用来搭“最小可用”的成本追踪脚手架：先把分组、过滤、分页、导出跑通，再接入你自己的监控/告警体系。
