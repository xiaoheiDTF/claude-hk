# Source-Analysis / Voice

> 来源: claudecn.com

# Voice 语音

Push-to-talk 语音输入——Claude Code 通过原生音频模块、STT 流和 OAuth 闸门构建了一套从麦克风到模型输入的完整语音通道。

## 核心问题

让 CLI 工具支持语音输入，面临的不是"怎么录音"，而是"怎么在不同平台上可靠地获取音频流、实时转写、并与现有权限体系对接"。

## 子系统全景

| 指标 | 数值 |
| --- | --- |
| **核心服务文件** | `voice.ts`（526 行） |
| **成熟度** | Emerging（正在浮现） |
| **闸门控制** | GrowthBook (`tengu_amber_quartz_disabled`) + Anthropic OAuth |
| **原生依赖** | `vendor/audio-capture/` — cpal 原生模块 |

## 架构分层

| 组件 | 文件 | 职责 |
| --- | --- | --- |
| **录音服务** | `src/services/voice.ts`（526 行） | 封装 cpal 原生模块（macOS/Linux/Windows），回退到 SoX `rec` / ALSA `arecord` |
| **STT 流** | `src/services/voiceStreamSTT.ts` | 连接 `claude.ai` 的 `voice_stream` 端点，实时语音转文本 |
| **关键词** | `src/services/voiceKeyterms.ts` | 语音命令关键词匹配 |
| **闸门** | `src/voice/voiceModeEnabled.ts` | GrowthBook + OAuth 双重检查 |
| **命令入口** | `src/commands/voice/` | `/voice` 命令注册 |
| **UI 集成** | `useVoice.ts`、`useVoiceEnabled.ts`、`useVoiceIntegration.tsx` | React 侧状态与渲染 |

```
语音转文本 → 音频捕获 → 闸门层 → 交互层 → useVoice.ts → Push-to-talk → /voice 命令 → GrowthBook → tengu_amber_quartz_disabled → Anthropic OAuth → 强绑定 → cpal 原生模块 → audio-capture.node → 回退方案 → SoX rec / ALSA arecord → voiceStreamSTT.ts → voice_stream 端点 → voiceKeyterms.ts → 关键词匹配
```

## 关键设计决策

### 延迟加载原生模块

`audio-capture.node` 链接 CoreAudio.framework，`dlopen` 同步阻塞可达 ~8 秒（冷启动）。因此延迟到首次语音按键才加载，而不是预加载。这是一个典型的**启动时间 vs 首次使用延迟**的权衡：选择牺牲首次使用时的响应，换取所有非语音场景的零额外开销。

### OAuth 强绑定

`voice_stream` 端点仅通过 `claude.ai` 提供。这意味着 API Key、Bedrock、Vertex、Foundry 用户不可用。语音功能与 Anthropic 自有认证体系强绑定——这可能是出于 STT 服务的成本和合规考虑。

### 跨平台音频方案

| 平台 | 首选方案 | 回退方案 |
| --- | --- | --- |
| **macOS** | cpal 原生模块（CoreAudio） | SoX `rec` |
| **Linux** | cpal 原生模块（ALSA/PulseAudio） | ALSA `arecord` |
| **Windows** | cpal 原生模块（WASAPI） | — |

回退方案确保即使原生模块加载失败（权限问题、依赖缺失），语音功能仍然可用——只是音频质量可能略有降级。

### Push-to-talk 模式

语音输入采用 Push-to-talk（按住录音、松开发送），而非持续监听。静音检测阈值为 2.0 秒 / 3%。这个设计避免了"持续监听"带来的隐私和资源消耗问题。

## 闸门机制

Voice Surface 的闸门是三重的：

- GrowthBook 远程开关：tengu_amber_quartz_disabled 可以在服务端随时关闭
- OAuth 检查：只有 Anthropic OAuth 登录的用户才能使用
- Feature Flag：VOICE_MODE（46 处引用）门控语音相关代码路径
任一层检查失败，语音功能就不会出现在 UI 和命令列表中。

## 对 agent 开发者的启示

| 模式 | 说明 |
| --- | --- |
| **分层回退** | 原生模块 → 系统工具 → 降级提示，每层都有明确的回退路径 |
| **延迟加载重资源** | 不要在启动时加载可能 block 8 秒的原生依赖 |
| **认证绑定** | 高成本能力（STT）与认证体系绑定，控制服务端资源消耗 |
| **Push-to-talk** | 比持续监听更节能、更隐私，且用户控制感更强 |

## 路径证据

| 路径 | 职责 |
| --- | --- |
| `src/services/voice.ts` | 录音服务核心（526 行） |
| `src/services/voiceStreamSTT.ts` | 语音转文本流 |
| `src/services/voiceKeyterms.ts` | 关键词匹配 |
| `src/voice/voiceModeEnabled.ts` | 闸门逻辑 |
| `src/commands/voice/` | 命令入口 |
| `src/hooks/useVoice.ts` | React Hook |
| `vendor/audio-capture/` | 音频捕获原生二进制 |

## 进一步阅读

- 扩展与信号 — Voice 的成熟度判定
- 架构地图 — 语音在六层结构中的位置
- Computer Use — 另一个依赖原生模块的能力面
