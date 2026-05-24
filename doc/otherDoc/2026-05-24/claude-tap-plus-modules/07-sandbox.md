# 模块 7：Sandbox 工作区与 Tool Adapter

> 阶段：M6（独立，可并行开发） | 依赖：Git

## 目标

基于 Git Worktree 的虚拟工作区系统，为不同工具（Claude Code、Cursor、Trae、IDEA）创建独立视图，实现文件系统隔离、分支隔离和工具隔离。

## 核心对象

| 对象 | 说明 |
|------|------|
| Project | 一个真实 Git 项目，用 git remote/root path 识别 |
| Sandbox | 一个隔离工作区，通常对应一个 issue 或任务 |
| Branch | Sandbox 绑定的 Git 分支 |
| Worktree | Sandbox 的真实文件目录 |

## 功能

### 功能 1：Sandbox 自动准备

用户显式指定 sandbox 时，系统必须查找或创建对应 worktree。

1. 当前目录用于识别项目。
2. sandbox 名称默认同时作为 branch 名。
3. sandbox 不存在时自动创建。
4. sandbox 已存在时复用。
5. 创建只影响 worktree 目录和 sandbox 元数据，不修改主项目源代码。

### 功能 2：Tool Adapter 透传

根据 `--tool` 启动对应工具。

1. `--tool claude` 在 worktree 内启动 Claude Code。
2. `--tool cursor` 打开 worktree。
3. `--tool idea` 打开 worktree。
4. `--tool trae` 打开 worktree。
5. `--tool cmd` 打开位于 worktree 的终端。
6. tool 后续参数原样传给对应工具。

### 功能 3：Sandbox 执行命令

支持在 sandbox 内执行命令。

1. 命令 cwd 必须是 sandbox worktree。
2. 注入 sandbox 环境变量。
3. 记录命令、开始时间、结束时间、退出码。
4. 不自动应用修改到主项目。

### 功能 4：Sandbox 同步与应用

同步需要分层，不允许自动覆盖用户主项目。

1. fetch — 只拉远端信息。
2. pull — 更新 sandbox，需要确认。
3. push — 推送 sandbox branch，需要显式执行。
4. apply — 从 sandbox 更新主项目，需要先展示 diff 并确认。

### 功能 5：Sandbox 状态查询

1. 显示 sandbox、branch、worktree、工具启动记录。
2. 显示 Git dirty 状态。
3. 显示 worktree 是否缺失。
4. 支持健康检查和修复建议。

## 本地文件保护规则

**强规则：agent 不直接修改用户当前主项目目录的本地文件。**

允许自动修改：sandbox worktree 内的文件、元数据、日志。

不允许自动修改：主项目根目录源代码、未指定的 sandbox、未经确认的 Git 操作。

主项目根目录只有用户显式 apply/sync 才能被修改。

## MVP 边界

**支持**：一个沙箱对应一个 Git worktree、IDEA/Cursor/Claude Code/Trae 打开 worktree、执行命令、状态管理。

**暂不支持**：虚拟文件系统挂载、自动强杀 IDE 进程、自动合并有未提交修改的 worktree、Session resume。
