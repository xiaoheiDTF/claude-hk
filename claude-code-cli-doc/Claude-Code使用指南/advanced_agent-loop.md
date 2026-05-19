# Claude-Code / Advanced / Agent-Loop

> 来源: claudecn.com

# Claude Code 背后的 Agent Loop（从零理解）

如果你把 Claude Code 当成"会写代码的 CLI"，很多能力会显得像魔法：它能读文件、跑命令、改代码、拆任务、并在复杂工作里保持不跑偏。

但从工程视角看，它的核心其实很朴素：**模型 + 工具 + 一个循环（Agent Loop）**。理解这个循环，你就能更清楚地：

- 什么时候该让 Claude 先"规划"，什么时候该直接"动手"
- 为什么"显式任务清单（Todo）“能显著降低跑偏
- 为什么"子代理（Subagents）“能提升探索效率与上下文质量
- Skills/MCP/Hook 这些能力在系统里分别扮演什么角色
这里把"原理"讲清楚，并给出一条可执行的练习路径：每一章只引入一个关键机制，便于你建立直觉并能复现验证。

## 适用范围

适合你满足以下任意一条：

- 已经能用 Claude Code 完成日常开发，但想理解它为什么"像一个 Agent”
- 需要把 Claude API/Agent SDK 集成进自己的产品，想先抓住最核心的控制流
- 想把团队里的最佳实践（规则、检查、知识）沉淀成可复用的约束
## 背景与概念：什么是 Agent Loop

Agent Loop 的本质是一个"工具驱动的对话循环”：

- 你的应用把上下文（messages）和工具定义（tools）发给模型
- 模型要么输出文本，要么请求调用某个工具（tool_use）
- 你的应用执行工具，把结果回传给模型（tool_result）
- 重复，直到模型结束（end_turn）
```
while → True → response → model → messages → tools → if → response → stop_reason → != → "tool_use" → return → response → text → results → execute → response → tool_calls → messages → append → results → 你可以把 Claude Code 里"读文件/搜索/编辑/跑测试/提交 Git"等行为，都理解为： → 模型在循环里不断申请工具调用 → 学习路径：v0→v4（每章一个关键机制） → v0（Bash Agent） → ：只有 1 个工具（bash），但能做完整的"读/写/搜/执行"，甚至可以用"递归自调用"实现子代理 → v1（Basic Agent） → ：加入更清晰的读/写/编辑工具，结构更接近常见的生产实现 → v2（Todo Agent） → ：加入 TodoWrite，让计划可见、可约束、可追踪 → v3（Subagent） → ：引入 Task/子代理，解决"探索污染主上下文"的问题 → v4（Skills Agent） → ：引入按需加载的 Skills，把领域知识从主提示词里解耦出来 → 这条路径的价值是：每一步只新增一个"关键机制"，你能明确知道"能力从哪里来"，而不是一次性吞一套复杂框架。 → 学习路径图 → 开始 → v0: Bash Agent → ~50 行 → 一个工具足够 → v1: 模型即代理 → ~200 行 → 核心循环 → v2: 显式 Todo → ~300 行 → 减少跑偏 → v3: 子代理 → ~450 行 → 任务分解 → v4: Skills → ~550 行 → 知识外化
```

### 版本对比表

| 版本 | 主题 | 新增行数 | 核心工具 | 关键洞察 |
| --- | --- | --- | --- | --- |
| v0 | Bash 就是一切 | ~50 | bash (1个) | 一个工具足够，递归 = 层级 |
| v1 | 模型即代理 | ~200 | bash + read + write + edit | 模型是决策者，代码只运行循环 |
| v2 | 显式 Todo | ~300 | + TodoWrite | 计划显式化：更少跑偏 |
| v3 | 子代理机制 | ~450 | + Task | 上下文隔离：探索不污染主会话 |
| v4 | Skills 机制 | ~550 | + Skill | 知识外化：从训练到编辑 |

### 章节速查（建议顺序）

| 章节 | 你要抓住的关键点 | 对应到 Claude Code |
| --- | --- | --- |
| v0 | 只靠 `bash` + tool loop 也能跑通"读/写/搜/执行" | 你看到的"会写代码的 CLI"本质就是工具循环 |
| v1 | 把工具拆清楚：Read/Write/Edit（更接近工程实现） | 为什么 Claude Code 的文件操作更稳定 |
| v2 | TodoWrite + 约束（只允许 1 个 in_progress） | 计划显式化：更少跑偏、更好协作 |
| v3 | 子代理隔离上下文（探索不污染主会话） | Subagents（Explore/Plan 等）的价值 |
| v4 | Skills 按需加载，把知识从主提示词里解耦 | Skills 让"规范/领域知识"可复用、可维护 |

## 开始学习
[v0：Bash 就是一切一个工具 + 循环，读/写/搜/执行
](v0-bash-agent/)[v1：模型即代理拆清 Read/Write/Edit，让行为更稳定
](v1-model-as-agent/)[v2：显式 Todo把计划做成可见状态机，减少跑偏
](v2-explicit-planning-todo/)[v3：子代理机制分而治之，上下文隔离
](v3-subagents/)[v4：Skills 机制知识外化：从训练到编辑的范式转变
](v4-skills/)

## 动手练习：把原理跑通（可验证）

下面的练习不依赖特定仓库：你可以在任意空目录里，用你熟悉的语言实现一个"最小可运行"的工具循环。

### 定义最小工具

建议从这两个开始：

- read_file(path)：读取文件并返回内容
- bash(command)：执行只读类命令（例如 ls、rg）
### 写一个最小循环（伪代码）

```python
messages = [{"role": "user", "content": "在当前目录找 README 并总结要点"}]
tools = [read_file, bash]

while True:
    resp = model(messages, tools)
    if resp.stop_reason != "tool_use":
        print(resp.text)
        break
    results = execute_tools(resp.tool_calls)
    messages.append({"role": "user", "content": results})
```

### 用 3 类问题验证循环确实工作

- 让它列出当前目录并解释结构（读/搜）
- 让它创建一个文件并写入内容（写）
- 让它运行一个简单命令并解释输出（执行）
## 常见坑与排障

- 卡在安装/运行：先确认 Python 版本与依赖已安装；再检查 .env 是否生效（API Key 是否存在）。
- 成本与速率：长时间探索/反复 tool loop 会增加调用次数；先用小任务验证控制流，再扩展复杂任务。
## 进一步阅读（相关链接）

- Claude Code 工作流：/docs/claude-code/workflows/
- Subagents：/docs/claude-code/advanced/subagents
- Skills：/docs/claude-code/advanced/skills
