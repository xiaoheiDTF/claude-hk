# Issue 全局管理 — 总览与架构

> 创建时间：2026-05-27
> 模块：claude-tap-plus / Issue 管理
> 简述：将 Issue 全局管理设计文档拆解为可独立实现的子需求，本文为总览与索引

---

## 问题背景

单 GitHub 账号多 Agent 场景下，`gh issue edit --add-assignee @me` 无法区分不同 session，导致多 Agent 同时领取同一 issue。

## 设计原则

1. **GitHub 是数据源，后端是锁**：后端只记录"哪个 session 正在处理哪个 issue"
2. **session 绑定，非用户绑定**：用 session_id 标识领取者
3. **状态流转由技能触发**：后端不主动感知 GitHub 变化
4. **会话关闭自动释放**：SessionEnd 时自动释放该 session 所有 issue

## 子需求清单

| 编号 | 子需求 | 文档 | 依赖 |
|------|--------|------|------|
| B1 | 数据库表与数据模型 | [database-schema.md](database-schema.md) | 无 |
| B2 | 批量状态查询 API | [issue-check-api.md](issue-check-api.md) | B1 |
| B3 | 原子领取 API | [issue-claim-api.md](issue-claim-api.md) | B1 |
| B4 | 状态流转 API | [issue-status-api.md](issue-status-api.md) | B1 |
| B5 | 释放机制 API | [issue-release-api.md](issue-release-api.md) | B1 |
| B6 | 技能脚本改造 | [skill-refactor/](skill-refactor/) | B2-B5, B7 |
| B7 | 公共后端调用模块与服务启动 | [backend-infrastructure.md](backend-infrastructure.md) | 无 |
| B8 | 统一降级策略 | [degradation-strategy.md](degradation-strategy.md) | B7 |

## 状态流转图

```
                    ┌─────────┐
         ┌─────────►│  idle   │◄────────┐
         │          └────┬────┘         │
         │               │ claim        │ release / session close
         │               ▼              │
         │          ┌─────────┐         │
         │    ┌────►│ claimed │◄───┐    │
         │    │     └────┬────┘    │    │
         │    │          │ fix     │    │
         │    │          ▼         │    │
         │    │     ┌─────────┐    │    │
         │    │     │ fixing  │────┘    │
         │    │     └────┬────┘  reject │
         │    │          │ done         │
         │    │          ▼              │
         │    │    ┌─────────────┐      │
         │    └────│ ready-for-pr│      │
         │         └──────┬──────┘      │
         │                │ pr          │
         │                ▼             │
         │           ┌──────────┐       │
         │           │pr-created│       │
         │           └─────┬────┘       │
         │                 │ test       │
         │                 ▼            │
         │            ┌─────────┐       │
         │            │ testing │       │
         │            └────┬────┘       │
         │                 │ review     │
         │                 ▼            │
         │           ┌───────────┐      │
         └───────────│ reviewing │      │
            reject   └─────┬─────┘      │
                          │ merge       │
                          ▼             │
                     ┌─────────┐        │
                     │ merged  │────────┘
                     └─────────┘
```

## 整体架构

```
GitHub ── gh CLI ──→ Agent (Claude Code 技能脚本) ── HTTP ──→ claude-tap-plus 后端
                         │                                        │
                    003-4 ~ 003-9 技能                        SQLite DB
                    SessionEnd hook                         issue_claims 表
```

## 实现顺序建议

```
B1 (数据库) + B7 (公共模块/服务启动) → B2 (查询) + B3 (领取) + B4 (状态) + B5 (释放) → B6 (技能改造)
                                                                                          B8 (降级策略，可与 B6 同步)
```

B1 和 B7 是基础，B2-B5 可并行开发，B6 依赖全部 API 和公共模块就绪。B8 贯穿所有 API 的降级设计，可与 B6 同步完善。

## 关于 `CLAUDE_SESSION_ID` 环境变量

技能脚本统一使用 `json_get '.session_id'` 获取 session_id，**不要求**模块 C（代理侧）注入 `CLAUDE_SESSION_ID` 环境变量。`backend.sh` 的 `_get_session_id()` 预留了环境变量优先路径，如后续模块 C 支持注入，无需修改技能脚本。
