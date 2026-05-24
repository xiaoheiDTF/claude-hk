---
title: 设计 claude_tap_plus 沙箱机制，实现多会话分支与工具视图隔离
labels: enhancement
assignee:
priority: P2
status: draft
created: 2026-05-23
---

## 需求描述

设计并实现一个由 `claude_tap_plus` 管理的沙箱机制，使不同 Claude Code / Cursor / Trae 等工具会话可以在同一项目上下文中绑定不同沙箱。每个沙箱对应独立的代码视图、Git 分支和编译器/工具链配置，从而避免多个工具或多个 Agent 在同一个工作目录中互相覆盖分支、索引和构建状态。

核心诉求：

| 维度 | 需求 |
|------|------|
| 文件系统隔离 | 同一项目入口下，不同工具/会话看到不同内容 |
| Git 分支隔离 | 无需反复 `git checkout`，不同会话绑定不同分支 |
| 编译器/IDE 隔离 | Cursor、Claude Code、Trae 等工具可看到不同视图和配置 |
| 入口控制 | 通过 `claude_tap_plus` 创建、进入、切换和销毁沙箱 |

## 背景与动机

当前多 Agent / 多工具并行开发时，最大冲突点是工作目录的物理唯一性：同一路径下只能有一个实际文件内容和一个当前 Git 分支。多个工具同时打开同一目录时，会共享同一份文件、索引、缓存、编译产物和分支状态，容易出现以下问题：

- 一个会话 `git checkout` 后影响其他会话。
- IDE 索引和编译器缓存互相污染。
- 不同 Agent 同时修改同一工作树，难以区分责任边界。
- Claude Code、Cursor、Trae 等工具无法自然绑定到各自的任务分支。

期望通过 `claude_tap_plus` 增加沙箱抽象，让每个会话都有稳定的 `sandbox_id`，并将它映射到独立 worktree、虚拟挂载点或编译器配置。

## Goals

- 支持 `claude_tap_plus sandbox create/enter/list/remove` 等基础沙箱生命周期命令。
- 每个沙箱绑定一个 Git worktree 和分支，保证不同会话的代码修改互不覆盖。
- 支持为不同沙箱记录工具链配置，例如编译器路径、环境变量、构建目录、IDE 配置目录。
- 支持 Claude Code / Cursor / Trae 等工具通过 wrapper 进入指定沙箱。
- 为后续 FUSE / WinFsp 虚拟文件系统层预留设计，使不同进程可以在统一挂载入口下看到不同代码视图。

## Non-Goals

- 首期不要求直接实现完整 FUSE / WinFsp 文件系统。
- 首期不要求所有 IDE 都在 UI 中显示完全相同的物理路径。
- 首期不实现跨沙箱的自动冲突合并。
- 首期不替代 Git 自身的分支、合并、rebase 工作流。

## 期望行为

### 最小可行版本：Git Worktree 沙箱

```bash
claude_tap_plus sandbox create feature-a --branch feature/a
claude_tap_plus sandbox enter feature-a -- claude
claude_tap_plus sandbox enter feature-a -- cursor
```

预期效果：

- 首次创建沙箱时，在项目下或统一缓存目录下创建对应 Git worktree。
- `feature-a` 沙箱绑定 `feature/a` 分支。
- 进入沙箱时注入环境变量，例如：
  - `CLAUDE_TAP_SANDBOX_ID=feature-a`
  - `CLAUDE_TAP_SANDBOX_ROOT=<worktree path>`
  - `CLAUDE_TAP_PROJECT_ROOT=<main project path>`
- 启动的工具默认打开该沙箱 worktree，而不是主工作树。
- 每个沙箱可以有独立构建目录、缓存目录和工具链配置。

### 后续增强版本：虚拟文件系统视图

后续可以引入 FUSE / WinFsp 层，将统一入口映射到不同 worktree：

```text
/mnt/sandboxes/project/src/main.py
  -> session A / sandbox feature-a / real path .worktrees/feature-a/src/main.py
  -> session B / sandbox hotfix-123 / real path .worktrees/hotfix-123/src/main.py
```

该层根据会话 ID、进程 PID、环境变量或注册表记录决定路径重定向目标，从而实现“同一挂载路径，不同进程看到不同内容”的效果。

## 替代方案

| 方案 | 原理 | 优点 | 局限 |
|------|------|------|------|
| Git Worktree | 同一仓库多个工作目录 | Git 原生、可靠、实现成本低 | 物理路径不同 |
| OverlayFS / Union Mount | 只读基座 + 可写覆盖层 | 文件级隔离，适合快照 | Windows 兼容复杂 |
| FUSE / WinFsp | 用户态文件系统拦截文件操作 | 最灵活，可实现同一路径多视图 | 实现和调试成本高 |
| 容器 | namespace + overlay | 隔离完整 | 对 IDE 交互偏重 |
| LVM / Btrfs 快照 | 块级或子卷快照 | 完整副本语义清晰 | 跨平台和资源成本较高 |

建议先实现 Git Worktree + 环境变量 wrapper，验证工作流收益；确认必要后再开发 FUSE / WinFsp 虚拟层。

## 实现思路

### 数据模型

```text
Sandbox
  - id: feature-a
  - project_root: D:\CodeDevelopment\CodeProject\xxx
  - worktree_path: D:\CodeDevelopment\CodeProject\xxx\.worktrees\feature-a
  - branch: feature/a
  - base_branch: main
  - compiler_profile: msvc-2022 | clang | python-venv | node
  - tool_profiles:
      claude: ...
      cursor: ...
      trae: ...
  - created_at
  - last_used_at
  - status: active | idle | removed
```

### 命令草案

```bash
claude_tap_plus sandbox create <name> --branch <branch> [--base main]
claude_tap_plus sandbox enter <name> -- <command...>
claude_tap_plus sandbox list
claude_tap_plus sandbox status <name>
claude_tap_plus sandbox remove <name>
claude_tap_plus sandbox gc
```

### Worktree 创建逻辑

```bash
git worktree add .worktrees/<sandbox> -b <branch> <base>
```

如果分支已存在：

```bash
git worktree add .worktrees/<sandbox> <branch>
```

### 会话绑定

`claude_tap_plus sandbox enter` 启动工具前写入 session registry：

```text
session_id -> sandbox_id -> branch -> worktree_path -> tool
```

这可以和现有 Session / Agent / Issue 绑定机制联动：

- issue 分支可自动创建对应沙箱。
- Agent 启动时自动绑定沙箱。
- Session 心跳更新沙箱 `last_used_at`。
- Session 结束时可提示是否保留或清理沙箱。

### 编译器与工具链隔离

每个沙箱支持独立 profile：

```yaml
compiler:
  id: clang
  env:
    CC: clang
    CXX: clang++
  build_dir: .tap/build/feature-a
tools:
  cursor:
    open_path: ${worktree_path}
  claude:
    cwd: ${worktree_path}
  trae:
    cwd: ${worktree_path}
```

## 修改范围

| 文件/模块 | 改动说明 |
|-----------|----------|
| `claude-tap-plus/internal/sandbox/` | 新增沙箱生命周期、worktree 管理、状态持久化 |
| `claude-tap-plus/internal/session/` | 会话与 sandbox_id 绑定 |
| `claude-tap-plus/internal/cli/` | 新增 `sandbox` 子命令 |
| `.claude/hooks/` | 可选：SessionStart 自动识别或创建沙箱 |
| `doc/` | 补充沙箱机制设计与使用说明 |

## 验收标准

- [ ] 可以通过 `claude_tap_plus sandbox create <name> --branch <branch>` 创建沙箱和对应 worktree。
- [ ] 可以通过 `claude_tap_plus sandbox enter <name> -- <command>` 在沙箱 worktree 中启动指定工具。
- [ ] 不同沙箱可以同时绑定不同 Git 分支，互不影响当前工作树的分支状态。
- [ ] `sandbox list/status` 能展示沙箱名称、分支、worktree 路径、最近使用时间和绑定会话。
- [ ] 沙箱环境变量可被子进程读取，用于后续 hooks、日志和 Issue 绑定。
- [ ] 支持为沙箱保存至少一种编译器/构建 profile，并在进入沙箱时注入环境变量。
- [ ] 文档说明首期 Worktree 方案与后续 FUSE / WinFsp 方案的边界。

## 发布记录

