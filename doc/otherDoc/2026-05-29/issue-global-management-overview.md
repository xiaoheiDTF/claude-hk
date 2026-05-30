# Issue 全局管理 — 总览与架构

> 创建时间：2026-05-29
> 模块：claude-tap-plus / 模块 B + 模块 D
> 简述：Issue 全局管理架构总览，汇总已实现的全部组件、API、数据模型和技能集成状态

---

## 一、设计目标

为单 GitHub 账号多 Agent 场景提供 Issue 领取去重与状态流转机制。后端中心化服务解决并发冲突，技能脚本通过 backend.sh 统一调用。

**核心原则：**

1. GitHub 是数据源，后端是锁
2. session 绑定，非用户绑定（同一账号多 Agent 可并行）
3. 状态流转由技能触发，后端不主动感知 GitHub
4. 会话关闭自动释放（SessionEnd hook）

---

## 二、已实现组件清单

### 2.1 后端 API（5 个 Issue 端点 + 健康检查）

| 方法 | 路径 | 实现文件 | 状态 |
|------|------|----------|------|
| GET | `/health` | `api/health_handler.go` | ✅ |
| POST | `/api/issue/check` | `api/issue_handler.go` | ✅ |
| POST | `/api/issue/claim` | `api/issue_handler.go` | ✅ |
| POST | `/api/issue/status` | `api/issue_handler.go` | ✅ |
| POST | `/api/issue/release` | `api/issue_handler.go` | ✅ |
| POST | `/api/issue/release-session` | `api/issue_handler.go` | ✅ |

### 2.2 分层架构

```
api/issue_handler.go   → HTTP 请求解析、参数校验、响应输出
       ↓
service/issue_service.go → 业务编排（透传 store 操作）
       ↓
store/issue_store.go    → SQLite 持久化（原子领取、乐观锁）
       ↓
store/migrations.go     → 建表（issue_claims + 4 张会话管理表）
```

### 2.3 数据模型

**issue_claims 表（唯一键：repo_full_name + issue_number）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| repo_full_name | TEXT | xiaoheiDTF/claude-hk |
| issue_number | INTEGER | GitHub issue 编号 |
| issue_title | TEXT | 标题缓存 |
| status | TEXT | idle/claimed/fixing/ready-for-pr/pr-created/testing/reviewing/merged/rejected |
| session_id | TEXT | 领取者 session（idle 时 NULL） |
| claimed_at | DATETIME | 领取时间 |
| updated_at | DATETIME | 最后更新 |

### 2.4 技能集成

| 技能 | 后端调用 | 状态 |
|------|----------|------|
| 003-4-issue-claim | `_call_backend /api/issue/claim` + `check_issue_status()` | ✅ |
| 003-5-issue-fix | `update_issue_status("fixing")` | ✅ |
| 003-6-issue-done | `update_issue_status("ready-for-pr")` | ✅ |
| 003-7-issue-pr | `update_issue_status("pr-created")` | ✅ |
| 003-8-issue-test | `update_issue_status("testing")` | ✅ |
| 003-9-issue-review | `update_issue_status("reviewing"/"merged"/"rejected")` | ✅ |
| SessionEnd hook | `_call_backend /api/issue/release-session` | ✅ |

### 2.5 公共基础设施

| 组件 | 文件 | 说明 |
|------|------|------|
| backend.sh | `.claude/skills/backend.sh` | 统一后端调用封装（`_backend_available`、`_call_backend`、`update_issue_status`） |
| backend.conf | `.claude/backend.conf` | `BACKEND_URL=http://127.0.0.1:8080` |

---

## 三、状态流转

```
idle ──claim──→ claimed ──fix──→ fixing ──done──→ ready-for-pr ──pr──→ pr-created
                                                                        │
                                                                   ──test──→ testing
                                                                        │
                                                                  ──review──→ reviewing
                                                                        │
                                                    ┌──────────merge──────────┐
                                                    ▼                         ▼
                                                 merged                   rejected
                                                 (终态)              (SessionEnd 释放为 idle)
```

**关键规则：**
- claim 原子化：同一 (repo, issue_number) 只有一个非 idle 领取者
- status 只能由当前 owner session 更新
- release-session 不释放 merged/rejected
- 后端不可用时 claim 阻止领取，status/release 静默降级

---

## 四、测试覆盖

| 测试文件 | 覆盖范围 |
|----------|----------|
| `tests/backend/issue_api_test.go` | check/claim/status/release/release-session 全部端到端流程（含并发、权限、终态保护） |
| `tests/backend/session_api_test.go` | session register/close/list/get（25 个子测试） |

---

## 五、数据流

```
/003-4-issue-claim
  → gh issue list → POST /api/issue/check 过滤 idle → 用户选择
  → POST /api/issue/claim 原子领取 → gh issue edit --add-assignee

/003-5 ~ /003-9
  → POST /api/issue/status 更新后端状态 → 对应 gh 操作

SessionEnd hook
  → POST /api/issue/release-session 自动释放
```

---

## 六、需求完成情况

| 需求编号 | 需求名称 | 状态 |
|----------|----------|------|
| B1 | 数据库表与数据模型 | ✅ migrations.go |
| B2 | 批量状态查询 API | ✅ /api/issue/check |
| B3 | 原子领取 API | ✅ /api/issue/claim |
| B4 | 状态流转 API | ✅ /api/issue/status |
| B5 | 释放机制 API | ✅ /api/issue/release + release-session |
| B6-1 | 003-4-issue-claim 改造 | ✅ claim_issue_backend() |
| B6-2 | 003-5 ~ 003-9 改造 | ✅ update_issue_status() |
| B6-3 | SessionEnd Hook 改造 | ✅ release_session_issues() |
| B7 | 公共后端调用模块 | ✅ backend.sh |
| D0 | 公共后端调用基础设施 | ✅ _call_backend / _backend_available |
| D1 ~ D7 | 技能集成 | ✅ 全部完成 |

---

## 七、原始设计文档索引

| 文档 | 位置 |
|------|------|
| Issue 全局管理设计 | `doc/otherDoc/2026-05-26/claude-tap-plus-issue-design.md` |
| 后端服务设计 | `doc/otherDoc/2026-05-26/claude-tap-plus-backend-design.md` |
| 架构与包结构设计 | `doc/otherDoc/2026-05-27/claude-tap-plus-architecture-package-design.md` |
| 数据库 Schema | `doc/otherDoc/2026-05-28/claude-tap-plus-database-schema.md` |
