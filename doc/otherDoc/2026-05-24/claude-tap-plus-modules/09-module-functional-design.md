# 模块 9：模块功能设计

> 设计范围：基于 01-08 模块文档整理的功能设计总览。
> 只描述功能和模块关系，不涉及具体实现。

## 总体定位

`claude-tap-plus` 是一个本地优先的 AI Agent 工作环境管理工具，围绕 Claude Code 提供以下能力：

1. 记录 AI Agent API 调用和 token 用量。
2. 固定保存 trace、session、issue、sandbox 元数据。
3. 支持跨机器查找 session 和 trace。
4. 支持 GitHub Issue 本地镜像和开发闭环。
5. 支持 Git Worktree Sandbox，隔离不同分支和工具工作区。
6. 支持多工具打开同一个 sandbox。
7. 保证主项目本地文件不会被 agent 自动修改。

## 功能清单

| 功能 | 作用 | 用户价值 |
|------|------|---------|
| 固定本地存储 | 统一所有数据目录 | 任意目录启动都能归档到同一位置 |
| 项目身份识别 | 统一 project identity | 所有模块用同一身份关联数据 |
| Proxy Trace | 代理 API 请求并记录 | 可回放、审计、统计 API 调用 |
| SSE 重组 | 将 streaming 响应重组 | trace 中能看到完整输出 |
| Usage 统计 | 归一化 token 用量 | 按项目、session、sandbox 统计成本 |
| Session 索引 | 扫描、导入、查询 session | 跨机器快速找到历史上下文 |
| Session 快照 | 备份和恢复 session | 保留现有 push/pull/status 能力 |
| Issue Mirror | 本地镜像 GitHub Issue | hooks、skills、scheduler 可快速查询 |
| Hooks Guard | 关键操作前检查规则 | 防止 main 分支编辑、错误 claim |
| Skill Integration | 给 skill 提供状态查询 | claim/fix/pr 流程更可靠 |
| Scheduler | 定期治理 stale/blocked issue | 无会话时也能维护 issue 状态 |
| Sandbox | Git Worktree 隔离工作区 | 不同分支、不同任务互不影响 |
| Tool Adapter | 多工具启动到 sandbox | 同一套入口管理不同工具 |
| Sync | 同步 metadata、trace、session | 支持跨机器和显式本地更新 |
| 审计日志 | 记录命令、动作、阻断、同步 | 出问题时可追踪 |

## 模块划分

| 模块 | 作用 | 依赖 |
|------|------|------|
| CLI Router | 所有命令入口，参数解析与分发 | 无 |
| Project Resolver | 识别 project、git root、remote、branch、slug | 无 |
| Storage Manager | 管理固定存储根目录，为各模块分配路径 | 无 |
| Proxy Trace | 启动本地代理，转发请求，记录 trace | Project、Storage、SSE、Usage、Trace |
| SSE Reassembler | 解析 SSE chunk，重组 streaming 响应 | 无 |
| Usage Normalizer | 统一多 provider token 字段 | 无 |
| Trace Writer | 写入 JSONL trace，附加上下文元数据 | Storage |
| Session Index | 只读扫描、导入、查询 session/trace | Storage、DB |
| Session Snapshot | 备份和恢复 Claude 原始 session | Storage |
| Issue Mirror | 同步 GitHub Issue，提供查询和看板 | DB |
| Hooks Guard | 操作前规则检查，返回 allow/warn/block | Issue Mirror |
| Skill Integration | 为 issue skill 提供状态查询和决策 | Issue Mirror、Hooks Guard |
| Scheduler | 定时治理 stale、blocked、in-progress | Issue Mirror |
| Sandbox Manager | 管理 Git Worktree sandbox | Storage、Git |
| Tool Adapter | 启动 Claude/Cursor/IDEA/Trae/CMD | Sandbox |
| Sync | 同步 metadata、trace、session、sandbox | Storage、Git |

## 模块关系

- CLI Router 是唯一用户入口。
- Project Resolver 和 Storage Manager 是所有模块的基础依赖。
- Proxy Trace 负责产生 trace 和 usage 数据。
- Session 模块负责 session 查询、导入、快照。
- Issue Mirror 是 Hooks、Skills、Scheduler 的状态来源。
- Sandbox Manager 负责 worktree 生命周期。
- Tool Adapter 依赖 Sandbox Manager 启动目标工具。
- Sync 模块处理跨机器、远端、本地显式同步。

## 数据流转原则

1. **自动写入**：trace、metadata、local db、sandbox 元数据可自动写入 `~/.claude-tap`。
2. **只读扫描**：session-collect 只读 `~/.claude/`，不写回。
3. **显式触发**：session-pull、sandbox apply/sync、GitHub 更新必须用户显式执行。
4. **禁止自动修改**：主项目源代码、Claude 原始 session 目录不能被 agent 自动修改。

## 模块优先级

| 优先级 | 模块 | 原因 |
|--------|------|------|
| P0 | Project Resolver + Storage Manager | 所有模块的基础依赖 |
| P0 | CLI 参数模型 | 后续新参数会和现有透传冲突 |
| P1 | Proxy Trace | 当前核心可用能力 |
| P1 | Session Index | 支撑跨机器查找 |
| P1 | Sandbox + Tool Adapter | 支撑隔离工作区 |
| P2 | Issue Mirror | 支撑 issue 闭环 |
| P2 | Hooks Guard + Skill Integration | 依赖 Issue Mirror |
| P3 | Scheduler | 依赖 Issue Mirror 稳定 |
| P3 | Sync | 依赖 storage/index schema 稳定 |

## 设计边界

**MVP 支持**：固定存储、Proxy trace、Usage 统计、Session 查询与快照、Sandbox worktree、多工具 adapter、Issue Mirror 查询、Hooks/Skill 基础检查。

**MVP 不支持**：虚拟文件系统、自动修改主项目源代码、自动合并/rebase/reset、企业级权限、实时多人协作。
