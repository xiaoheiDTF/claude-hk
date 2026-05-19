# Claude-Code / Advanced / Agent-Loop / V4-Skills

> 来源: claudecn.com

# v4：Skills 机制

## 知识外化：从训练到编辑的范式转变

Skills 机制体现了一个深刻的范式转变：**知识外化 (Knowledge Externalization)**。

### 传统方式：知识内化于参数

传统 AI 系统的知识都藏在模型参数里。你没法访问、没法修改、没法复用。

想让模型学会新技能？你需要：

- 收集大量训练数据
- 设置分布式训练集群
- 进行复杂的参数微调（LoRA、全量微调等）
- 部署新模型版本
这就像大脑突然失忆，但你没有任何笔记可以恢复记忆。知识被锁死在神经网络的权重矩阵中，对用户完全不透明。

### 新范式：知识外化为文档

代码执行范式改变了这一切。

```
┌─────────────────────────────────────────────────────────────────┐
│                     知识存储层级                                  │
│                                                                  │
│  Model Parameters → Context Window → File System → Skill Library │
│       (内化)            (运行时)        (持久化)      (结构化)     │
│                                                                  │
│  ←───────── 训练修改 ──────────→  ←────── 自然语言修改 ─────────→  │
│     需要集群、数据、专业知识              任何人都可以编辑           │
└─────────────────────────────────────────────────────────────────┘
```

**关键突破**：

- 过去：修改模型行为 = 修改参数 = 需要训练 = 需要 GPU 集群 + 训练数据 + 专业知识
- 现在：修改模型行为 = 修改 SKILL.md = 编辑文本文件 = 任何人都可以做
这就像给 base model 外挂了一个可热插拔的 LoRA 权重，但你不需要对模型本身进行任何参数训练。

### 为什么这很重要

- 民主化：不再需要 ML 专业知识来定制模型行为
- 透明性：知识以人类可读的 Markdown 存储，可审计、可理解
- 复用性：一个 Skill 写一次，可以在任何兼容 Agent 上使用
- 版本控制：Git 管理知识变更，支持协作和回滚
- 在线学习：模型在更大的上下文窗口中"学习"，无需离线训练
传统的微调是**离线学习**：收集数据→训练→部署→使用。
Skills 是**在线学习**：运行时按需加载知识，立即生效。

### 知识层级对比

| 层级 | 修改方式 | 生效时间 | 持久性 | 成本 |
| --- | --- | --- | --- | --- |
| Model Parameters | 训练/微调 | 数小时-数天 | 永久 | $10K-$1M+ |
| Context Window | API 调用 | 即时 | 会话内 | ~$0.01/次 |
| File System | 编辑文件 | 下次加载 | 永久 | 免费 |
| **Skill Library** | **编辑 SKILL.md** | **下次触发** | **永久** | **免费** |

Skills 是最甜蜜的平衡点：持久化存储 + 按需加载 + 人类可编辑。

### 实际意义

假设你想让 Claude 学会你公司特有的代码规范：

**传统方式**：

```
1. 收集公司代码库作为训练数据
2. 准备微调脚本和基础设施
3. 运行 LoRA 微调（需要 GPU）
4. 部署自定义模型
5. 成本：$1000+ 和数周时间
```

**Skills 方式**：

```markdown
# skills/company-standards/SKILL.md
---
name: company-standards
description: 公司代码规范和最佳实践
---

## 命名规范
- 函数名使用小写+下划线
- 类名使用 PascalCase
...
```

```
成本：0，时间：5分钟
```

这就是知识外化的力量：**把需要训练才能编码的知识，变成任何人都能编辑的文档**。

## 问题背景

v3 给了我们子代理来分解任务。但还有一个更深的问题：**模型怎么知道如何处理特定领域的任务？**

- 处理 PDF？需要知道用 pdftotext 还是 PyMuPDF
- 构建 MCP 服务器？需要知道协议规范和最佳实践
- 代码审查？需要一套系统的检查清单
这些知识不是工具——是**专业技能**。Skills 通过让模型按需加载领域知识来解决这个问题。

## 核心概念

### 1. 工具 vs 技能

| 概念 | 是什么 | 例子 |
| --- | --- | --- |
| **Tool** | 模型能**做**什么 | bash, read_file, write_file |
| **Skill** | 模型**知道怎么做** | PDF 处理、MCP 构建 |

工具是能力，技能是知识。

### 2. 渐进式披露

```
Layer 1: 元数据 (始终加载)     ~100 tokens/skill
         └─ name + description

Layer 2: SKILL.md 主体 (触发时)   ~2000 tokens
         └─ 详细指南

Layer 3: 资源文件 (按需)        无限制
         └─ scripts/, references/, assets/
```

这让上下文保持轻量，同时允许任意深度的知识。

### 3. SKILL.md 标准

```
skills/
├── pdf/
│   └── SKILL.md          # 必需
├── mcp-builder/
│   ├── SKILL.md
│   └── references/       # 可选
└── code-review/
    ├── SKILL.md
    └── scripts/          # 可选
```

**SKILL.md 格式**：YAML 前置 + Markdown 正文

```markdown
---
name: pdf
description: 处理 PDF 文件。用于读取、创建或合并 PDF。
---

# PDF 处理技能

## 读取 PDF

使用 pdftotext 快速提取：
\`\`\`bash
pdftotext input.pdf -
\`\`\`
...
```

## 核心代码实现

```python
#!/usr/bin/env python3
"""
v4_skills_agent.py - Mini Claude Code: Skills Mechanism (~550 lines)

Core Philosophy: "Knowledge Externalization"
============================================
v3 gave us subagents for task decomposition. But there's a deeper question:

    How does the model know HOW to handle domain-specific tasks?

The Paradigm Shift: Knowledge Externalization
--------------------------------------------
Traditional AI: Knowledge locked in model parameters
  - To teach new skills: collect data -> train -> deploy
  - Cost: $10K-$1M+, Timeline: Weeks

Skills: Knowledge stored in editable files
  - To teach new skills: write a SKILL.md file
  - Cost: Free, Timeline: Minutes
"""

import os
import re
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
SKILLS_DIR = WORKDIR / "skills"
client = Anthropic(base_url=os.getenv("ANTHROPIC_BASE_URL"))
MODEL = os.getenv("MODEL_ID", "claude-sonnet-4-5-20250929")

# =============================================================================
# SkillLoader - The core addition in v4
# =============================================================================

class SkillLoader:
    """
    Manages loading and parsing of skill files.

    Key Design Decisions:
    --------------------
    1. Progressive disclosure: Metadata always loaded, body on demand
    2. Standard format: YAML frontmatter + Markdown body
    3. File-based: Git-friendly, human-readable, editable
    """

    def __init__(self, skills_dir: Path):
        self.skills_dir = skills_dir
        self.skills = {}
        self.load_skills()

    def load_skills(self):
        """Discover and load all skill metadata."""
        if not self.skills_dir.exists():
            return

        for skill_path in self.skills_dir.glob("*/SKILL.md"):
            skill = self.parse_skill_md(skill_path)
            if skill:
                self.skills[skill["name"]] = skill

    def parse_skill_md(self, path: Path) -> dict:
        """
        Parse YAML frontmatter + Markdown body.

        Returns:
            {name, description, body, path, dir}
        """
        content = path.read_text()

        # Match YAML frontmatter
        match = re.match(r'^---\s*\n(.*?)\n---\s*\n(.*)

## 消息注入（保持缓存）
关键洞察：Skill 内容进入 **tool_result**（user message 的一部分），而不是 system prompt：

```python
def run_skill(skill_name: str) -> str:
    content = SKILLS.get_skill_content(skill_name)
    # 完整内容作为 tool_result 返回
    # 成为对话历史的一部分（user message）
    return f"""<skill-loaded name="{skill_name}">
{content}
</skill-loaded>

Follow the instructions in the skill above."""

def agent_loop(messages: list) -> list:
    while True:
        response = client.messages.create(
            model=MODEL,
            system=SYSTEM,  # 永不改变 - 缓存保持有效！
            messages=messages,
            tools=ALL_TOOLS,
        )
        # Skill 内容作为 tool_result 进入 messages...
```

**关键洞察**：

- Skill 内容作为新消息追加到末尾
- 之前的所有内容（system prompt + 历史消息）都被缓存复用
- 只有新追加的 skill 内容需要计算，整个前缀都命中缓存
## 与生产版本对比

| 机制 | Claude Code / Kode | v4 |
| --- | --- | --- |
| 格式 | SKILL.md (YAML + MD) | 相同 |
| 加载 | Container API | SkillLoader 类 |
| 触发 | 自动 + Skill 工具 | 仅 Skill 工具 |
| 注入 | newMessages (user message) | tool_result (user message) |
| 缓存机制 | 追加到末尾，前缀全部缓存 | 追加到末尾，前缀全部缓存 |
| 版本控制 | Skill Versions API | 省略 |
| 权限 | allowed-tools 字段 | 省略 |

**关键共同点**：两者都将 skill 内容注入对话历史（而非 system prompt），保持 prompt cache 有效。

## 为什么这很重要：缓存与成本

### 自回归模型与 KV Cache

大模型是自回归的：生成每个 token 都要 attend 之前所有 token。为避免重复计算，提供商实现了 **KV Cache**：

```
请求 1: [System, User1, Asst1, User2]
        ←────── 全部计算 ──────→

请求 2: [System, User1, Asst1, User2, Asst2, User3]
        ←────── 缓存命中 ──────→ ←─ 新计算 ─→
               (更便宜)            (正常价格)
```

缓存命中要求**前缀完全相同**。

### 需要注意的模式

| 操作 | 影响 | 结果 |
| --- | --- | --- |
| 编辑历史 | 改变前缀 | 缓存无法复用 |
| 中间插入 | 后续前缀变化 | 需要重新计算 |
| 修改 system prompt | 最前面变化 | 整个前缀需重新计算 |

### 推荐：只追加

```python
# 避免: 编辑历史
messages[2]["content"] = "edited"  # 缓存失效

# 推荐: 只追加
messages.append(new_msg)  # 前缀不变，缓存命中
```

### 长上下文支持
主流模型支持较大的上下文窗口：

- Claude Sonnet 4.5 / Opus 4.5: 200K
- GPT-5.2: 256K+
- Gemini 3 Flash/Pro: 1M
200K tokens 约等于 15 万词，一本 500 页的书。对于大多数 Agent 任务，现有上下文窗口已经足够。

**把上下文当作只追加日志，而非可编辑文档。**

## 系列总结

| 版本 | 主题 | 新增行数 | 核心洞察 |
| --- | --- | --- | --- |
| v1 | Model as Agent | ~200 | 模型是 80%，代码只是循环 |
| v2 | 结构化规划 | ~100 | Todo 让计划可见 |
| v3 | 分而治之 | ~150 | 子代理隔离上下文 |
| **v4** | **领域专家** | **~100** | **Skills 注入专业知识** |

---

**工具让模型能做事，技能让模型知道怎么做。**
[v3：子代理机制分而治之，上下文隔离
](../v3-subagents/)[Agent Skills 文档深入了解 Claude Code 的 Skills 系统
](../../skills/)

, content, re.DOTALL)
        if not match:
            return None

        frontmatter_text, body = match.groups()

        # Parse YAML frontmatter
        frontmatter = {}
        for line in frontmatter_text.strip().split('\n'):
            if ':' in line:
                key, value = line.split(':', 1)
                frontmatter[key.strip()] = value.strip()

        return {
            "name": frontmatter.get("name", path.parent.name),
            "description": frontmatter.get("description", ""),
            "body": body.strip(),
            "path": path,
            "dir": path.parent
        }

    def get_descriptions(self) -> str:
        """
        Generate metadata list for system prompt.

        This is always loaded (Layer 1: ~100 tokens/skill).
        """
        if not self.skills:
            return "No skills available."

        return "\n".join(
            f"- {name}: {skill['description']}"
            for name, skill in self.skills.items()
        )

    def get_skill_content(self, name: str) -> str:
        """
        Get full skill content for context injection.

        This is loaded on demand (Layer 2: ~2000 tokens).
        """
        if name not in self.skills:
            return f"Skill '{name}' not found."

        skill = self.skills[name]
        return f"# Skill: {name}\n\n{skill['body']}"

# Global skill loader instance
SKILLS = SkillLoader(SKILLS_DIR)

# =============================================================================
# System Prompt - Updated for v4
# =============================================================================

SYSTEM = f"""You are a coding agent at {WORKDIR}.

Loop: plan -> act with tools -> update todos -> report.

Rules:
- Use TodoWrite to track multi-step tasks
- Use Skill tool to load domain expertise when needed
- Prefer tools over prose. Act, don't just explain.
- After finishing, summarize what changed.

Available Skills:
{SKILLS.get_descriptions()}"""

# =============================================================================
# Skill Tool - Loads domain knowledge on demand
# =============================================================================

TOOLS = [
    # v1-v3 tools (unchanged)
    # ... bash, read_file, write_file, edit_file, TodoWrite, Task

    # NEW in v4: Skill tool
    {
        "name": "Skill",
        "description": "Load a skill to get domain expertise.",
        "input_schema": {
            "type": "object",
            "properties": {
                "skill": {
                    "type": "string",
                    "description": "Name of the skill to load"
                }
            },
            "required": ["skill"],
        },
    },
]

# =============================================================================
# Tool Execution with Skill Support
# =============================================================================

def run_skill(skill_name: str) -> str:
    """
    Load and inject skill content into conversation.

    Key insight: Skill content enters as tool_result (user message),
    not system prompt. This preserves cache!
    """
    content = SKILLS.get_skill_content(skill_name)

    # Return as tool result - becomes part of conversation history
    return f"""<skill-loaded name="{skill_name}">
{content}
</skill-loaded>

Follow the instructions in the skill above."""

def execute_tool(name: str, args: dict) -> str:
    """Dispatch tool call to implementation."""
    if name == "Skill":
        return run_skill(args["skill"])
    # ... (other tools from v1-v3)
    return f"Unknown tool: {name}"
```

## 消息注入（保持缓存）
关键洞察：Skill 内容进入 **tool_result**（user message 的一部分），而不是 system prompt：

%%CODE_BLOCK_8%%

**关键洞察**：

- Skill 内容作为新消息追加到末尾
- 之前的所有内容（system prompt + 历史消息）都被缓存复用
- 只有新追加的 skill 内容需要计算，整个前缀都命中缓存
## 与生产版本对比

| 机制 | Claude Code / Kode | v4 |
| --- | --- | --- |
| 格式 | SKILL.md (YAML + MD) | 相同 |
| 加载 | Container API | SkillLoader 类 |
| 触发 | 自动 + Skill 工具 | 仅 Skill 工具 |
| 注入 | newMessages (user message) | tool_result (user message) |
| 缓存机制 | 追加到末尾，前缀全部缓存 | 追加到末尾，前缀全部缓存 |
| 版本控制 | Skill Versions API | 省略 |
| 权限 | allowed-tools 字段 | 省略 |

**关键共同点**：两者都将 skill 内容注入对话历史（而非 system prompt），保持 prompt cache 有效。

## 为什么这很重要：缓存与成本

### 自回归模型与 KV Cache

大模型是自回归的：生成每个 token 都要 attend 之前所有 token。为避免重复计算，提供商实现了 **KV Cache**：

%%CODE_BLOCK_9%%

缓存命中要求**前缀完全相同**。

### 需要注意的模式

| 操作 | 影响 | 结果 |
| --- | --- | --- |
| 编辑历史 | 改变前缀 | 缓存无法复用 |
| 中间插入 | 后续前缀变化 | 需要重新计算 |
| 修改 system prompt | 最前面变化 | 整个前缀需重新计算 |

### 推荐：只追加

%%CODE_BLOCK_10%%

### 长上下文支持
主流模型支持较大的上下文窗口：

- Claude Sonnet 4.5 / Opus 4.5: 200K
- GPT-5.2: 256K+
- Gemini 3 Flash/Pro: 1M
200K tokens 约等于 15 万词，一本 500 页的书。对于大多数 Agent 任务，现有上下文窗口已经足够。

**把上下文当作只追加日志，而非可编辑文档。**

## 系列总结

| 版本 | 主题 | 新增行数 | 核心洞察 |
| --- | --- | --- | --- |
| v1 | Model as Agent | ~200 | 模型是 80%，代码只是循环 |
| v2 | 结构化规划 | ~100 | Todo 让计划可见 |
| v3 | 分而治之 | ~150 | 子代理隔离上下文 |
| **v4** | **领域专家** | **~100** | **Skills 注入专业知识** |

---

**工具让模型能做事，技能让模型知道怎么做。**
[v3：子代理机制分而治之，上下文隔离
](../v3-subagents/)[Agent Skills 文档深入了解 Claude Code 的 Skills 系统
](../../skills/)
