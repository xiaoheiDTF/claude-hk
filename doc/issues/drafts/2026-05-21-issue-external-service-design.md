---
title: Issue 机制外部辅助服务最小功能设计（个人版）
labels: skill-system,enhancement
assignee: 
priority: P2
status: draft
created: 2026-05-21
parent: #18
---

## 背景

父 Issue #18 已明确：当前 Issue 闭环机制中，**以下任务无法通过 Claude Code Hooks 或 Skill 单独完成**：

| 缺陷 | 无法通过 Hooks 解决的原因 |
|------|------------------------|
| in-progress 超时释放 | Hooks 无 cron 能力，SessionEnd 后不再触发 |
| stale issue 清理 | 同上，需要定时任务 |
| issue 依赖关系管理 | Hooks 无法感知其他 issue 状态变更 |
| 跨会话状态持久化 | Hooks 只能写文件/环境变量，无结构化查询能力 |
| 个人看板/当前任务总览 | Claude Code 无持久化 dashboard |

本 Issue 设计一个**最小化的外部辅助服务**，仅满足**个人开发者**使用，不考虑团队协作、权限管理、高可用等企业级需求。

---

## 设计原则

1. **极简**：只解决 Hooks/Skill 做不了的事，不重复造轮子
2. **本地优先**：优先本地运行，不强制要求公网服务器
3. **零维护成本**：SQLite 替代 Postgres，单文件部署
4. **免费基础设施**：尽量复用 GitHub Actions + GitHub API，减少自建服务负担

---

## 功能模块设计

### 模块一：GitHub Webhook 接收与事件处理

**为什么需要**：Hooks 无法主动感知 GitHub 上的事件（issue 被评论、PR 被 review、label 被修改等）。

**功能点**：
- 接收 GitHub Webhook（`issues`、`pull_request`、`issue_comment`、`label` 事件）
- 解析事件，更新本地 SQLite 状态库
- 事件类型覆盖：
  - `issues.opened/closed/reopened/edited` → 更新 issue 状态
  - `issues.assigned/unassigned` → 更新 assignee
  - `issues.labeled/unlabeled` → 更新标签
  - `issue_comment.created` → 更新 last_activity 时间
  - `pull_request.opened/closed/merged` → 关联 issue 的 PR 状态

**个人简化**：
- 不处理签名验证（个人使用，内网服务）
- 不处理重试/幂等（GitHub 默认重试 3 次，SQLite upsert 即可）

---

### 模块二：定时任务调度

**为什么需要**：Hooks 没有 cron，无法执行超时释放、stale 清理等定时任务。

**功能点**：

| 定时任务 | 频率 | 动作 |
|---------|------|------|
| in-progress 提醒 | 每 6 小时 | 检查 `in-progress` 且 last_activity > 6h 的 issue，输出提醒 |
| in-progress 超时释放 | 每天 00:00 | 检查 `in-progress` 且 last_activity > 7 天的 issue，自动移除 assignee 和标签，发 comment |
| stale 标记 | 每天 00:00 | 检查 open 且 last_activity > 30 天的 issue，加 `stale` 标签 |
| stale 关闭 | 每天 00:00 | 检查 `stale` 且 last_activity > 37 天的 issue，自动关闭 |
| 依赖阻塞检查 | 每天 00:00 | 检查声明了 `depends on #X` 但 #X 未关闭的 issue，加 `blocked` 标签 |
| 依赖解除检查 | 每天 00:00 | 检查 `blocked` 且依赖 issue 已关闭的，移除 `blocked` 标签 |

**个人简化**：
- 使用 Python APScheduler（内置，无需额外依赖）
- 或完全用 GitHub Actions cron 替代，服务本身只提供 HTTP API

---

### 模块三：轻量状态数据库（SQLite）

**为什么需要**：Hooks 只能写文件/环境变量，无法做结构化查询和跨会话关联。

**Schema 设计**：

```sql
-- issue 主表
CREATE TABLE issues (
  id INTEGER PRIMARY KEY,           -- GitHub issue number
  title TEXT,
  state TEXT,                       -- open/closed
  assignee TEXT,                    -- GitHub login
  labels TEXT,                      -- json array
  claim_time TIMESTAMP,             -- 被 claim 的时间
  last_activity TIMESTAMP,          -- 最后活动时间（评论、commit、label 变更）
  pr_number INTEGER,                -- 关联的 PR number
  pr_state TEXT,                    -- open/closed/merged
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);

-- issue 依赖关系
CREATE TABLE issue_dependencies (
  issue_id INTEGER,
  depends_on_id INTEGER,
  PRIMARY KEY (issue_id, depends_on_id)
);

-- 里程碑（简单版）
CREATE TABLE milestones (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT,
  due_date TIMESTAMP,
  description TEXT
);

-- milestone 关联
CREATE TABLE milestone_issues (
  milestone_id INTEGER,
  issue_id INTEGER,
  PRIMARY KEY (milestone_id, issue_id)
);

-- 操作日志（用于审计和恢复）
CREATE TABLE activity_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  issue_id INTEGER,
  action TEXT,                      -- claim/release/stale/close
  actor TEXT,                       -- agent / user
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**个人简化**：
- 单 SQLite 文件，随服务启动创建
- 不需要迁移工具，schema 变更直接改代码

---

### 模块四：HTTP API（供 Skill/Hooks 调用）

**为什么需要**：Skill 需要查询当前状态（谁在做什么、哪些 issue 可领取、依赖链是否通畅）。

**API 设计**：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/issues` | 所有 issue 列表，支持 `?state=open&label=in-progress` 过滤 |
| GET | `/issues/:id` | 单个 issue 详情，包含依赖关系 |
| GET | `/issues/:id/dependencies` | 返回依赖链（递归） |
| GET | `/issues/ready` | 返回可领取的 issue（open + 无 assignee + 无 blocked） |
| POST | `/issues/:id/claim` | 记录 claim 时间，更新 assignee |
| POST | `/issues/:id/release` | 记录释放时间，清除 assignee |
| GET | `/issues/in-progress` | 返回当前进行中的 issue（用于个人看板） |
| GET | `/board` | 简单文本看板，终端友好 |
| POST | `/webhook/github` | GitHub Webhook 接收端点 |

**个人简化**：
- 无认证（内网服务或本地使用）
- 返回格式优先文本/plaintext，方便终端阅读

---

### 模块五：个人看板（终端/极简 Web）

**为什么需要**：Claude Code 没有持久化 dashboard，无法一眼看到所有 issue 状态。

**功能点**：

**终端版（推荐，个人用足够）**：
```bash
$ curl http://localhost:8080/board

📋 Issue 看板 (xiaoheiDTF/claude-hk)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🟡 进行中 (2)
  #15  [P1] 细化 issue PR 阶段流程        assignee: xiaoheiDTF  已活跃 2h
  #18  [P1] 缺陷分析(Hooks版)             assignee: xiaoheiDTF  已活跃 5min

🟢 可领取 (3)
  #16  [P2] 优化模板（待方案确认）
  #17  [P1] 优化模板（已发布）
  #19  [P0] 外部服务设计                  ← 当前

🔴 阻塞中 (1)
  #20  [P1] 某功能实现                    depends on: #15

⚪ 最近关闭 (2)
  #12  [P2] xxx                          closed 2 days ago
```

**Web 版（可选，一个 HTML 文件）**：
- 单文件 HTML + vanilla JS，直接读取 `/issues` API
- 无框架，无构建步骤

---

## 技术选型（个人极简版）

| 层级 | 选型 | 理由 |
|------|------|------|
| 语言 | Python 3.11+ | 标准库丰富，APScheduler + FastAPI 生态成熟 |
| Web 框架 | FastAPI | 自动 API 文档，异步支持，代码量少 |
| 数据库 | SQLite | 零配置，单文件，备份就是 cp |
| 定时任务 | APScheduler | Python 内置能力，无需系统 cron |
| Webhook 暴露 | ngrok / Cloudflare Tunnel | 免费，内网穿透，个人用足够 |
| 部署 | 本地常驻 / pm2 | 个人电脑常驻运行即可 |

**替代方案（更轻量）**：
- 如果完全不想本地常驻服务，可用 **GitHub Actions** 做定时任务，服务只保留 Webhook 接收 + API 查询
- 如果完全不想写服务，可用 **GitHub Projects** 做看板，定时任务用 Actions，依赖关系手动维护

---

## 与现有 Skill 体系的集成点

```
001-4-issue-claim
  → 调用 POST /issues/:id/claim
  → 查询 GET /issues/ready（失败时返回其他可领取 issue）

001-5-issue-fix
  → 查询 GET /issues/:id/dependencies（检查依赖是否已解决）

001-6-issue-pr
  → 查询 GET /issues/:id（确认 issue 当前状态）

Hooks (PreToolUse)
  → 查询 GET /issues/in-progress（阻止 claim 多个 issue）
  → 查询 GET /issues/:id（claim 前检查是否 blocked）

个人日常
  → 访问 /board 或本地网页看板，了解当前任务状态
```

---

## 最简化实现路径（MVP）

### Phase 1：Webhook + SQLite（1~2 天）
- [ ] 搭建 FastAPI 服务，接收 GitHub Webhook
- [ ] SQLite schema 创建
- [ ] issues 表同步（接收 webhook 后 upsert）
- [ ] `/issues`、`/issues/:id`、`/board` API

### Phase 2：定时任务（0.5 天）
- [ ] APScheduler 集成
- [ ] 超时释放逻辑
- [ ] stale 标记/关闭逻辑

### Phase 3：依赖管理（0.5 天）
- [ ] issue_dependencies 表
- [ ] 解析 issue body 中的 `depends on #X` 语法
- [ ] 定时检查阻塞/解除阻塞

### Phase 4：Skill 集成（0.5 天）
- [ ] 001-4-issue-claim 调用外部 API
- [ ] 001-5-issue-fix 检查依赖
- [ ] Hooks 配置中添加查询调用

**总计**：约 3 天可完成 MVP。

---

## 涉及文件

| 文件 | 说明 |
|------|------|
| `issue-service/`（新目录） | 外部服务代码 |
| `issue-service/main.py` | FastAPI 主服务 |
| `issue-service/scheduler.py` | 定时任务 |
| `issue-service/webhook.py` | GitHub Webhook 处理 |
| `issue-service/models.py` | SQLite schema |
| `issue-service/board.html` | 可选：极简 Web 看板 |
| `.github/workflows/webhook-relay.yml` | 可选：如果用 Actions 做定时任务 |

## 验收标准

- [ ] 本地运行服务，能接收 GitHub Webhook 并同步 issue 状态到 SQLite
- [ ] 访问 `/board` 能看到当前进行中的 issue 列表
- [ ] 模拟一个 issue 被 claim 后 7 天无活动，服务自动释放并 comment
- [ ] 001-4-issue-claim 能查询 `/issues/ready` 获取可领取列表
- [ ] 服务部署文档（本地运行 + ngrok 暴露）

## 发布记录

- Issue #19: https://github.com/xiaoheiDTF/claude-hk/issues/19 (发布于 2026-05-21 22:45)

