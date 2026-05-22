# PRD: Issue 闭环辅助服务与 Session 注册机制

> 版本: v0.2 | 日期: 2026-05-22 | 关联: claude-hk #18/#19/#20/#21/#23

## 1. 背景

claude-hk #18 对现有 Issue 闭环机制做了能力边界评估：Claude Code Hooks 适合做会话内护栏和工具调用前后拦截，但不能主动监听 GitHub 事件，不能在无活跃会话时定时执行，也不适合承载复杂跨会话状态查询。因此，Issue 闭环需要拆成三层：

| 层级 | 负责内容 | 来源 |
| --- | --- | --- |
| Hooks/Skill 护栏 | 禁止 main 分支直接编辑、commit 关联 issue、PR 前 rebase 提醒、claim 前状态检查 | #18 |
| 外部状态服务 | Webhook 接收、SQLite 状态镜像、定时释放/stale、终端看板、API 查询 | #19 |
| Session 注册中心 | SessionID/AgentID、Issue 绑定、心跳、orphaned 检测、交接摘要 | #20 |

当前仓库 `claude_tap_plus` 已经是 Go 单模块，并已有 `internal/session/`、`tests/session/`、`cmd/claude-tap/main.go` 等结构。PRD 建议把 #19 的“外部辅助服务”收敛为 `claude-tap-plus` 的本地 HTTP 服务能力，而不是新增 Python/FastAPI 服务，以保持单二进制部署和统一测试体系。

本 PRD 已通过 `gh issue list/view --repo xiaoheiDTF/claude-hk` 核对当前 issue 列表。当前与本主题直接相关的 open issue：

| Issue | 标题 | 定位 |
| --- | --- | --- |
| #18 | 当前 Issue 闭环机制缺陷分析（结合 Claude Code Hooks 能力评估） | 父问题与能力边界 |
| #19 | Issue 机制外部辅助服务最小功能设计（个人版） | 外部服务与状态库 |
| #20 | 细分子需求：Claude Code 会话注册机制（Session ↔ Agent ↔ Issue 绑定） | Session 注册中心 |
| #21 | Issue 生命周期流程缺少自动推进机制，存在未合并分支和缺失实现分支 | 阶段自动推进缺口 |
| #23 | 继续拆分 issue-fix 和 issue-pr：单职责化 Skill 拆分方案 | Skill 单职责拆分 |

## 2. 产品目标

1. 为个人开发者提供一个本地优先的 Issue 闭环辅助服务。
2. 让 Claude Code Session、Agent、GitHub Issue 之间形成可查询、可恢复、可审计的绑定关系。
3. 将 Hooks/Skill 无法解决的能力转移到本地服务和 GitHub Actions/Webhook。
4. 让异常退出后的 Session 能被识别、释放和接管，降低“任务被占用但无人处理”的风险。
5. 让 issue 生命周期从 `discuss → claim → fix → done → pr → test → review` 有明确状态标签和推进触发器。

## 3. 非目标

1. 不做企业级多用户权限、OAuth、RBAC。
2. 不保证公网高可用；默认本地或内网运行。
3. 不替代 GitHub Issues/Projects，只做状态镜像和辅助决策。
4. 不自动决定复杂产品优先级；里程碑规划仍需人工确认。

## 4. 用户与核心场景

| 用户 | 场景 | 期望 |
| --- | --- | --- |
| 个人开发者 | 查看当前哪些 issue 正在处理、哪些可领取 | 一个终端命令显示看板 |
| Claude Code Agent | 启动时知道当前分支绑定哪个 issue | 自动注册 Session 并绑定 Issue |
| Claude Code Agent | claim/fix/pr 时检查状态 | 调用本地 API 获取 issue/依赖/Session 状态 |
| 个人开发者 | Session 异常退出后继续处理 | 新 Session 能发现 orphaned issue 并提示接管 |
| 维护者 | 长时间无活动的 issue 自动释放或 stale | 定时任务自动更新标签/assignee/comment |

## 5. 总体方案

### 5.1 架构

```text
GitHub Webhook / gh 同步
        |
        v
claude-tap-plus local service
        |
        +-- SQLite: issues / dependencies / sessions / activity_log
        +-- HTTP API: /issues /board /sessions
        +-- Scheduler: timeout release / stale / dependency check
        |
        +-- Claude Code Hooks
        |     +-- SessionStart -> session register
        |     +-- Stop -> heartbeat
        |     +-- SessionEnd -> deregister + handoff
        |
        +-- Skills
              +-- claim/fix/pr 查询本地 API
```

### 5.2 技术选择

| 模块 | 推荐实现 | 原因 |
| --- | --- | --- |
| HTTP 服务 | Go `net/http` 或轻量 router | 当前仓库是 Go，避免新增 Python 运行时 |
| 数据库 | SQLite | 单文件、易备份、适合个人本地服务 |
| GitHub 状态来源 | `gh issue view/list` + Webhook + API fallback | 优先复用本机 gh 登录态，Webhook 只做实时增强 |
| 定时任务 | Go ticker / cron-like scheduler | 满足个人定时释放和 stale 需求 |
| Hooks 集成 | `.claude/hooks/*.sh` 调用本地 API | 与 Claude Code 生命周期对齐 |
| Skill 集成 | claim/fix/pr 调 API | 复用现有 issue 工作流 |

## 6. 功能需求

### FR-1 Hooks 护栏体系

来源: #18

优先实现以下护栏：

| 优先级 | 功能 | 行为 |
| --- | --- | --- |
| P0 | 禁止 main 分支直接编辑 | PreToolUse 对 Edit/Write/MultiEdit 阻断 |
| P0 | PR merge 后清理分支 | PostToolUse 监听 `gh pr merge` 后删除本地/远程分支 |
| P1 | commit 关联 issue 提醒 | branch 含 issue 编号但 commit message 无 `Refs #N` 时提醒 |
| P1 | PR 前 rebase 提醒 | `gh pr create` 前检查是否落后 `origin/main` |
| P1 | issue 创建重复检测 | `gh issue create` 前搜索相似标题 |
| P1 | claim 前状态检查 | issue 必须 open 且未 resolved/blocked |

验收标准：

1. main 分支编辑被阻断并输出明确原因。
2. issue 分支 commit 缺少关联编号时输出提醒。
3. PR 创建前分支落后 main 时输出 rebase 建议。
4. claim 已关闭或 blocked issue 时失败并给出可领取列表。

### FR-2 Issue 状态镜像服务

来源: #19，建议拆成独立 Issue 继续实现。

服务维护 GitHub Issue 的本地状态镜像，支持 `gh` CLI、Webhook 和手动同步三种入口。GitHub Provider 的优先级为：

1. 首选 `gh issue view/list --repo <owner/repo> --json ...`，复用用户本机 `gh auth login` 状态。
2. `gh` 不可用或未登录时，返回明确错误并提示 `gh auth login`。
3. 如果显式配置 token，则 fallback 到 GitHub API。
4. Webhook 只作为实时更新增强，不作为唯一读取来源。

核心表：

```sql
CREATE TABLE issues (
  number INTEGER PRIMARY KEY,
  title TEXT NOT NULL,
  state TEXT NOT NULL,
  assignee TEXT,
  labels_json TEXT NOT NULL DEFAULT '[]',
  pr_number INTEGER,
  pr_state TEXT,
  last_activity_at TEXT,
  created_at TEXT,
  updated_at TEXT
);

CREATE TABLE issue_dependencies (
  issue_number INTEGER NOT NULL,
  depends_on_number INTEGER NOT NULL,
  PRIMARY KEY (issue_number, depends_on_number)
);

CREATE TABLE activity_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  issue_number INTEGER,
  action TEXT NOT NULL,
  actor TEXT,
  payload_json TEXT,
  created_at TEXT NOT NULL
);
```

API：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/webhook/github` | 接收 GitHub Webhook 并 upsert SQLite |
| `POST` | `/sync/github` | 主动拉取 GitHub issue 状态 |
| `GET` | `/issues` | 查询 issue 列表，支持 state/label/assignee 过滤 |
| `GET` | `/issues/{number}` | 查询单个 issue |
| `GET` | `/issues/ready` | 返回 open、无 assignee、无 blocked 的 issue |
| `GET` | `/issues/in-progress` | 返回进行中的 issue |
| `GET` | `/board` | 返回终端友好的文本看板 |

验收标准：

1. 本地启动服务后，`POST /sync/github` 能把 open issue 同步进 SQLite。
2. 模拟 GitHub Webhook 能更新 issue state、labels、assignee、last_activity。
3. `/board` 能显示进行中、可领取、阻塞、最近关闭四类信息。
4. `/issues/ready` 能排除 closed、blocked、已有 assignee 的 issue。

### FR-3 定时任务与自动释放

来源: #19

| 任务 | 频率 | 动作 |
| --- | --- | --- |
| in-progress 提醒 | 6 小时 | 超 6 小时无活动则记录提醒，可选 comment |
| in-progress 释放 | 每天 | 超 7 天无活动则移除 assignee/in-progress，添加 stale 或 released |
| stale 标记 | 每天 | open 且 30 天无活动加 stale |
| stale 关闭 | 每天 | stale 后 7 天仍无活动则关闭 |
| 依赖阻塞检查 | 每天 | `depends on #N` 未关闭则加 blocked |
| 依赖解除检查 | 每天 | 依赖关闭后移除 blocked |

验收标准：

1. 定时任务可通过配置开关启停。
2. 每次自动修改 GitHub 前写入 `activity_log`。
3. dry-run 模式只输出计划动作，不修改 GitHub。
4. 超时释放后 issue comment 包含释放原因和最后活动时间。

### FR-4 Session 注册中心

来源: #20

核心模型：

```text
Session
- session_id
- agent_id
- issue_number nullable
- branch
- cwd
- status: active / idle / ended / orphaned
- started_at
- last_heartbeat_at
- ended_at nullable
```

核心行为：

| 阶段 | 触发 | 行为 |
| --- | --- | --- |
| SessionStart | Claude Hook | 生成 SessionID/AgentID，读取分支，自动绑定 issue |
| Stop | Claude Hook | 发送 heartbeat，更新 last_activity |
| SessionEnd | Claude Hook | 注销 Session；未完成 issue 生成 handoff note |
| 定时检测 | 本地服务 | 超 60 分钟无心跳标记 orphaned 并释放绑定 |
| 查询 | Skill/用户 | 查询 session-list/status/resume |

API：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/sessions/register` | 注册 Session |
| `POST` | `/sessions/{id}/bind` | 绑定 Issue |
| `POST` | `/sessions/{id}/unbind` | 解绑 Issue |
| `POST` | `/sessions/{id}/heartbeat` | 更新心跳 |
| `POST` | `/sessions/{id}/end` | 注销并生成交接 |
| `GET` | `/sessions` | 查询所有 Session |
| `GET` | `/sessions/{id}` | 查询当前 Session |
| `GET` | `/sessions/orphaned` | 查询可接管遗留任务 |

验收标准：

1. 启动 Claude Code 时输出 SessionID、AgentID、绑定状态。
2. 在 `issue-<N>`、`fix/issue-<N>-*`、`feat/issue-<N>-*` 分支上自动绑定 Issue #N。
3. 同一 issue 被多个 active Session 绑定时输出冲突警告，但不默认阻断。
4. 60 分钟无心跳的 Session 被标记为 orphaned。
5. 正常 SessionEnd 且 issue 未完成时，生成 `doc/otherDoc/handoff/<issue-number>.md`。

### FR-5 Skill 集成

来源: #18/#19/#20

| Skill | 集成点 |
| --- | --- |
| `003-4-issue-claim` | claim 前查 `/issues/{id}` 和 `/sessions`，成功后 `/sessions/{id}/bind` |
| `003-5-issue-fix` | 开始前查依赖，done 时更新 issue last_activity 和 session 状态 |
| `003-6-issue-pr` | 创建 PR 前查 issue 状态、分支 rebase 状态、Test Plan |
| 新增 `/session-bind` | 显式绑定当前 Session 到 Issue |
| 新增 `/session-unbind` | 显式解绑 |
| 新增 `/session-status` | 查询当前 Session |
| 新增 `/session-resume` | 查询并接管 orphaned issue |

验收标准：

1. claim 失败时返回可领取 issue 列表。
2. fix 阶段如果 issue blocked，则提示依赖链。
3. session-bind/unbind/status/resume 有终端友好的输出。

## 7. 新拆 Issue 建议

### Issue 标题

细分子需求：GitHub Issue 状态镜像与终端看板 MVP

### 拆分理由

#20 已经把 Session 注册机制从 #19 中拆出，但 #19 仍然过大，包含 Webhook、SQLite、Scheduler、API、看板、Skill 集成。建议先拆出“状态镜像与终端看板”，作为 Session 注册和 Skill 集成的基础设施。没有这个本地状态层，#20 的 `/sessions` 与 issue 冲突检测只能落在文件或实时 GitHub API 上，后续会重复返工。

### 需求范围

1. 在 `claude_tap_plus/internal/issue/` 新增 issue 状态模型和 SQLite 存储。
2. 新增本地 HTTP API：`/sync/github`、`/webhook/github`、`/issues`、`/issues/{number}`、`/issues/ready`、`/board`。
3. 新增 `claude-tap issue-service` 或 `claude-tap serve` 命令启动本地服务。
4. 支持 GitHub CLI/token 拉取 issue 列表作为 webhook fallback。
5. 提供终端看板文本输出。

### 不包含

1. 不做 Session 注册；由 #20 负责。
2. 不做定时释放/stale；后续从 #19 再拆。
3. 不做 Web UI；只提供 plaintext `/board`。
4. 不做认证；默认只监听 `127.0.0.1`。

### 验收标准

1. `claude-tap serve --repo owner/name` 能启动本地服务并创建 SQLite。
2. `POST /sync/github` 后，`GET /issues` 返回同步后的 issue 列表。
3. `GET /issues/ready` 正确过滤 closed、blocked、assigned issue。
4. `GET /board` 输出进行中、可领取、阻塞、最近关闭分区。
5. 有单元测试覆盖 issue label 解析、ready 过滤、board 渲染。

## 8. 里程碑

| 阶段 | 内容 | 依赖 | 预计 |
| --- | --- | --- | --- |
| M1 | Hooks 护栏 P0/P1 | 无 | 0.5-1 天 |
| M2 | 新拆 Issue: 状态镜像 + `/board` | GitHub CLI/token | 1-2 天 |
| M3 | Session 注册中心 MVP | M2 | 1-2 天 |
| M4 | 定时释放/stale/依赖检查 | M2 | 1 天 |
| M5 | Skill 集成与文档 | M2/M3 | 0.5-1 天 |

## 9. 风险与决策

| 风险 | 影响 | 决策 |
| --- | --- | --- |
| Webhook 本地不可达 | issue 状态不实时 | 提供 `/sync/github` fallback |
| SQLite schema 频繁变更 | 开发期迁移成本 | MVP 允许重建库，稳定后加 migration |
| Hooks 阻断过强影响开发 | 降低效率 | P0 用阻断，P1 默认提醒 |
| SessionEnd 不一定可靠触发 | 交接摘要缺失 | 用 heartbeat orphaned 检测兜底 |
| 多 Session 同时处理同一 issue | 冲突 | 默认警告不阻断，保留观察/协作模式 |

## 10. 成功指标

1. 个人开发时通过 `/board` 能在 3 秒内看到当前 issue 状态。
2. Session 异常退出后，新 Session 能发现并接管 orphaned issue。
3. claim/fix/pr 流程中的常见错误能在执行前被提醒或阻断。
4. issue 状态、Session 状态、操作日志都能从 SQLite 查询。
5. 不引入额外服务运行时，保持 `claude-tap-plus` 单二进制分发路径。
