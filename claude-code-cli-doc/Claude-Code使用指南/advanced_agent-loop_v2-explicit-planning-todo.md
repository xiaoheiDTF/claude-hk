# Claude-Code / Advanced / Agent-Loop / V2-Explicit-Planning-Todo

> 来源: claudecn.com

# v2：显式 Todo

v1 能工作。但对于复杂任务，模型会失去方向。

让它"重构认证、添加测试、更新文档"，看看会发生什么。没有显式规划，它在任务间跳跃、忘记步骤、失去焦点。

v2 只添加一样东西：**Todo 工具**。约 100 行新代码，根本性地改变了 Agent 的工作方式。

## 问题

在 v1 中，计划只存在于模型的"脑中"：

```
v1："我先做 A，再做 B，然后 C"（不可见）
    10 次工具调用后："等等，我在干什么？"
```

Todo 工具让它显式化：

```
v2:
  [ ] 重构认证模块
  [>] 添加单元测试         <- 当前在这
  [ ] 更新文档
```

现在你和模型都能看到计划。

## TodoManager

带约束的列表：

```python
class TodoManager:
    def __init__(self):
        self.items = []  # 最多 20 条

    def update(self, items):
        # 验证规则：
        # - 每条需要: content, status, activeForm
        # - Status: pending | in_progress | completed
        # - 只能有一个 in_progress
        # - 无重复，无空项
```

约束很重要：

| 规则 | 原因 |
| --- | --- |
| 最多 20 条 | 防止无限列表 |
| 只能一个进行中 | 强制聚焦 |
| 必填字段 | 结构化输出 |

这些不是任意的——它们是护栏。

## 核心代码实现

```python
#!/usr/bin/env python3
"""
v2_todo_agent.py - Mini Claude Code: Structured Planning (~300 lines)

Core Philosophy: "Make Plans Visible"
=====================================
v1 works great for simple tasks. But ask it to "refactor auth, add tests,
update docs" and watch what happens. Without explicit planning, the model:
  - Jumps between tasks randomly
  - Forgets completed steps
  - Loses focus mid-way

The Solution - TodoWrite Tool:
-----------------------------
v2 adds ONE new tool that fundamentally changes how the agent works:
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
# TodoManager - The core addition in v2
# =============================================================================

class TodoManager:
    """
    Manages a structured task list with enforced constraints.

    Key Design Decisions:
    --------------------
    1. Max 20 items: Prevents the model from creating endless lists
    2. One in_progress: Forces focus - can only work on ONE thing at a time
    3. Required fields: Each item needs content, status, and activeForm

    The activeForm field deserves explanation:
    - It's the PRESENT TENSE form of what's happening
    - Shown when status is "in_progress"
    - Example: content="Add tests", activeForm="Adding unit tests..."

    This gives real-time visibility into what the agent is doing.
    """

    def __init__(self):
        self.items = []

    def update(self, items: list) -> str:
        """
        Validate and update the todo list.

        Validation Rules:
        - Each item must have: content, status, activeForm
        - Status must be: pending | in_progress | completed
        - Only ONE item can be in_progress at a time
        - Maximum 20 items allowed

        Returns:
            Rendered text view of the todo list
        """
        validated = []
        in_progress_count = 0

        for i, item in enumerate(items):
            # Extract and validate fields
            content = str(item.get("content", "")).strip()
            status = str(item.get("status", "pending")).lower()
            active_form = str(item.get("activeForm", "")).strip()

            # Validation checks
            if not content:
                raise ValueError(f"Item {i}: content required")
            if status not in ("pending", "in_progress", "completed"):
                raise ValueError(f"Item {i}: invalid status '{status}'")
            if not active_form:
                raise ValueError(f"Item {i}: activeForm required")

            if status == "in_progress":
                in_progress_count += 1

            validated.append({
                "content": content,
                "status": status,
                "activeForm": active_form
            })

        # Enforce constraints
        if len(validated) > 20:
            raise ValueError("Max 20 todos allowed")
        if in_progress_count > 1:
            raise ValueError("Only one task can be in_progress at a time")

        self.items = validated
        return self.render()

    def render(self) -> str:
        """
        Render the todo list as human-readable text.

        Format:
            [x] Completed task
            [>] In progress task <- Doing something...
            [ ] Pending task

            (2/3 completed)
        """
        if not self.items:
            return "No todos."

        lines = []
        for item in self.items:
            if item["status"] == "completed":
                lines.append(f"[x] {item['content']}")
            elif item["status"] == "in_progress":
                lines.append(f"[>] {item['content']} <- {item['activeForm']}")
            else:
                lines.append(f"[ ] {item['content']}")

        completed = sum(1 for t in self.items if t["status"] == "completed")
        lines.append(f"\n({completed}/{len(self.items)} completed)")

        return "\n".join(lines)

# Global todo manager instance
TODO = TodoManager()

# =============================================================================
# System Prompt - Updated for v2
# =============================================================================

SYSTEM = f"""You are a coding agent at {WORKDIR}.

Loop: plan -> act with tools -> update todos -> report.

Rules:
- Use TodoWrite to track multi-step tasks
- Mark tasks in_progress before starting, completed when done
- Prefer tools over prose. Act, don't just explain.
- After finishing, summarize what changed."""

# =============================================================================
# System Reminders - Soft prompts to encourage todo usage
# =============================================================================

# Shown at the start of conversation
INITIAL_REMINDER = "<reminder>Use TodoWrite for multi-step tasks.</reminder>"

# Shown if model hasn't updated todos in a while
NAG_REMINDER = "<reminder>10+ turns without todo update. Please update todos.</reminder>"

# =============================================================================
# Tool Definitions (v1 tools + TodoWrite)
# =============================================================================

TOOLS = [
    # v1 tools (unchanged) - bash, read_file, write_file, edit_file
    # ... (same as v1)

    # NEW in v2: TodoWrite
    {
        "name": "TodoWrite",
        "description": "Update the task list. Use to plan and track progress.",
        "input_schema": {
            "type": "object",
            "properties": {
                "items": {
                    "type": "array",
                    "description": "Complete list of tasks (replaces existing)",
                    "items": {
                        "type": "object",
                        "properties": {
                            "content": {
                                "type": "string",
                                "description": "Task description"
                            },
                            "status": {
                                "type": "string",
                                "enum": ["pending", "in_progress", "completed"],
                                "description": "Task status"
                            },
                            "activeForm": {
                                "type": "string",
                                "description": "Present tense action, e.g. 'Reading files'"
                            },
                        },
                        "required": ["content", "status", "activeForm"],
                    },
                }
            },
            "required": ["items"],
        },
    },
]

# =============================================================================
# Agent Loop (with todo tracking)
# =============================================================================

# Track how many rounds since last todo update
rounds_without_todo = 0

def agent_loop(messages: list) -> list:
    """
    Agent loop with todo usage tracking.

    Same core loop as v1, but now we track whether the model
    is using todos. If it goes too long without updating,
    we inject a reminder into the next user message.
    """
    global rounds_without_todo

    while True:
        response = client.messages.create(
            model=MODEL,
            system=SYSTEM,
            messages=messages,
            tools=TOOLS,
            max_tokens=8000,
        )

        tool_calls = []
        for block in response.content:
            if hasattr(block, "text"):
                print(block.text)
            if block.type == "tool_use":
                tool_calls.append(block)

        if response.stop_reason != "tool_use":
            messages.append({"role": "assistant", "content": response.content})
            return messages

        results = []
        used_todo = False

        for tc in tool_calls:
            print(f"\n> {tc.name}")
            output = execute_tool(tc.name, tc.input)
            preview = output[:300] + "..." if len(output) > 300 else output
            print(f"  {preview}")

            results.append({
                "type": "tool_result",
                "tool_use_id": tc.id,
                "content": output,
            })

            # Track todo usage
            if tc.name == "TodoWrite":
                used_todo = True

        # Update counter: reset if used todo, increment otherwise
        if used_todo:
            rounds_without_todo = 0
        else:
            rounds_without_todo += 1

        messages.append({"role": "assistant", "content": response.content})

        # Inject NAG_REMINDER into user message if model hasn't used todos
        if rounds_without_todo > 10:
            results.insert(0, {"type": "text", "text": NAG_REMINDER})

        messages.append({"role": "user", "content": results})

def execute_tool(name: str, args: dict) -> str:
    """Dispatch tool call to implementation."""
    if name == "TodoWrite":
        return TODO.update(args["items"])
    # ... (other tools from v1)
    return f"Unknown tool: {name}"
```

## 工具定义

```python
{
    "name": "TodoWrite",
    "input_schema": {
        "items": [{
            "content": "任务描述",
            "status": "pending | in_progress | completed",
            "activeForm": "现在进行时: '正在读取文件'"
        }]
    }
}
```

`activeForm` 显示正在发生什么：

```
[>] 正在读取认证代码...  <- activeForm
[ ] 添加单元测试
```

## 系统提醒
软约束，鼓励使用 todo：

```python
INITIAL_REMINDER = "<reminder>多步骤任务请使用 TodoWrite。</reminder>"
NAG_REMINDER = "<reminder>已超过 10 轮未更新 todo，请更新。</reminder>"
```

作为上下文注入，不是命令：

```python
if rounds_without_todo > 10:
    inject_reminder(NAG_REMINDER)
```

模型看到它们但不回应。

## 反馈循环

当模型调用 `TodoWrite`：

```
输入：
  [x] 重构认证（已完成）
  [>] 添加测试（进行中）
  [ ] 更新文档（待办）

返回：
  "[x] 重构认证
   [>] 添加测试
   [ ] 更新文档
   (1/3 已完成)"
```

模型看到自己的计划，更新它，带着上下文继续。

## 何时使用 Todo

不是每个任务都需要：

| 适合 | 原因 |
| --- | --- |
| 多步骤工作 | 5+ 步需要追踪 |
| 长对话 | 20+ 次工具调用 |
| 复杂重构 | 多个文件 |
| 教学 | 可见的"思考过程" |

经验法则：**如果你会写清单，就用 todo**。

## 集成方式

v2 在 v1 基础上添加，不改变它：

```python
# v1 工具
tools = [bash, read_file, write_file, edit_file]

# v2 添加
tools.append(TodoWrite)
todo_manager = TodoManager()

# v2 追踪使用
if rounds_without_todo > 10:
    inject_reminder()
```

约 100 行新代码。Agent 循环不变。

## 更深的洞察

**结构既约束又赋能。**

Todo 的约束（最大条目、只能一个进行中）赋能了（可见计划、追踪进度）。

Agent 设计中的模式：

- max_tokens 约束 → 赋能可管理的响应
- 工具 Schema 约束 → 赋能结构化调用
- Todo 约束 → 赋能复杂任务完成
好的约束不是限制，而是脚手架。

---

**显式规划让 Agent 可靠。**
[v1：模型即代理把工具拆清楚：Read/Write/Edit
](../v1-model-as-agent/)[v3：子代理与上下文隔离探索不污染主会话，主对话更聚焦
](../v3-subagents/)
