# 2026-05-27 claude-tap-plus 架构与包结构设计

> 基于 `doc/otherDoc/2026-05-26` 与 `doc/otherDoc/2026-05-27` 下的会话管理、Issue 全局管理、Module D 技能改造需求整理。

## 设计目标

本轮只定义架构边界和包结构，不展开具体实现代码。

核心目标：

1. 在现有 `claude_tap_plus` Go 工程内增加轻量后端服务。
2. 支持 Session 注册、关闭、查询。
3. 支持 Issue 批量状态查询、原子领取、状态流转、释放。
4. 支持技能脚本通过统一 `backend.sh` 调用后端。
5. 后端只做协调、锁定和索引，不替代 GitHub，也不存储完整对话内容。

## 总体架构

```text
Claude Code / skills / hooks
        |
        | curl
        v
claude-tap-plus backend
        |
        | net/http
        v
internal/backend/api
        |
        v
internal/backend/service
        |
        v
internal/backend/store
        |
        v
SQLite

claude-tap-plus proxy
        |
        v
internal/proxy -> internal/trace -> local JSONL trace files
```

系统分成两条主链路：

1. `proxy` 链路：继续负责 Claude API 代理、SSE 重组、usage 统计、trace JSONL 写入。
2. `backend` 链路：新增 HTTP 服务，负责 Session 元数据和 Issue 协作状态。

两条链路共享配置、路径解析、领域模型中的少量通用能力，但不互相耦合。后端不读取或写入完整 trace 内容，只保存 `trace_path`。

## CLI 入口设计

当前工程已有单一入口：

```text
claude_tap_plus/cmd/claude-tap/main.go
```

建议继续使用单二进制模式，在现有命令下新增子命令：

```bash
claude-tap-plus backend --port 8080 --db ./backend.db
claude-tap-plus backend --config ./backend.conf
```

不优先新增 `cmd/claude-tap-server`。原因是需求文档中的启动方式已经倾向 `claude-tap-plus backend`，现有 `main.go` 也已经是子命令路由结构，直接扩展成本最低。

推荐入口结构：

```text
claude_tap_plus/
└── cmd/
    └── claude-tap/
        ├── main.go              # 子命令路由
        ├── proxy_cmd.go         # 可选：拆出现有 runProxy
        ├── session_cmd.go       # 可选：拆出现有 session-push/pull/status
        └── backend_cmd.go       # 新增：backend 子命令参数解析和启动
```

如果短期不拆文件，也可以只在 `main.go` 增加 `backend` case。后续文件过大时再拆。

## Go 包结构

推荐新增结构如下：

```text
claude_tap_plus/
├── cmd/
│   └── claude-tap/
│       └── backend_cmd.go
├── internal/
│   ├── backend/
│   │   ├── config.go
│   │   ├── server.go
│   │   ├── errors.go
│   │   ├── api/
│   │   │   ├── router.go
│   │   │   ├── health_handler.go
│   │   │   ├── session_handler.go
│   │   │   ├── issue_handler.go
│   │   │   ├── request.go
│   │   │   └── response.go
│   │   ├── domain/
│   │   │   ├── session.go
│   │   │   ├── issue.go
│   │   │   ├── machine.go
│   │   │   └── project.go
│   │   ├── service/
│   │   │   ├── session_service.go
│   │   │   ├── issue_service.go
│   │   │   └── cleanup_service.go
│   │   └── store/
│   │       ├── store.go
│   │       ├── sqlite.go
│   │       ├── migrations.go
│   │       ├── session_store.go
│   │       └── issue_store.go
│   ├── config/
│   ├── proxy/
│   ├── session/
│   ├── trace/
│   ├── usage/
│   └── ...
└── tests/
    ├── backend/
    │   ├── session_api_test.go
    │   ├── issue_api_test.go
    │   └── concurrency_test.go
    └── integration/
        └── backend_skill_flow_test.go
```

## 包职责

### `internal/backend`

后端服务装配层。

职责：

1. 定义 `Config`，包含 `Host`、`Port`、`DBPath`、`ClaimTTL` 等。
2. 创建 SQLite store。
3. 创建 service。
4. 创建 HTTP router。
5. 管理 HTTP server 生命周期。

该包不放具体业务规则，业务规则下沉到 `service`。

### `internal/backend/api`

HTTP 接入层。

职责：

1. 注册路由。
2. 解析请求 JSON 和 query 参数。
3. 做基础参数校验。
4. 调用 service。
5. 统一输出 JSON 响应和 HTTP 状态码。

建议路由：

```text
GET  /health
POST /api/session/register
POST /api/session/close
GET  /api/sessions
GET  /api/session/{id}

POST /api/issue/check
POST /api/issue/claim
POST /api/issue/status
POST /api/issue/release
POST /api/issue/release-session
```

### `internal/backend/domain`

领域模型层。

职责：

1. 定义 Session、Machine、Project、IssueClaim 等核心结构。
2. 定义状态枚举。
3. 定义状态判断方法，例如 `IsTerminalIssueStatus`、`CanRelease`。

Issue 状态：

```text
idle
claimed
fixing
ready-for-pr
pr-created
testing
reviewing
merged
rejected
```

Session 状态：

```text
active
closed
```

### `internal/backend/service`

业务编排层。

职责：

1. `SessionService` 负责注册、关闭、查询 session。
2. `IssueService` 负责 check、claim、status、release。
3. `CleanupService` 负责启动时超时清理和未来的 claim TTL 清理。

关键规则放在这里：

1. `claim` 必须原子化，保证同一 `(repo_full_name, issue_number)` 只有一个非 idle 领取者。
2. `status` 只能由当前领取 session 更新。
3. `release` 只能释放当前 session 领取的未终态 issue。
4. `release-session` 不释放 `merged`、`rejected`。
5. 初期可以不强校验完整状态流转，但要集中保留校验入口。

### `internal/backend/store`

持久化层。

职责：

1. SQLite 连接管理。
2. 建表和迁移。
3. 封装事务。
4. 提供面向 service 的接口，不暴露 SQL 细节。

建议接口：

```go
type SessionStore interface {
    RegisterSession(ctx context.Context, s domain.Session, m domain.Machine, p domain.Project) error
    CloseSession(ctx context.Context, sessionID string, reason string) error
    ListSessions(ctx context.Context, filter SessionFilter) ([]domain.Session, error)
    GetSession(ctx context.Context, sessionID string) (domain.Session, error)
}

type IssueStore interface {
    CheckIssues(ctx context.Context, repo string, numbers []int) ([]domain.IssueClaim, error)
    ClaimIssue(ctx context.Context, claim domain.IssueClaim) (domain.IssueClaim, error)
    UpdateIssueStatus(ctx context.Context, repo string, number int, sessionID string, status domain.IssueStatus) (previous domain.IssueStatus, err error)
    ReleaseIssue(ctx context.Context, repo string, number int, sessionID string) (bool, error)
    ReleaseSessionIssues(ctx context.Context, sessionID string) ([]int, error)
}
```

## 数据库结构归属

### Session 表

归属：`internal/backend/store/session_store.go`

表：

```text
machines
projects
sessions
```

来源需求：

1. SessionStart hook 注册。
2. SessionEnd hook 注销。
3. 后端保存机器、项目、trace 路径关联。

### Issue 表

归属：`internal/backend/store/issue_store.go`

表：

```text
issue_claims
```

唯一键：

```text
UNIQUE(repo_full_name, issue_number)
```

关键索引：

```text
idx_issue_claims_repo
idx_issue_claims_session
idx_issue_claims_status
```

## 与现有包的关系

### `internal/proxy`

保持当前职责：反向代理 Claude API 请求。

后端不应依赖 `proxy`。如果未来要在代理启动时写 `.proxy.json` 或注册 backend，只在 CLI 装配层调用 backend client，不让 `proxy` 包直接知道后端。

### `internal/trace`

保持当前职责：trace 路径生成和 JSONL 写入。

需要为 SR-1 调整路径时，优先在 `trace` 包内部完成。后端只接收最终 `trace_path` 字符串。

### `internal/session`

当前已有 session push/pull/status 本地同步能力。不要把新后端 Session 管理直接塞进现有 `internal/session`，否则本地文件同步和后端会话生命周期会混在一起。

建议：

1. `internal/session` 继续代表本地 Claude session 文件同步。
2. `internal/backend/domain.Session` 代表后端注册会话。

如果后续发现模型重复，再抽 `internal/sessionmeta` 之类的共享小包。

### `internal/config`

可复用现有命令解析、Claude client 配置能力。

后端自己的配置放在 `internal/backend/config.go`，避免污染 Claude 代理配置。

## 技能与 Hook 包结构

Go 后端之外，还需要脚本侧结构稳定。以下为当前实际结构，仅标注需要新增/修改的部分。

### 现有 Hooks 结构（29 个事件目录）

```text
.claude/hooks/
├── base.sh                           # 共享基础：JSON 解析、日志、skill 分发
├── platform.sh                       # 平台检测 + Python 路径解析
├── json_get.py                       # Python fallback JSON 解析
├── logs/                             # 按日期日志 hooks/logs/YYYY-MM-DD.log
│
├── 01-session-start/
│   └── base.sh                       # 【修改】新增 backend register 调用
├── 02-setup/
│   └── base.sh
├── 03-user-prompt-submit/
│   └── skill-inject.sh               # Skill 注入触发器
├── 04-user-prompt-expansion/
│   └── base.sh
├── 05-pre-tool-use/
│   └── base.sh                       # enforce_boundary + skill dispatch
├── 06-permission-request/
│   ├── base.sh
│   └── win32-foreground.sh
├── 07-permission-denied/
│   └── base.sh
├── 08-post-tool-use/
│   └── base.sh
├── 09-post-tool-use-failure/
│   └── base.sh
├── 10-post-tool-batch/
│   └── base.sh
├── 11-notification/
│   └── base.sh
├── 12-subagent-start/
│   └── base.sh
├── 13-subagent-stop/
│   └── base.sh
├── 14-task-created/
│   └── base.sh
├── 15-task-completed/
│   └── base.sh
├── 16-stop/
│   ├── base.sh
│   ├── skill-register.sh             # Skill 自动注册
│   └── task-complete-notify.sh
├── 17-stop-failure/
│   └── base.sh
├── 18-teammate-idle/
│   └── base.sh
├── 19-instructions-loaded/
│   └── base.sh
├── 20-config-change/
│   └── base.sh
├── 21-cwd-changed/
│   └── base.sh
├── 22-file-changed/
│   └── base.sh
├── 23-worktree-create/
│   └── base.sh
├── 24-worktree-remove/
│   └── base.sh
├── 25-pre-compact/
│   └── base.sh
├── 26-post-compact/
│   └── base.sh
├── 27-elicitation/
│   └── base.sh
├── 28-elicitation-result/
│   └── base.sh
└── 29-session-end/
    └── base.sh                       # 【修改】新增 backend release-session + close 调用
```

### 现有 Skills 结构（17 个 skill）

```text
.claude/skills/
├── active.sh                         # .active 文件 CRUD（session_id ↔ skill_name 映射）
├── enforce_boundary.sh               # 工具边界校验（解析 SKILL.md allowed-tools）
├── log.sh                            # 双写日志：统一日志 + 模块日志
├── lock.sh                           # 跨平台文件锁（mkdir 原子操作）
├── registry.conf                     # Skill 注册表
├── .active                           # 运行时 session-skill 映射文件
│
├── 001-testcode-python/
│   ├── SKILL.md
│   └── scripts/{init,init_check,03UserPromptSubmit,16Stop}.sh
├── 002-otherdoc/
│   ├── SKILL.md
│   └── scripts/{init,init_check,03UserPromptSubmit,16Stop}.sh
│
├── 003-1-issue-init/                 # 【无需修改】标签初始化（一次性）
│   ├── SKILL.md
│   ├── labels.conf
│   └── scripts/{init,init_check,03UserPromptSubmit,16Stop}.sh
├── 003-2-issue/                      # 【无需修改】创建 issue
│   ├── SKILL.md
│   └── scripts/{init_check,03UserPromptSubmit,16Stop}.sh
├── 003-3-issue-discuss/              # 【无需修改】讨论 issue
│   ├── SKILL.md
│   └── scripts/{init_check,03UserPromptSubmit,16Stop}.sh
│
├── 003-4-issue-claim/                # 【修改】新增 backend.sh 集成
│   ├── SKILL.md
│   └── scripts/{init_check,03UserPromptSubmit,16Stop}.sh
├── 003-5-issue-fix/                  # 【修改】新增 backend.sh 集成
│   ├── SKILL.md
│   └── scripts/{init_check,03UserPromptSubmit,16Stop}.sh
├── 003-6-issue-done/                 # 【修改】新增 backend.sh 集成
│   ├── SKILL.md
│   └── scripts/{init_check,03UserPromptSubmit,16Stop}.sh
├── 003-7-issue-pr/                   # 【修改】新增 backend.sh 集成
│   ├── SKILL.md
│   └── scripts/{init_check,03UserPromptSubmit,16Stop}.sh
├── 003-8-issue-test/                 # 【修改】新增 backend.sh 集成
│   ├── SKILL.md
│   └── scripts/{init_check,03UserPromptSubmit,16Stop}.sh
├── 003-9-issue-review/               # 【修改】新增 backend.sh 集成
│   ├── SKILL.md
│   └── scripts/{init_check,03UserPromptSubmit,16Stop}.sh
│
├── 004-git-push/
│   ├── SKILL.md
│   └── scripts/{init,init_check,03UserPromptSubmit,16Stop}.sh
├── 005-git-commit/
│   ├── SKILL.md
│   └── scripts/{init,init_check,03UserPromptSubmit,16Stop}.sh
└── 999-other-110-requirement-planning/
    ├── SKILL.md
    └── references/{interview-guide,templates}.md
```

### 新增文件

```text
.claude/
├── backend.conf                      # 【新增】后端连接配置（host/port/db_path）
└── skills/
    └── backend.sh                    # 【新增】后端 API 调用封装（curl wrapper）
```

### 改动点汇总

| 位置 | 改动 | 说明 |
|------|------|------|
| `hooks/01-session-start/base.sh` | 修改 | 会话启动时 curl `/api/session/register` |
| `hooks/29-session-end/base.sh` | 修改 | 会话结束时先 curl `/api/issue/release-session`，再 curl `/api/session/close` |
| `skills/backend.sh` | 新增 | 统一后端调用封装：`backend_call "POST" "/api/issue/claim" '{"repo":"...","issue_number":5}'` |
| `.claude/backend.conf` | 新增 | `BACKEND_URL=http://127.0.0.1:8080` |
| `skills/003-4-issue-claim/scripts/03UserPromptSubmit.sh` | 修改 | claim 时调用 `backend.sh` 做 backend 状态检查 + 写入 |
| `skills/003-5-issue-fix/scripts/03UserPromptSubmit.sh` | 修改 | fix 时调用 `backend.sh` 更新 issue status → `fixing` |
| `skills/003-6-issue-done/scripts/03UserPromptSubmit.sh` | 修改 | done 时调用 `backend.sh` 更新 issue status → `ready-for-pr` |
| `skills/003-7-issue-pr/scripts/03UserPromptSubmit.sh` | 修改 | pr 时调用 `backend.sh` 更新 issue status → `pr-created` |
| `skills/003-8-issue-test/scripts/03UserPromptSubmit.sh` | 修改 | test 时调用 `backend.sh` 更新 issue status → `testing` |
| `skills/003-9-issue-review/scripts/03UserPromptSubmit.sh` | 修改 | review 时调用 `backend.sh` 更新 issue status → `reviewing`/`merged`/`rejected` |

### 脚本调用约束

1. 003-4 到 003-9 统一 `source "$CLAUDE_PROJECT_DIR/.claude/skills/backend.sh"`。
2. `backend.sh` 从 `backend.conf` 读取 `BACKEND_URL`，封装 `backend_call(method, path, body)` 函数。
3. SessionEnd hook（事件 29）独立运行，直接 curl `/api/issue/release-session` 和 `/api/session/close`，不依赖 active skill。
4. `claim` 后端不可用时应阻止领取或明确提示，避免多 Agent 冲突。
5. `status`、`release-session` 后端不可用时静默降级，不阻断 GitHub 主流程。
6. 共享模块 `active.sh`、`log.sh`、`lock.sh`、`enforce_boundary.sh` 不需要修改。

## API 到包映射

| API | Handler | Service | Store |
|-----|---------|---------|-------|
| `GET /health` | `health_handler.go` | 无或 `Server` | 无 |
| `POST /api/session/register` | `session_handler.go` | `SessionService.Register` | `SessionStore.RegisterSession` |
| `POST /api/session/close` | `session_handler.go` | `SessionService.Close` | `SessionStore.CloseSession` |
| `GET /api/sessions` | `session_handler.go` | `SessionService.List` | `SessionStore.ListSessions` |
| `GET /api/session/{id}` | `session_handler.go` | `SessionService.Get` | `SessionStore.GetSession` |
| `POST /api/issue/check` | `issue_handler.go` | `IssueService.Check` | `IssueStore.CheckIssues` |
| `POST /api/issue/claim` | `issue_handler.go` | `IssueService.Claim` | `IssueStore.ClaimIssue` |
| `POST /api/issue/status` | `issue_handler.go` | `IssueService.UpdateStatus` | `IssueStore.UpdateIssueStatus` |
| `POST /api/issue/release` | `issue_handler.go` | `IssueService.Release` | `IssueStore.ReleaseIssue` |
| `POST /api/issue/release-session` | `issue_handler.go` | `IssueService.ReleaseSession` | `IssueStore.ReleaseSessionIssues` |

## 依赖方向

只允许下面方向：

```text
cmd/claude-tap
  -> internal/backend
  -> internal/backend/api
  -> internal/backend/service
  -> internal/backend/store
  -> internal/backend/domain
```

`domain` 不依赖任何业务包。

`store` 可以依赖 `domain`，不依赖 `api`。

`service` 可以依赖 `domain` 和 `store` 接口，不依赖 SQLite 具体实现。

`api` 可以依赖 `service` 和 `domain`，不写 SQL。

## 测试包结构

建议测试分两层：

```text
claude_tap_plus/tests/
├── backend/                          # 后端单元测试
│   ├── session_api_test.go           # session register/close/list/get
│   ├── issue_api_test.go             # issue check/claim/status/release
│   └── concurrency_test.go           # 并发 claim、竞态条件
└── integration/                      # 端到端集成测试
    └── backend_skill_flow_test.go    # backend ↔ skill 脚本联动验证
```

重点测试：

1. SQLite 自动建表。
2. session register/close 幂等和错误码。
3. issue check 对未知 issue 返回 `idle`。
4. 并发 claim 同一 issue 只有一个成功。
5. 非 owner session 无法 status/release。
6. release-session 不释放 `merged`、`rejected`。
7. backend 不可用时脚本侧降级行为单独做 shell 测试。

## 分阶段落地顺序

### Phase 1: 后端骨架

1. 新增 `backend` 子命令。
2. 新增 `internal/backend` 基础包。
3. 实现 `/health`。
4. 实现 SQLite 连接、迁移。

### Phase 2: Session 管理

1. 实现 `machines`、`projects`、`sessions`。
2. 实现 register/close/list/get API。
3. 接 SessionStart 和 SessionEnd hook。

### Phase 3: Issue 管理

1. 实现 `issue_claims`。
2. 实现 check/claim/status/release/release-session。
3. 增加并发 claim 测试。

### Phase 4: 技能集成

1. 新增 `.claude/skills/backend.sh`。
2. 改造 003-4 到 003-9。
3. SessionEnd hook 先 release issue，再 close session。

### Phase 5: 代理与 trace 补强

1. 调整 trace 路径结构。
2. 写 `.proxy.json` 供 hook 查找 backend/proxy 状态。
3. 完成端到端验证。

## 关键取舍

1. 后端 Session 管理不复用现有 `internal/session` 包，避免本地 session 文件同步和后端生命周期混杂。
2. 初期使用 `net/http` 足够，不必引入 gin；如果后续路由复杂再引入框架。
3. 单二进制新增 `backend` 子命令优先于新增 server 二进制，符合当前 CLI 结构和文档启动方式。
4. Issue 状态流转规则集中在 service 层，初期可弱校验，但接口要预留强校验能力。
5. GitHub 仍是 issue 内容和评论的权威来源，SQLite 只保存协作锁和状态索引。
