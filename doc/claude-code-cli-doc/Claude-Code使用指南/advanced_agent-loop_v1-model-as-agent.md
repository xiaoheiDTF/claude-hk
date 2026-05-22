# Claude-Code / Advanced / Agent-Loop / V1-Model-As-Agent

> 来源: claudecn.com

# v1：模型即代理

Claude Code 的秘密？**没有秘密。**

剥去 CLI 外观、进度条、权限系统，剩下的出奇简单：一个让模型持续调用工具直到任务完成的循环。

## 核心洞察

传统助手：

```
用户 -> 模型 -> 文本回复 → Agent 系统： → 用户 -> 模型 -> [工具 -> 结果]* -> 回复
                   ^_________| → 星号很重要。模型 → 反复 → 调用工具，直到它决定任务完成。这将聊天机器人转变为自主代理。 → 需要工具 → 任务完成 → 用户 → 模型 → 工具调用 → 执行结果 → 回复
```

**核心洞察**：模型是决策者，代码只提供工具并运行循环。

## 四个核心工具

Claude Code 有约 20 个工具，但 4 个覆盖 90% 的场景：

| 工具 | 用途 | 示例 |
| --- | --- | --- |
| `bash` | 运行命令 | `npm install`, `git status` |
| `read_file` | 读取内容 | 查看 `src/index.ts` |
| `write_file` | 创建/覆盖 | 创建 `README.md` |
| `edit_file` | 精确修改 | 替换一个函数 |

有了这 4 个工具，模型可以：

- 探索代码库（bash: find, grep, ls）
- 理解代码（read_file）
- 做出修改（write_file, edit_file）
- 运行任何东西（bash: python, npm, make）
## 完整代码

```
""" → v1_basic_agent.py - Mini Claude Code: Model as Agent (~200 lines) → Core Philosophy: "The Model IS the Agent" → ========================================= → Strip away the CLI polish, progress bars, permission systems. What remains → is surprisingly simple: a LOOP that lets the model call tools until done. → Traditional Assistant: → User -> Model -> Text Response → Agent System: → User -> Model -> [Tool -> Result]* -> Response → ^________| → The asterisk (*) matters! The model calls tools REPEATEDLY until it decides → the task is complete. This transforms a chatbot into an autonomous agent. → KEY INSIGHT: The model is the decision-maker. Code just provides tools and → runs the loop. The model decides: → - Which tools to call → - In what order → - When to stop → The Four Essential Tools: → ------------------------ → Claude Code has ~20 tools. But these 4 cover 90 → % o → f use cases: → | Tool       | Purpose              | Example                    | → |------------|----------------------|----------------------------| → | bash       | Run any command      | npm install, git status    | → | read_file  | Read file contents   | View src/index.ts          | → | write_file | Create/overwrite     | Create README.md           | → | edit_file  | Surgical changes     | Replace a function         | → With just these 4 tools, the model can: → - Explore codebases (bash: find, grep, ls) → - Understand code (read_file) → - Make changes (write_file, edit_file) → - Run anything (bash: python, npm, make) → Usage: → python v1_basic_agent.py → """ → import → os → import → subprocess → import → sys → from → pathlib → import → Path → from → anthropic → import → Anthropic → from → dotenv → import → load_dotenv → load_dotenv → override → True → WORKDIR → Path → cwd → () → MODEL → os → getenv → "MODEL_ID" → "claude-sonnet-4-5-20250929" → client → Anthropic → base_url → os → getenv → "ANTHROPIC_BASE_URL" → )) → SYSTEM → """You are a coding agent at → WORKDIR → Loop: think briefly -> use tools -> report results. → Rules: → - Prefer tools over prose. Act, don't just explain. → - Never invent file paths. Use bash ls/find first if unsure. → - Make minimal changes. Don't over-engineer. → - After finishing, summarize what changed.""" → TOOLS → "name" → "bash" → "description" → "Run a shell command. Use for: ls, find, grep, git, npm, python, etc." → "input_schema" → "type" → "object" → "properties" → "command" → "type" → "string" → "description" → "The shell command to execute" → "required" → "command" → ], → "name" → "read_file" → "description" → "Read file contents. Returns UTF-8 text." → "input_schema" → "type" → "object" → "properties" → "path" → "type" → "string" → "description" → "Relative path to the file" → "limit" → "type" → "integer" → "description" → "Max lines to read (default: all)" → "required" → "path" → ], → "name" → "write_file" → "description" → "Write content to a file. Creates parent directories if needed." → "input_schema" → "type" → "object" → "properties" → "path" → "type" → "string" → "description" → "Relative path for the file" → "content" → "type" → "string" → "description" → "Content to write" → "required" → "path" → "content" → ], → "name" → "edit_file" → "description" → "Replace exact text in a file. Use for surgical edits." → "input_schema" → "type" → "object" → "properties" → "path" → "type" → "string" → "description" → "Relative path to the file" → "old_text" → "type" → "string" → "description" → "Exact text to find (must match precisely)" → "new_text" → "type" → "string" → "description" → "Replacement text" → "required" → "path" → "old_text" → "new_text" → ], → def → safe_path → str → -> → Path → """ → Ensure path stays within workspace (security measure). → Prevents the model from accessing files outside the project directory. → Resolves relative paths and checks they don't escape via '../'. → """ → path → WORKDIR → resolve → () → if → not → path → is_relative_to → WORKDIR → ): → raise → ValueError → "Path escapes workspace: → return → path → def → run_bash → command → str → -> → str → """ → Execute shell command with safety checks. → Security: Blocks obviously dangerous commands. → Timeout: 60 seconds to prevent hanging. → Output: Truncated to 50KB to prevent context overflow. → """ → dangerous → "rm -rf /" → "sudo" → "shutdown" → "reboot" → "> /dev/" → if → any → in → command → for → in → dangerous → ): → return → "Error: Dangerous command blocked" → try → result → subprocess → run → command → shell → True → cwd → WORKDIR → capture_output → True → text → True → timeout → 60 → output → result → stdout → result → stderr → strip → () → return → output → [: → 50000 → if → output → else → "(no output)" → except → subprocess → TimeoutExpired → return → "Error: Command timed out (60s)" → except → Exception → as → return → "Error: → def → run_read → path → str → limit → int → None → -> → str → """ → Read file contents with optional line limit. → For large files, use limit to read just the first N lines. → Output truncated to 50KB to prevent context overflow. → """ → try → text → safe_path → path → read_text → () → lines → text → splitlines → () → if → limit → and → limit → < → len → lines → ): → lines → lines → [: → limit → lines → append → "... ( → len → text → splitlines → ()) → limit → more lines)" → return → \n → join → lines → )[: → 50000 → except → Exception → as → return → "Error: → def → run_write → path → str → content → str → -> → str → """ → Write content to file, creating parent directories if needed. → This is for complete file creation/overwrite. → For partial edits, use edit_file instead. → """ → try → fp → safe_path → path → fp → parent → mkdir → parents → True → exist_ok → True → fp → write_text → content → return → "Wrote → len → content → bytes to → path → except → Exception → as → return → "Error: → def → run_edit → path → str → old_text → str → new_text → str → -> → str → """ → Replace exact text in a file (surgical edit). → Uses exact string matching - the old_text must appear verbatim. → Only replaces the first occurrence to prevent accidental mass changes. → """ → try → fp → safe_path → path → content → fp → read_text → () → if → old_text → not → in → content → return → "Error: Text not found in → path → new_content → content → replace → old_text → new_text → fp → write_text → new_content → return → "Edited → path → except → Exception → as → return → "Error: → def → execute_tool → name → str → args → dict → -> → str → """ → Dispatch tool call to the appropriate implementation. → This is the bridge between the model's tool calls and actual execution. → Each tool returns a string result that goes back to the model. → """ → if → name → == → "bash" → return → run_bash → args → "command" → ]) → if → name → == → "read_file" → return → run_read → args → "path" → ], → args → get → "limit" → )) → if → name → == → "write_file" → return → run_write → args → "path" → ], → args → "content" → ]) → if → name → == → "edit_file" → return → run_edit → args → "path" → ], → args → "old_text" → ], → args → "new_text" → ]) → return → "Unknown tool: → name → def → agent_loop → messages → list → -> → list → """ → The complete agent in one function. → This is the pattern that ALL coding agents share: → while True: → response = model(messages, tools) → if no tool calls: return → execute tools, append results, continue → The model controls the loop: → - Keeps calling tools until stop_reason != "tool_use" → - Results become context (fed back as "user" messages) → - Memory is automatic (messages list accumulates history) → Why this works: → 1. Model decides which tools, in what order, when to stop → 2. Tool results provide feedback for next decision → 3. Conversation history maintains context across turns → """ → while → True → response → client → messages → create → model → MODEL → system → SYSTEM → messages → messages → tools → TOOLS → max_tokens → 8000 → tool_calls → [] → for → block → in → response → content → if → hasattr → block → "text" → ): → print → block → text → if → block → type → == → "tool_use" → tool_calls → append → block → if → response → stop_reason → != → "tool_use" → messages → append → ({ → "role" → "assistant" → "content" → response → content → return → messages → results → [] → for → tc → in → tool_calls → print → \n → > → tc → name → tc → input → output → execute_tool → tc → name → tc → input → preview → output → [: → 200 → "..." → if → len → output → > → 200 → else → output → print → " → preview → results → append → ({ → "type" → "tool_result" → "tool_use_id" → tc → id → "content" → output → messages → append → ({ → "role" → "assistant" → "content" → response → content → messages → append → ({ → "role" → "user" → "content" → results → def → main → (): → """ → Simple Read-Eval-Print Loop for interactive use. → The history list maintains conversation context across turns, → allowing multi-turn conversations with memory. → """ → print → "Mini Claude Code v1 - → WORKDIR → print → "Type 'exit' to quit. → \n → history → [] → while → True → try → user_input → input → "You: " → strip → () → except → EOFError → KeyboardInterrupt → ): → break → if → not → user_input → or → user_input → lower → () → in → "exit" → "quit" → "q" → ): → break → history → append → ({ → "role" → "user" → "content" → user_input → try → agent_loop → history → except → Exception → as → print → "Error: → print → () → if → __name__ → == → "__main__" → main → () → Agent 循环 → 整个 Agent 在一个函数里： → def → agent_loop → messages → ): → while → True → response → client → messages → create → model → MODEL → system → SYSTEM → messages → messages → tools → TOOLS → for → block → in → response → content → if → hasattr → block → "text" → ): → print → block → text → if → response → stop_reason → != → "tool_use" → return → messages → results → [] → for → tc → in → response → tool_calls → output → execute_tool → tc → name → tc → input → results → append → ({ → "type" → "tool_result" → "tool_use_id" → tc → id → "content" → output → messages → append → ({ → "role" → "assistant" → "content" → response → content → messages → append → ({ → "role" → "user" → "content" → results → 为什么这能工作： → 模型控制循环（持续调用工具直到 → stop_reason != "tool_use" → 结果成为上下文（作为 “user” 消息反馈） → 记忆自动累积（messages 列表保存历史） → 开始: 用户消息 → 调用模型 API → 有工具调用? → 执行工具 → 收集结果 → 追加到对话历史 → 返回文本回复 → 结束
```

## 系统提示词

唯一需要的"配置"：

```python
SYSTEM = f"""You are a coding agent at {WORKDIR}.

Loop: think briefly -> use tools -> report results.

Rules:
- Prefer tools over prose. Act, don't just explain.
- Never invent file paths. Use ls/find first if unsure.
- Make minimal changes. Don't over-engineer.
- After finishing, summarize what changed."""
```

没有复杂逻辑，只有清晰的指令。

## 为什么这个设计有效

**1. 简单**
没有状态机，没有规划模块，没有框架。

**2. 模型负责思考**
模型决定用哪些工具、什么顺序、何时停止。

**3. 透明**
每个工具调用可见，每个结果在对话中。

**4. 可扩展**
添加工具 = 一个函数 + 一个 JSON schema。

## 缺少什么

| 特性 | 为什么省略 | 添加于 |
| --- | --- | --- |
| 待办追踪 | 非必需 | v2 |
| 子代理 | 复杂度 | v3 |
| 权限 | 学习目的信任模型 | 生产版 |

关键点：**核心是微小的**，其他都是精化。

## 更大的图景

这一代 agent coding 系统都共享这个模式：

```python
while not done:
    response = model(conversation, tools)
    results = execute(response.tool_calls)
    conversation.append(results)
```

差异在于工具、显示、安全性。但本质始终是：**给模型工具，让它工作**。

## 运行示例

```bash
# 安装依赖
pip install anthropic python-dotenv

# 配置环境变量
echo "ANTHROPIC_API_KEY=sk-ant-xxx" > .env

# 运行
python v1_basic_agent.py

# 示例对话
You: 列出当前目录的 Python 文件
> bash: {'command': 'ls *.py'}
  (no output)

You: 用 find 再试一次
> bash: {'command': 'find . -name "*.py"'}
  ./v0_bash_agent.py
  ./v1_basic_agent.py
```

---
**模型即代理。这就是全部秘密。**
[v0：Bash 就是一切一个工具 + 循环，完整的 Agent 能力
](../v0-bash-agent/)[v2：显式 Todo把计划做成可见状态机，减少跑偏
](../v2-explicit-planning-todo/)
