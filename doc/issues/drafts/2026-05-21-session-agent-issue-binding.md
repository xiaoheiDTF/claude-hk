---
title: 细分子需求：Claude Code 会话注册机制（Session ↔ Agent ↔ Issue 绑定）
labels: skill-system,enhancement
assignee: 
priority: P1
status: draft
created: 2026-05-21
parent: #18
---

## 背景

父 Issue #18 指出：当前机制缺乏 **跨会话状态持久化** 和 **Agent 异常恢复** 能力。根本原因是：
- 每个 Claude Code 会话（Session）是独立的，没有注册机制
- Session 中的 Agent 没有身份标识，无法追踪"哪个 Agent 在处理哪个 Issue"
- Session 异常退出后，新 Session 无法知道上一个 Session 遗留下的任务

本 Issue 将"会话注册机制"拆分为 **6 个可独立实现的子需求**，每个子需求都有明确的输入、输出和验收标准。

---

## 核心模型

```
Session (一次 Claude Code 会话)
  └── SessionID (唯一标识)
  └── AgentID (该会话中的 Agent 身份)
  └── IssueID (当前处理的 Issue，可为空)
  └── Branch (当前 git 分支)
  └── Status: active / idle / ended / orphaned
  └── StartTime / LastHeartbeat / EndTime
```

---

## 子需求 1：Session 启动注册（Session Registration）

### 需求描述
Claude Code 会话启动时（SessionStart Hook），自动向注册中心上报会话信息，完成注册。

### 输入
- `SessionStart` Hook 触发
- 环境信息：`cwd`、git 分支名、系统时间

### 输出
- 生成 `SessionID`（基于时间戳 + 随机数，如 `sess_20260521_143022_a7f3`）
- 生成 `AgentID`（基于 `SessionID` + 主机名前缀）
- 写入注册表（claude-tap-plus SQLite 或本地 JSONL）
- 输出到终端：`[Session] Agent-<agent_id> started, session=<session_id>`

### 自动绑定逻辑
```
1. 读取当前 git 分支名
2. 如果分支名匹配 `issue-<N>-*`，提取 IssueID = N，自动绑定
3. 如果分支名匹配 `fix/issue-<N>-*`、`feat/issue-<N>-*` 等，同样提取 N
4. 如果无法提取 IssueID，标记为 "unbound"
5. 如果该 Issue 已被其他 Session 绑定且状态为 active：
   - 输出警告："Issue #N is already bound to Session <sid>"
   - 不强制阻断（允许观察模式）
```

### 验收标准
- [ ] SessionStart 时终端输出 SessionID 和 AgentID
- [ ] 在 `issue-<N>` 分支上启动时，自动显示 `Bound to Issue #N`
- [ ] 注册信息持久化到本地（session_registry.jsonl 或 SQLite）
- [ ] 同一 Issue 被多个 Session 绑定时输出警告

---

## 子需求 2：Session 与 Issue 显式绑定/解绑（Manual Bind/Unbind）

### 需求描述
除了自动绑定，支持 Agent 在会话中显式切换/解绑 Issue。

### 场景
- Agent 创建了新分支但尚未命名，需要手动绑定 Issue
- Agent 完成了当前 Issue，需要切换到另一个 Issue
- Agent 只是想讨论，不想绑定任何 Issue

### 输入
- 用户命令或 Skill 调用
- 目标 IssueID（绑定）或 `none`（解绑）

### 输出
- 更新注册表中的 IssueID
- 更新 Issue 的 `last_activity`
- 输出：`[Session] Agent-<id> bound to Issue #N` / `unbound`

### 触发方式
```
# 显式绑定
/session-bind #15
# 或
/001-4-issue-claim #15  # claim 时自动调用绑定

# 显式解绑
/session-unbind
```

### 验收标准
- [ ] `/session-bind #N` 能显式将当前 Session 绑定到 Issue #N
- [ ] `/session-unbind` 能解除绑定，标记为 unbound
- [ ] 绑定/解绑时更新注册表和 Issue 的 last_activity
- [ ] 绑定已处于 `in-progress` 且 assignee 不是自己的 Issue 时，提示冲突

---

## 子需求 3：Agent 身份标识与多 Agent 区分（Agent Identity）

### 需求描述
每个 Session 中的 Agent 需要唯一、可读的身份标识，用于多 Agent 场景下的区分和审计。

### AgentID 生成规则
```
AgentID = <hostname-prefix>_<session-short-id>

示例：
  - mypc_a7f3        (主机 mypc 上的 session a7f3)
  - work-laptop_b2e9 (另一台机器上的 session)
```

### 用途
| 场景 | AgentID 用途 |
|------|-------------|
| Issue comment | `Agent mypc_a7f3: 开始解决` |
| 操作日志 | 记录 "谁" 执行了 claim/release |
| 冲突检测 | "Issue #15 is already bound to Agent work-laptop_b2e9" |
| 看板显示 | 显示当前哪个 Agent 在处理哪个 Issue |

### 验收标准
- [ ] Session 注册时生成唯一的 AgentID
- [ ] AgentID 包含在 Issue comment、操作日志中
- [ ] 看板 `/board` 显示 AgentID 而非 SessionID
- [ ] 同一机器上不同 Session 的 AgentID 不重复

---

## 子需求 4：Session 心跳与存活检测（Heartbeat & Liveness）

### 需求描述
Session 运行期间定期心跳，注册中心通过心跳判断 Session 是否存活。长时间无心跳的 Session 标记为 `orphaned`，自动释放关联资源。

### 心跳机制
```
触发方式：Stop Hook（每轮对话结束时）
动作：POST /api/session/<sid>/heartbeat
      { "timestamp": "2026-05-21T14:30:00Z", "round": 15 }
```

### 存活检测规则
| 状态 | 条件 | 动作 |
|------|------|------|
| active | 心跳在 30 分钟内 | 正常 |
| idle | 心跳在 30~60 分钟 | 标记 idle，可选提醒 |
| orphaned | 心跳超过 60 分钟 | 标记 orphaned，释放 Issue 绑定 |

### orphaned Session 处理
```
1. Session 被标记 orphaned
2. 如果绑定了 Issue：
   a. 解除 Issue 绑定
   b. Issue 状态不变（仍 in-progress），但记录 "Agent orphaned"
   c. 下次 SessionStart 时，如果有 orphaned + in-progress 的 Issue，提示是否接管
3. Session 记录 EndTime = 最后一次心跳时间
```

### 验收标准
- [ ] 每轮对话结束（Stop Hook）触发一次心跳
- [ ] 注册中心能检测 orphaned Session（60 分钟无心跳）
- [ ] orphaned Session 自动释放 Issue 绑定
- [ ] 新 Session 启动时，提示是否有 orphaned 遗留的 in-progress Issue
- [ ] 心跳日志写入 `session_heartbeat.jsonl`

---

## 子需求 5：Session 注销与交接（Session Deregistration & Handoff）

### 需求描述
Session 正常结束时（SessionEnd Hook）注销注册信息。如果有关联的未完成任务，生成交接摘要供下一个 Session 接管。

### 正常注销流程
```
SessionEnd Hook 触发
       │
       ▼
检查当前 Issue 状态
       │
       ├── Issue 已关闭 / PR 已合并
       │      └── 正常注销，标记 completed
       │
       └── Issue 仍为 in-progress
              ├── 生成交接摘要（Session Handoff Note）
              │   - 当前分支名
              │   - 未提交变更（git status）
              │   - 最近 3 条 commit
              │   - 当前 TODO 列表（如果有）
              │   - 已知阻塞点
              │
              └── 注销 Session，Issue 保持 in-progress
                     └── 下次 SessionStart 提示接管
```

### 交接摘要格式
```markdown
## Session Handoff Note
- Session: mypc_a7f3
- Issue: #15
- Branch: fix/issue-15-login-white-screen
- Status: in-progress (未完成)

### 当前状态
- 未提交变更: 2 files (src/auth.js, tests/auth.test.js)
- 最近 commit: fix: 修复登录参数校验

### 已知阻塞
- 需要确认 OAuth 回调 URL 配置

### 建议下一步
- 完成测试用例编写
- 提 PR 前 rebase main
```

### 验收标准
- [ ] SessionEnd 时触发注销流程
- [ ] 未完成的 Issue 生成 Handoff Note，写入 `doc/otherDoc/handoff/<issue-id>.md`
- [ ] 下次 SessionStart 时，如有遗留 in-progress Issue，提示 `Resume Issue #N?`
- [ ] Handoff Note 包含 git status、最近 commit、分支名

---

## 子需求 6：跨 Session 状态查询（Cross-Session Query）

### 需求描述
Session 中的 Agent 能查询注册中心，了解其他 Session 的状态、当前任务分配、可领取 Issue 等。

### 查询接口

```bash
# 查询当前所有活跃的 Session
/session-list
# 输出：
#  AgentID      Issue   Branch                     Status   LastHB
#  mypc_a7f3    #15     fix/issue-15-...           active   2min ago
#  mypc_b2e9    -       main                       idle     45min ago

# 查询自己的 Session 状态
/session-status
# 输出：
#  Session: mypc_a7f3
#  Issue: #15 (in-progress)
#  Branch: fix/issue-15-login-white-screen
#  Started: 2026-05-21 14:00
#  Rounds: 23
#  Status: active

# 查询可接管的遗留 Issue
/session-resume
# 输出：
#  Orphaned Issues:
#  #12  [P2] xxx  (Agent mypc_x1y2 orphaned 2h ago)
#  Resume? Y/n
```

### 验收标准
- [ ] `/session-list` 显示所有活跃/idle/orphaned Session
- [ ] `/session-status` 显示当前 Session 的绑定状态
- [ ] `/session-resume` 列出可接管的 orphaned Issue
- [ ] 查询结果包含 AgentID、IssueID、分支、状态、最后心跳时间

---

## 数据流汇总

```
SessionStart
  │
  ├── 生成 SessionID + AgentID
  ├── 读取 git 分支 → 提取 IssueID → 自动绑定
  ├── 写入注册表（active）
  ├── 检查 orphaned Issue → 提示接管
  └── 输出：SessionID, AgentID, Bound Issue

每轮 Stop
  │
  └── POST heartbeat → 更新 last_activity

SessionEnd
  │
  ├── 检查 Issue 完成状态
  ├── 未完成 → 生成 Handoff Note
  ├── 注销 Session（标记 ended）
  └── 释放资源

定时检测（claude-tap-plus 侧）
  │
  └── 60min 无心跳 → 标记 orphaned → 释放 Issue 绑定
```

---

## 涉及改动

| 文件 | 改动 |
|------|------|
| `.claude/settings.json` | 新增 SessionStart/Stop/StopFailure Hooks |
| `.claude/hooks/session-register.sh` | Session 注册脚本 |
| `.claude/hooks/session-deregister.sh` | Session 注销脚本 |
| `.claude/hooks/session-heartbeat.sh` | 心跳脚本 |
| `claude-tap-plus/internal/session/` (新增) | Session 注册中心模块 |
| `claude-tap-plus/internal/api/session.go` (新增) | Session HTTP API |
| `001-4-issue-claim` | claim 时自动调用 `/session-bind` |
| `001-5-issue-fix` | fix 完成后更新 Session 状态 |

## 验收标准（总体）

- [ ] 每次启动 Claude Code 都能看到 SessionID 和 AgentID
- [ ] 在 issue 分支上自动绑定 Issue，不在 issue 分支上显示 unbound
- [ ] 60 分钟无活动后 Session 被标记 orphaned，Issue 自动释放
- [ ] Session 结束时生成 Handoff Note（如有未完成任务）
- [ ] 新 Session 能查询并接管 orphaned Issue
- [ ] 多 Session 同时 claim 同一 Issue 时输出警告

## 发布记录

- Issue #20: https://github.com/xiaoheiDTF/claude-hk/issues/20 (发布于 2026-05-21 22:52)

