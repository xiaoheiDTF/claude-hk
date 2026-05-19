# Claude-Code / Workflows / Continuous-Learning

> 来源: claudecn.com

# 持续学习：把会话复盘沉淀成 Skills

很多“真正有价值的经验”不会出现在最终代码里，而是出现在过程里：你怎么定位 bug、怎么拆解需求、怎么绕开某个坑。一个实用的做法是：在会话结束时触发一次轻量复盘，把可复用的模式沉淀到本地 Skills（例如 `~/.claude/skills/learned/`）。

这不是 Claude Code 的默认能力，而是一种“用 Hooks 驱动复盘”的工程化做法。

## 1) 为什么用 Stop Hook（而不是每条消息都分析）

社区脚本的理由很务实：

- Stop 只在会话结束触发一次，开销低
- 如果每条消息都做分析，会增加延迟并污染上下文
## 2) 最小实现：满足门槛才提示“可提炼”

示例脚本会读取 `CLAUDE_TRANSCRIPT_PATH` 指向的 transcript，并用最小门槛过滤“短会话”（例如少于 10 条 user 消息就跳过）：

```bash
transcript_path="${CLAUDE_TRANSCRIPT_PATH:-}"
message_count=$(grep -c '\"type\":\"user\"' \"$transcript_path\" 2>/dev/null || echo \"0\")
```

在达到门槛后，它只输出提示信息（节选）：

```text
[ContinuousLearning] Session has N messages - evaluate for extractable patterns
[ContinuousLearning] Save learned skills to: ~/.claude/skills/learned
```

这类设计的关键点是：**先提醒人复盘**，而不是自动生成一堆未经审查的“规则”。

## 3) 配置建议：把“提炼范围”写进 config

示例 `config.json` 提供了两类列表：

- patterns_to_detect：例如 error_resolution、workarounds、debugging_techniques、project_specific
- ignore_patterns：例如 simple_typos、one_time_fixes
这能帮你避免把“偶发修补”误沉淀成长期规则。

## 4) 安全与隐私注意事项（团队落地必看）

会话 transcript 往往包含：

- 代码片段
- 日志与错误栈
- 环境变量名甚至密钥（如果有人贴了）
因此建议：

- learned skills 目录不要纳入仓库提交（除非明确筛选与脱敏）
- 复盘前先做“敏感信息检查”（参考 安全指南）
- 只沉淀“可迁移的方法”，不要沉淀“项目私密细节”
## 5) 如何把“提炼”变成团队习惯

一个推荐节奏：

- 每个任务结束：先写 3 行 handoff（见 会话连续性与战略压缩）
- 每周一次：集中挑 2–3 个可复用模式，整理成正式 Skill / Rule / Command
- 大项目：把 codemaps 与 docs sync 纳入例行维护（见 文档同步与 Codemaps）
## 参考

无
