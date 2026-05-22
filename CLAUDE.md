# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Claude Code configuration and documentation project. Contains no application code — the repo provides a hooks system, custom skills, and Chinese-translated Claude Code documentation.

## Directory Structure

```
claude-hk/
├── .claude/
│   ├── .initialized                    # 首次运行标记文件（JSON：os/python/utf8/时间）
│   ├── dirs.conf                       # 项目目录声明（ensure_dirs.sh 读取此文件）
│   ├── init.sh                         # 首次运行统一初始化（目录、Python、UTF-8）
│   ├── settings.json                   # Hooks 配置（全部 29 个生命周期事件）
│   ├── settings.local.json             # 本地权限白名单
│   ├── hooks/
│   │   ├── base.sh                     # 所有 hook 的公共基础：JSON 读取、日志、输出
│   │   ├── platform.sh                 # 平台检测（linux/macos/windows）+ Python 路径解析
│   │   ├── json_get.py                 # 无 jq 时的 JSON 解析回退
│   │   ├── logs/                       # 按日期的 hook 日志（YYYY-MM-DD.log）
│   │   ├── 01-session-start/base.sh    # 会话启动：首次→init.sh，每次→UTF-8/Python/目录检查
│   │   ├── 03-user-prompt-submit/
│   │   │   ├── base.sh                 # 调度器：调用 skill-inject.sh 注入上下文
│   │   │   └── skill-inject.sh         # 匹配 /技能名，查 registry.conf，运行 on_load.sh 注入上下文
│   │   ├── 02,04~29/base.sh            # 其余生命周期 hook（纯日志记录，暂无其他脚本）
│   │   ├── registry.conf               # Skill 注册表（skill-inject.sh 读取此文件匹配）
│   ├── scripts/
│   │   └── ensure_dirs.sh              # 读取 dirs.conf 创建/检查目录
│   ├── skills/
│   │   ├── log.sh                      # Skill 统一日志模块（skill_log 函数）
│   │   ├── log/                        # Skill 日志目录
│   │   ├── 001-testcode-python/
│   │   │   ├── SKILL.md                # Python 测试脚本 skill 定义
│   │   │   └── scripts/on_load.sh, run.sh
│   │   ├── 002-otherdoc/
│   │   │   ├── SKILL.md                # 文档归档 skill 定义
│   │   │   └── scripts/on_load.sh
│   │   ├── 003-1-issue-init/           # 初始化 issue 标签体系
│   │   ├── 003-2-issue/                # 创建 GitHub Issue
│   │   ├── 003-3-issue-discuss/        # 拉取 Issue 内容讨论
│   │   ├── 003-4-issue-claim/          # 原子领取 Issue
│   │   ├── 003-5-issue-fix/            # 根据 issue 创建分支
│   │   ├── 003-6-issue-done/           # 标记开发完成
│   │   ├── 003-7-issue-pr/             # 创建 PR 关联 issue
│   │   ├── 003-8-issue-test/           # 执行 PR Test Plan
│   │   ├── 003-9-issue-review/         # 审核合并或打回
│   │   ├── 004-git-push/
│   │   │   ├── SKILL.md                # 提交并推送 skill 定义
│   │   │   └── scripts/on_load.sh
│   │   └── 005-git-commit/
│   │       ├── SKILL.md                # 仅提交 skill 定义
│   │       └── scripts/on_load.sh
│   └── myRule/                         # 用户自定义规则（手动维护，不自动注入）
├── claude-code-cli-doc/                # Claude Code 中文文档
│   ├── README.md                       # 文档索引
│   ├── 01~09                           # 官方文档中文翻译（CLI/命令/环境变量/工具/交互/Hooks/Plugins/Channels）
│   ├── Agent技能体系/                  # Agent skill 体系文档
│   ├── Claude-API实战手册/             # Claude API 实战手册
│   ├── Claude-Code使用指南/            # Claude Code 使用指南
│   └── Claude-Code源码分析/            # Claude Code 源码分析
├── doc/                                # Skills 运行时输出目录
│   ├── testcode/python/api/            # Python API 自动化测试脚本
│   ├── testcode/python/other/          # Python 其他脚本
│   ├── otherDoc/YYYY-MM-DD/            # 按日期归档的文档
│   └── issues/
│       ├── drafts/                     # Issues 本地草稿
│       └── templates/                  # Issues 可复用模板
├── .gitignore
└── CLAUDE.md
```

## Architecture

### Hooks Pipeline

**两层架构：`base.sh`（调度器）+ 同目录下的其他脚本（业务逻辑）**

每个数字编号的 hook 目录（`01-session-start/` ~ `29-session-end/`）都遵循同一模式：

```
XX-event-name/
├── base.sh            # 入口，由 settings.json 调用
├── other-script.sh    # base.sh 调用的具体业务脚本（可选）
└── ...
```

1. **`base.sh` 是调度器** — `settings.json` 中每个事件只指向 `base.sh`。`base.sh` 先 source 公共的 `hooks/base.sh`（提供 JSON stdin 读取、`json_get()` 解析、结构化日志、`hook_output()` 输出），然后调用同目录下的其他 `.sh` 脚本来执行具体业务逻辑。目前只有 `01-session-start` 和 `03-user-prompt-submit` 的 `base.sh` 有额外调度逻辑，其余（`02`、`04`~`29`）都是纯日志记录。
2. **同目录下的其他 `.sh` 是实际逻辑** — 目前只有 `03-user-prompt-submit/skill-inject.sh` 一个：它从用户 prompt 中提取 `/技能名`，在 `skills/registry.conf` 注册表中查找匹配，找到后运行对应 skill 的 `on_load.sh`，将输出作为 `additionalContext` 注入会话。这些脚本由 `base.sh` 调度执行，不是独立运行的。
3. **扩展方式** — 给某个事件加新功能时，在该事件目录下新建 `.sh` 脚本，然后在 `base.sh` 中添加调用即可。不需要改 `settings.json`。

**公共基础设施**（`hooks/base.sh`）：
- JSON stdin 读取 → `json_get()` 解析（jq 优先，回退到 `json_get.py` via Python）
- 日志写入 `.claude/hooks/logs/YYYY-MM-DD.log`，格式 `[timestamp] [LEVEL] [EventName]`
- `hook_output()` 封装退出码和结果 JSON

**平台检测**（`hooks/platform.sh`）：通过 `uname -s` 检测 OS，解析 Python 路径（优先嵌入版 `.claude/localLanguage/python/python.exe`，回退到系统 `python3`/`python`）。

### Initialization Flow

First session: `01-session-start/base.sh` detects `.initialized` is missing → runs `init.sh` which:
1. Reads `dirs.conf` and creates all declared directories
2. Detects platform and sets up Python (downloads embeddable Python on Windows)
3. Configures UTF-8 in `~/.bashrc` (with idempotent marker guards)
4. Writes `.initialized` marker with environment info

Every subsequent session: `01-session-start/base.sh` re-checks UTF-8, Python availability, and directory integrity.

### Skill System

Skills are numbered (`001-` through `005-`) for load-order. Each skill目录结构：

```
XXX-skill-name/
├── SKILL.md                # Skill 定义（frontmatter: name/description/allowed-tools + 使用说明）
└── scripts/
    ├── on_load.sh          # 上下文注入脚本（Skill 被调用时自动执行）
    └── run.sh              # 执行脚本（可选，部分 skill 有）
```

**`on_load.sh` 的作用：前置上下文注入。** 当用户提交包含 `/技能名` 的 prompt 时，`03-user-prompt-submit/base.sh` 调用 `skill-inject.sh`，后者从用户输入中提取 `/技能名`，在 `skills/registry.conf` 中查找匹配 → 找到后运行对应 skill 的 `scripts/on_load.sh` → 将 stdout 作为 `additionalContext` 注入到当前会话。`on_load.sh` 动态输出环境信息（日期、目录、Python 版本、git 分支等），让 Skill 执行时拥有更准确的上下文。

**新增 Skill 时需要两步注册：**
1. 在 `skills/` 下创建目录和 `SKILL.md` + `scripts/on_load.sh`
2. 在 `skills/registry.conf` 中添加一行（`skill-inject.sh` 依赖此文件匹配）

**完整调用链：**
```
用户提交包含 /技能名 的 prompt
  → settings.json 触发 UserPromptSubmit 事件
    → 03-user-prompt-submit/base.sh（调度器）
      → skill-inject.sh（提取 /技能名，查 registry.conf）
        → 对应 skill 的 scripts/on_load.sh（生成上下文）
          → stdout 作为 additionalContext 注入会话
            → Skill 正式执行（SKILL.md 中的指令）
```

Directory requirements are centralized in `.claude/dirs.conf` — `ensure_dirs.sh` reads it to create or verify directories.

### Skills Summary

| Skill | Output Directory | Purpose |
|-------|-----------------|---------|
| `001-testcode-python` | `doc/testcode/python/{api,other}/` | Python test scripts and utilities |
| `002-otherdoc` | `doc/otherDoc/YYYY-MM-DD/` | General documentation by date |
| `003-1-issue-init` | — | 初始化 issue 标签体系（一次性） |
| `003-2-issue` | `doc/issues/{drafts,templates}/` | 创建 GitHub Issue |
| `003-3-issue-discuss` | — | 拉取 Issue 内容进行讨论 |
| `003-4-issue-claim` | — | 原子领取 Issue |
| `003-5-issue-fix` | — | 根据 issue 创建分支并开始开发 |
| `003-6-issue-done` | — | 标记开发完成，准备提 PR |
| `003-7-issue-pr` | — | 创建 PR 关联 issue |
| `003-8-issue-test` | — | 执行 PR 的 Test Plan |
| `003-9-issue-review` | — | 审核合并或打回 PR |
| `004-git-push` | — | Commit (grouped, Chinese messages) + push |
| `005-git-commit` | — | Commit only (grouped, Chinese messages), no push |

## Git Commit Convention (Skills 004/005)

All commits use Chinese: `<type>: <主描述>` with `- 具体修改描述` sub-items.

Types: fix/feat/update/style/refactor/perf/test/docs/revert/build/chore

Grouping priority: type → directory/module → functional association → impact scope. Never `git add .` or `git add -A` — always add specific files by group.


# CLAUDE.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.