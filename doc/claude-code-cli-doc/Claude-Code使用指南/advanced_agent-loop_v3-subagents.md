# Claude-Code / Advanced / Agent-Loop / V3-Subagents

> 来源: claudecn.com

# v3：子代理机制

v2 添加了规划。但对于大型任务如"探索代码库然后重构认证"，单一 Agent 会撞上上下文限制。探索过程把 20 个文件倒进历史记录，重构时失去焦点。

v3 添加了 **Task 工具**：生成带有隔离上下文的子代理。

## 问题

单 Agent 的上下文污染：

```
主 Agent 历史：
  [探索中...] cat file1.py -> 500 行
  [探索中...] cat file2.py -> 300 行
  ... 15 个文件 ...
  [现在重构...] "等等，file1 里有什么来着？"
```

解决方案：**把探索委托给子代理**：

```
主 Agent 历史：
  [Task: 探索代码库]
    -> 子代理探索 20 个文件
    -> 返回: "认证在 src/auth/，数据库在 src/models/"
  [现在用干净的上下文重构]
```

## 代理类型注册表
每种代理类型定义其能力：

```python
AGENT_TYPES = {
    "explore": {
        "description": "只读，用于搜索和分析",
        "tools": ["bash", "read_file"],  # 不能写
        "prompt": "搜索和分析。不要修改。返回简洁摘要。"
    },
    "code": {
        "description": "完整代理，用于实现",
        "tools": "*",  # 所有工具
        "prompt": "高效实现更改。"
    },
    "plan": {
        "description": "规划和分析",
        "tools": ["bash", "read_file"],  # 只读
        "prompt": "分析并输出编号计划。不要改文件。"
    }
}
```

## 核心代码实现

```python
#!/usr/bin/env python3
"""
v3_subagent.py - Mini Claude Code: Subagents (~450 lines)

Core Philosophy: "Divide and Conquer with Context Isolation"
===========================================================
v2 added planning. But for large tasks like "explore codebase then refactor auth",
a single agent hits context limits. Exploration dumps 20 files into history,
then refactor loses focus.

v3 adds the Task tool: spawns subagents with isolated contexts.
"""

import os
import subprocess
import sys
from pathlib import Path

from anthropic import Anthropic
from dotenv import load_dotenv

load_dotenv(override=True)

# =============================================================================
# Configuration
# =============================================================================

WORKDIR = Path.cwd()
client = Anthropic(base_url=os.getenv("ANTHROPIC_BASE_URL"))
MODEL = os.getenv("MODEL_ID", "claude-sonnet-4-5-20250929")

# =============================================================================
# Agent Type Registry - The core addition in v3
# =============================================================================

AGENT_TYPES = {
    "explore": {
        "description": "Read-only, for searching and analyzing",
        "tools": ["bash", "read_file"],  # No write tools
        "prompt": "Search and analyze. Do not modify. Return concise summary."
    },
    "code": {
        "description": "Full agent, for implementation",
        "tools": "*",  # All tools
        "prompt": "Implement changes efficiently."
    },
    "plan": {
        "description": "Planning and analysis",
        "tools": ["bash", "read_file"],  # Read-only
        "prompt": "Analyze and output numbered plan. Do not modify files."
    }
}

def get_tools_for_agent(agent_type: str) -> list:
    """
    Filter tools based on agent type.

    - explore: Only bash + read_file
    - code: All tools
    - plan: Only bash + read_file

    Subagents don't get Task tool (prevents infinite recursion).
    """
    allowed = AGENT_TYPES[agent_type]["tools"]
    if allowed == "*":
        # Return all tools except Task (no recursion in demo)
        return [t for t in TOOLS if t["name"] != "Task"]
    return [t for t in TOOLS if t["name"] in allowed]

# =============================================================================
# Task Tool - Spawns subagents
# =============================================================================

TOOLS = [
    # v1 tools (unchanged)
    # ... bash, read_file, write_file, edit_file

    # v2 TodoWrite (unchanged)
    # ... TodoWrite

    # NEW in v3: Task tool
    {
        "name": "Task",
        "description": "Spawn a subagent for focused subtasks.",
        "input_schema": {
            "type": "object",
            "properties": {
                "description": {
                    "type": "string",
                    "description": "Short task name (3-5 words)"
                },
                "prompt": {
                    "type": "string",
                    "description": "Detailed instructions for the subagent"
                },
                "agent_type": {
                    "type": "string",
                    "enum": ["explore", "code", "plan"],
                    "description": "Type of subagent to spawn"
                }
            },
            "required": ["description", "prompt", "agent_type"],
        },
    },
]

# =============================================================================
# Subagent Execution
# =============================================================================

def run_subagent(description: str, prompt: str, agent_type: str) -> str:
    """
    Execute a subagent with isolated context.

    Key concepts:
    1. Context isolation: Fresh sub_messages = []
    2. Tool filtering: get_tools_for_agent()
    3. Specialized behavior: Agent-specific system prompt
    4. Result abstraction: Only return final text
    """
    config = AGENT_TYPES[agent_type]

    # 1. Agent-specific system prompt
    sub_system = f"""You are a {agent_type} subagent working in {WORKDIR}.

{config['prompt']}

Rules:
- Prefer tools over prose. Act, don't just explain.
- Return a concise summary of your findings."""

    # 2. Filtered tools
    sub_tools = get_tools_for_agent(agent_type)

    # 3. Isolated history (KEY: no parent context)
    sub_messages = [{"role": "user", "content": prompt}]

    # 4. Same agent loop
    tool_count = 0
    while True:
        response = client.messages.create(
            model=MODEL,
            system=sub_system,
            messages=sub_messages,
            tools=sub_tools,
            max_tokens=8000,
        )

        # Collect tool calls
        tool_calls = []
        for block in response.content:
            if block.type == "tool_use":
                tool_calls.append(block)

        # If no tool calls, we're done
        if response.stop_reason != "tool_use":
            # Return final text only
            return "".join(b.text for b in response.content if hasattr(b, "text"))

        # Execute tools
        results = []
        for tc in tool_calls:
            tool_count += 1
            output = execute_tool(tc.name, tc.input)
            results.append({
                "type": "tool_result",
                "tool_use_id": tc.id,
                "content": output[:50000],  # Truncate long outputs
            })

        sub_messages.append({"role": "assistant", "content": response.content})
        sub_messages.append({"role": "user", "content": results})

def execute_tool(name: str, args: dict) -> str:
    """Dispatch tool call to implementation."""
    if name == "Task":
        return run_subagent(args["description"], args["prompt"], args["agent_type"])
    # ... (other tools from v1/v2)
    return f"Unknown tool: {name}"
```

## Task 工具

```python
{
    "name": "Task",
    "description": "为聚焦的子任务生成子代理",
    "input_schema": {
        "description": "短任务名（3-5 词）",
        "prompt": "详细指令",
        "agent_type": "explore | code | plan"
    }
}
```

主代理调用 Task → 子代理运行 → 返回摘要。

## 子代理执行

Task 工具的核心：

```python
def run_task(description, prompt, agent_type):
    config = AGENT_TYPES[agent_type]

    # 1. 代理特定的系统提示词
    sub_system = f"You are a {agent_type} subagent.\n{config['prompt']}"

    # 2. 过滤后的工具
    sub_tools = get_tools_for_agent(agent_type)

    # 3. 隔离的历史（关键：没有父上下文）
    sub_messages = [{"role": "user", "content": prompt}]

    # 4. 同样的查询循环
    while True:
        response = client.messages.create(
            model=MODEL, system=sub_system,
            messages=sub_messages, tools=sub_tools
        )
        if response.stop_reason != "tool_use":
            break
        # 执行工具，追加结果...

    # 5. 只返回最终文本
    return extract_final_text(response)
```

**关键概念：**

| 概念 | 实现 |
| --- | --- |
| 上下文隔离 | 全新的 `sub_messages = []` |
| 工具过滤 | `get_tools_for_agent()` |
| 专门化行为 | 代理特定的系统提示词 |
| 结果抽象 | 只返回最终文本 |

## 工具过滤

```python
def get_tools_for_agent(agent_type):
    allowed = AGENT_TYPES[agent_type]["tools"]
    if allowed == "*":
        return BASE_TOOLS  # 不给 Task（演示中不递归）
    return [t for t in BASE_TOOLS if t["name"] in allowed]
```

- explore：只有 bash + read_file
- code：所有工具
- plan：只有 bash + read_file子代理不获得 Task 工具（防止无限递归）。

## 进度显示

子代理输出不污染主聊天：

```
你: 探索代码库
> Task: 探索代码库
  [explore] 探索代码库 ... 5 个工具, 3.2s
  [explore] 探索代码库 - 完成 (8 个工具, 5.1s)

这是我发现的: ...
```

实时进度，干净的最终输出。

## 典型流程

```
用户: "把认证重构为 JWT"

主 Agent:
  1. Task(explore): "找到所有认证相关文件"
     -> 子代理读取 10 个文件
     -> 返回: "认证在 src/auth/login.py，session 在..."

  2. Task(plan): "设计 JWT 迁移方案"
     -> 子代理分析结构
     -> 返回: "1. 添加 jwt 库 2. 创建 token 工具..."

  3. Task(code): "实现 JWT tokens"
     -> 子代理写代码
     -> 返回: "创建了 jwt_utils.py，更新了 login.py"

  4. 总结更改
```

每个子代理有干净的上下文。主代理保持聚焦。

## 对比

| 方面 | v2 | v3 |
| --- | --- | --- |
| 上下文 | 单一，增长中 | 每任务隔离 |
| 探索 | 污染历史 | 包含在子代理中 |
| 并行 | 否 | 可能（演示中没有） |
| 新增代码 | ~300 行 | ~450 行 |

## 模式

```
复杂任务
  └─ 主 Agent（协调者）
       ├─ 子代理 A (explore) -> 摘要
       ├─ 子代理 B (plan) -> 计划
       └─ 子代理 C (code) -> 结果
```

同样的 Agent 循环，不同的上下文。这就是全部技巧。

---

**分而治之。上下文隔离。**
[v2：显式 Todo把计划做成可见状态机，减少跑偏
](../v2-explicit-planning-todo/)[v4：Skills 机制知识外化：从训练到编辑的范式转变
](../v4-skills/)
