# Claude-Code / Advanced / Agent-Loop / V0-Bash-Agent

> 来源: claudecn.com

# v0：Bash 就是一切

在构建 v1、v2、v3 之后，一个问题浮现：Agent 的**本质**到底是什么？

v0 通过反向思考来回答——剥离一切，直到只剩下核心。

## 核心洞察

Unix 哲学：一切皆文件，一切皆可管道。Bash 是这个世界的入口：

| 你需要 | Bash 命令 |
| --- | --- |
| 读文件 | `cat`, `head`, `grep` |
| 写文件 | `echo '...' > file` |
| 搜索 | `find`, `grep`, `rg` |
| 执行 | `python`, `npm`, `make` |
| **子代理** | `python v0_bash_agent.py "task"` |

最后一行是关键洞察：**通过 bash 调用自身就实现了子代理**。不需要 Task 工具，不需要 Agent Registry——只需要递归。

## 完整代码

```python
#!/usr/bin/env python3
"""
v0_bash_agent.py - Mini Claude Code: Bash is All You Need (~50 lines)

Core Philosophy: "Bash is All You Need"
======================================
This is the ULTIMATE simplification of a coding agent. After building v1-v3,
we ask: what is the ESSENCE of an agent?

The answer: ONE tool (bash) + ONE loop = FULL agent capability.

Usage:
    # Interactive mode
    python v0_bash_agent.py

    # Subagent mode (called by parent agent or directly)
    python v0_bash_agent.py "explore src/ and summarize"
"""

from anthropic import Anthropic
from dotenv import load_dotenv
import subprocess
import sys
import os

load_dotenv(override=True)

# Initialize Anthropic client (uses ANTHROPIC_API_KEY and ANTHROPIC_BASE_URL env vars)
client = Anthropic(base_url=os.getenv("ANTHROPIC_BASE_URL"))
MODEL = os.getenv("MODEL_ID", "claude-sonnet-4-5-20250929")

# The ONE tool that does everything
# Notice how the description teaches the model common patterns AND how to spawn subagents
TOOL = [{
    "name": "bash",
    "description": """Execute shell command. Common patterns:
- Read: cat/head/tail, grep/find/rg/ls, wc -l
- Write: echo 'content' > file, sed -i 's/old/new/g' file
- Subagent: python v0_bash_agent.py 'task description' (spawns isolated agent, returns summary)""",
    "input_schema": {
        "type": "object",
        "properties": {"command": {"type": "string"}},
        "required": ["command"]
    }
}]

# System prompt teaches the model HOW to use bash effectively
# Notice the subagent guidance - this is how we get hierarchical task decomposition
SYSTEM = f"""You are a CLI agent at {os.getcwd()}. Solve problems using bash commands.

Rules:
- Prefer tools over prose. Act first, explain briefly after.
- Read files: cat, grep, find, rg, ls, head, tail
- Write files: echo '...' > file, sed -i, or cat << 'EOF' > file
- Subagent: For complex subtasks, spawn a subagent to keep context clean:
  python v0_bash_agent.py "explore src/ and summarize the architecture"

When to use subagent:
- Task requires reading many files (isolate the exploration)
- Task is independent and self-contained
- You want to avoid polluting current conversation with intermediate details

The subagent runs in isolation and returns only its final summary."""

def chat(prompt, history=None):
    """
    The complete agent loop in ONE function.

    This is the core pattern that ALL coding agents share:
        while not done:
            response = model(messages, tools)
            if no tool calls: return
            execute tools, append results

    Args:
        prompt: User's request
        history: Conversation history (mutable, shared across calls in interactive mode)

    Returns:
        Final text response from the model
    """
    if history is None:
        history = []

    history.append({"role": "user", "content": prompt})

    while True:
        # 1. Call the model with tools
        response = client.messages.create(
            model=MODEL,
            system=SYSTEM,
            messages=history,
            tools=TOOL,
            max_tokens=8000
        )

        # 2. Build assistant message content (preserve both text and tool_use blocks)
        content = []
        for block in response.content:
            if hasattr(block, "text"):
                content.append({"type": "text", "text": block.text})
            elif block.type == "tool_use":
                content.append({
                    "type": "tool_use",
                    "id": block.id,
                    "name": block.name,
                    "input": block.input
                })
        history.append({"role": "assistant", "content": content})

        # 3. If model didn't call tools, we're done
        if response.stop_reason != "tool_use":
            return "".join(b.text for b in response.content if hasattr(b, "text"))

        # 4. Execute each tool call and collect results
        results = []
        for block in response.content:
            if block.type == "tool_use":
                cmd = block.input["command"]
                print(f"\033[33m$ {cmd}\033[0m")  # Yellow color for commands

                try:
                    out = subprocess.run(
                        cmd,
                        shell=True,
                        capture_output=True,
                        text=True,
                        timeout=300,
                        cwd=os.getcwd()
                    )
                    output = out.stdout + out.stderr
                except subprocess.TimeoutExpired:
                    output = "(timeout after 300s)"

                print(output or "(empty)")
                results.append({
                    "type": "tool_result",
                    "tool_use_id": block.id,
                    "content": output[:50000]  # Truncate very long outputs
                })

        # 5. Append results and continue the loop
        history.append({"role": "user", "content": results})

if __name__ == "__main__":
    if len(sys.argv) > 1:
        # Subagent mode: execute task and print result
        # This is how parent agents spawn children via bash
        print(chat(sys.argv[1]))
    else:
        # Interactive REPL mode
        history = []
        while True:
            try:
                query = input("\033[36m>> \033[0m")  # Cyan prompt
            except (EOFError, KeyboardInterrupt):
                break
            if query in ("q", "exit", ""):
                break
            print(chat(query, history))
```

这就是整个 Agent。~50 行。

## 子代理工作原理

```
主代理
  └─ bash: python v0_bash_agent.py "分析架构"
       └─ 子代理（独立进程，全新历史）
            ├─ bash: find . -name "*.py"
            ├─ bash: cat src/main.py
            └─ 通过 stdout 返回摘要
```

**进程隔离 = 上下文隔离**

- 子进程有自己的 history=[]
- 父进程捕获 stdout 作为工具结果
- 递归调用实现无限嵌套
## v0 牺牲了什么

| 特性 | v0 | v3 |
| --- | --- | --- |
| 代理类型 | 无 | explore/code/plan |
| 工具过滤 | 无 | 白名单 |
| 进度显示 | 普通 stdout | 行内更新 |
| 代码复杂度 | ~50 行 | ~450 行 |

## v0 证明了什么

**复杂能力从简单规则中涌现：**

- 一个工具足够 — Bash 是通往一切的入口
- 递归 = 层级 — 自我调用实现子代理
- 进程 = 隔离 — 操作系统提供上下文分离
- 提示词 = 约束 — 指令塑造行为
核心模式从未改变：

```python
while True:
    response = model(messages, tools)
    if response.stop_reason != "tool_use":
        return response.text
    results = execute(response.tool_calls)
    messages.append(results)
```

其他一切——待办、子代理、权限——都是围绕这个循环的精化。

## 运行示例

```bash
# 安装依赖
pip install anthropic python-dotenv

# 配置环境变量（创建 .env 文件）
echo "ANTHROPIC_API_KEY=sk-ant-xxx" > .env
# 可选：配置自定义端点（默认使用官方 API）

# 交互模式
python v0_bash_agent.py

# 子代理模式
python v0_bash_agent.py "分析当前目录结构并总结"
```

## 常见问题
**Q: 为什么不直接用 v1？**
A: v0 适合建立直觉。理解"一个工具也能跑通"，再看 v1 的工具拆分会更有体会。

**Q: 子代理不会浪费 token 吗？**
A: 不会。子代理独立运行，只返回摘要到父进程。这反而能减少主上下文中的噪音。

**Q: 生产环境能用 v0 吗？**
A: 不推荐。缺少安全护栏、权限系统、进度显示等。v0 是教学用的最小实现。

---

**Bash 就是一切。**
[v1：模型即代理把工具拆清楚：Read/Write/Edit
](../v1-model-as-agent/)
