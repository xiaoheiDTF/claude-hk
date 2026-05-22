# Cookbook / Tool-Use

> 来源: claudecn.com

# 工具调用：从 Tool Choice 到 PTC

Claude Cookbooks 的 `tool_use/` 目录更像一组“可复用的工程配方”：把工具定义、调用循环、以及生产里的成本/延迟/安全问题放在同一个视角下讲清楚。

## 你将学到什么

- Tool Use 的基础接口形态：tools 定义、tool_use/tool_result 回传、以及多轮循环
- 如何让模型在 多个工具里做选择（tool choice），以及如何减少“瞎用工具”
- 并行工具调用、结构化抽取、以及常见的“agentic workflow”骨架
- 进阶：PTC（Programmatic Tool Calling）、Tool Search（海量工具检索）、自动上下文压缩
## 心智模型：把 tool loop 当成一套协议

多数 notebook 都遵循同一个骨架：

- 你定义 tools（name/description + 类 JSON Schema 的 input_schema）。
- 调用 Claude：tools=... + 一个 tool_choice。
- Claude 要么直接输出文本，要么请求调用工具（stop_reason == "tool_use"）。
- 你的应用执行工具，并把结果作为 tool_result 回传（必须带 tool_use_id）。
- 重复直到 stop_reason == "end_turn"。
## 最小工具定义长什么样

下面这种 JSON Schema 风格的 `input_schema`，几乎贯穿所有 tool_use 示例：

```python
tools = [
  {
    "name": "print_sentiment_scores",
    "description": "Print sentiment scores of a given text.",
    "input_schema": {"type": "object", "properties": {"positive_score": {"type": "number"}}},
  }
]
```

## 回传结果：tool_result 必须引用 tool_use_id
当 Claude 输出 `tool_use` block 后，你需要回传一个对应的 `tool_result`，并用 `tool_use_id` 关联起来。

这个写法在并行工具调用 notebook 中是明确展示的：

```python
MESSAGES.append({"role": "assistant", "content": response.content})
MESSAGES.append(
  {"role": "user", "content": [{"type": "tool_result", "tool_use_id": last_tool_call.id, "content": result}]}
)
```

## 推荐 Notebook（按“从基础到进阶”）

### 1) 入门：先把 tool loop 跑起来
[Calculator tool最小可运行的 tool loop 示例
](calculator-tool/)[Customer service agent客户端工具 + 模拟 tool results
](customer-service-agent/)[Extracting structured JSON用 tools 强化结构化输出
](extracting-structured-json/)

### 2) 进阶：让“用工具”更可控
[Tool choiceAuto / Any / 强制 tool
](tool-choice/)[Parallel tool calls多工具并行 + batch tool 模式
](parallel-tools/)

#### Tool choice（Auto / Any / 强制某个 tool）
在 `tool_use/tool_choice.ipynb` 里，“自动选择”示例写法是：

```python
tool_choice={"type": "auto"}
```

生产落地时，关键不只是参数，而是提示词契约：明确告诉模型何时应当用工具、何时应当直接回答，以及不确定时的兜底策略。

#### 并行工具调用 + “batch tool”思路

`tool_use/parallel_tools.ipynb` 展示了两类实用模式：

- 允许模型在一次响应里发起多个工具调用（你的应用遍历 response.content，逐个处理 tool_use）。
- 引入一个 “batch tool”，把多个工具调用打包成一次（减少回合数，降低端到端延迟）。
### 3) 工程化：更低延迟、更少 token、更长对话
[Programmatic tool calling (PTC)code execution + allowed_callers
](ptc-programmatic-tool-calling/)[Tool search with embeddings工具规模上千时的检索式选择
](tool-search-with-embeddings/)[Automatic context compactiontool_runner + compaction_control
](automatic-context-compaction/)[Session memory compaction后台记忆 + prompt caching
](session-memory-compaction/)[Memory & context management长对话/长流程的记忆与编辑
](memory-and-context-management/)

#### PTC（Programmatic Tool Calling）
PTC 可以理解为“让代码来调用工具”。在 PTC notebook 里：

- 传统 tool loop 使用 beta 接口，并显式带上 betas=[...]。
- PTC 方案会加入 code execution tool，并限制“哪些工具允许被 code execution 调用”：
```python
tool["allowed_callers"] = ["code_execution_20250825"]
ptc_tools.append({"type": "code_execution_20250825", "name": "code_execution"})
```

它适合放在你已经有安全的代码执行沙箱、且希望把“多步工具编排”交给代码来做的场景。

#### Tool Search with embeddings（工具规模上千时）

该 notebook 会构建 tool library，用 SentenceTransformers 做 tool embeddings，再用语义相似度先筛选工具，再进入 tool loop。

注意：示例使用 `sentence-transformers/all-MiniLM-L6-v2`，首次运行会下载模型。

#### Automatic Context Compaction（长对话/长工单不崩）

`tool_use/automatic-context-compaction.ipynb` 使用 beta 的 `tool_runner`，并通过：

```python
compaction_control={"enabled": True, "context_token_threshold": 5000}
```

来自动压缩历史消息、插入摘要 message，减少你手写“截断策略”的复杂度。

### 4) 视觉 + 工具（多模态落地）
[Using vision with toolsbase64 图片输入 + 工具抽取结构化字段
](vision-with-tools/)
在 `vision_with_tools.ipynb` 里，图片以 base64 的形式作为 message content 传入，然后用工具抽取结构化字段（营养成分表）。

## 生产化提示（建议先对齐团队共识）

- 工具要可审计：工具入参/出参、失败重试、以及 side effects（写库/发消息）需要日志与权限约束。
- 把成本写进架构：并行、缓存、PTC、压缩上下文，本质都在换取更可控的延迟与 token。
- 先用结构测试兜底：改 notebook/示例时，优先 make test-notebooks 把“可复现”守住。
