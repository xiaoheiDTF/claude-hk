# B1: 数据库表与数据模型

> 创建时间：2026-05-27
> 模块：claude-tap-plus / Issue 管理
> 简述：定义 issue_claims 表结构、字段来源、索引及核心查询

---

## 需求描述

创建 SQLite 数据库表，存储 issue 领取关系。只存"哪个 session 领取了哪个 issue"及其状态，不存 issue 完整内容。

## 表结构

```sql
CREATE TABLE issue_claims (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_full_name  TEXT NOT NULL,                  -- xiaoheiDTF/claude-hk
    issue_number    INTEGER NOT NULL,               -- GitHub issue 编号
    issue_title     TEXT,                           -- issue 标题（缓存）
    status          TEXT NOT NULL DEFAULT 'idle',   -- 见状态枚举
    session_id      TEXT,                           -- 领取者 session_id，idle 时为 null
    claimed_at      DATETIME,                       -- 领取时间
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(repo_full_name, issue_number)
);

CREATE INDEX idx_issue_claims_repo ON issue_claims(repo_full_name);
CREATE INDEX idx_issue_claims_session ON issue_claims(session_id);
CREATE INDEX idx_issue_claims_status ON issue_claims(status);
```

## 状态枚举

| 状态 | 说明 | GitHub Label |
|------|------|-------------|
| `idle` | 空闲，无人领取 | 无 |
| `claimed` | 已被某 session 领取 | `in-progress` |
| `fixing` | 正在开发中 | `fixing` |
| `ready-for-pr` | 开发完成，等待提 PR | `ready-for-pr` |
| `pr-created` | PR 已创建 | `pr-created` |
| `testing` | 测试中 | `testing` |
| `reviewing` | 审核中 | `reviewing` |
| `merged` | 已合并（终态） | 关闭 issue |
| `rejected` | 被打回 | `rejected` |

> GitHub Label 需要在 `003-1-issue-init` 的 `labels.conf` 中定义：`in-progress`、`fixing`、`ready-for-pr`、`pr-created`、`testing`、`reviewing`、`rejected`。

## 字段来源

| 字段 | 来源 | 获取方式 |
|------|------|----------|
| repo_full_name | gh repo view | `gh repo view --json nameWithOwner --jq '.nameWithOwner'` |
| issue_number | gh issue list/view | `.number` |
| issue_title | gh issue view | `.title` |
| status | 后端维护 | 根据 API 调用更新 |
| session_id | 请求体传入 | API 调用方提供 |
| claimed_at | 后端生成 | 领取时 CURRENT_TIMESTAMP |

## ER 关系

```
┌─────────────────┐         ┌─────────────────┐
│  sessions (C)   │         │  issue_claims   │
├─────────────────┤         ├─────────────────┤
│ session_id (PK) │◄────────│ session_id (FK) │
│ machine_id      │   1:N   │ repo_full_name  │
│ project_slug    │         │ issue_number    │
└─────────────────┘         │ status          │
                            └─────────────────┘
```

> sessions 表在模块 C（session-design.md）中定义。

## 核心查询

```sql
-- 查询某仓库所有 issue 状态
SELECT issue_number, issue_title, status, session_id, claimed_at
FROM issue_claims WHERE repo_full_name = 'xiaoheiDTF/claude-hk';

-- 查询某 session 领取的所有 issue
SELECT issue_number, repo_full_name, status, claimed_at
FROM issue_claims
WHERE session_id = 'xxx' AND status != 'idle';

-- 批量查询指定 issue 状态（check API 用）
SELECT issue_number, status, session_id, claimed_at
FROM issue_claims
WHERE repo_full_name = 'xiaoheiDTF/claude-hk' AND issue_number IN (9, 10, 11, 12);

-- 释放某 session 的所有 issue（SessionEnd 用）
UPDATE issue_claims
SET status = 'idle', session_id = NULL, claimed_at = NULL
WHERE session_id = 'xxx' AND status NOT IN ('merged', 'rejected');

-- 统计某仓库各状态数量
SELECT status, COUNT(*) FROM issue_claims
WHERE repo_full_name = 'xiaoheiDTF/claude-hk' GROUP BY status;
```

## 写入时序

| 时机 | 操作 |
|------|------|
| 首次 check/claim | INSERT OR IGNORE（issue 首次出现，状态 idle） |
| claim 成功 | UPDATE status='claimed', session_id=xxx |
| 状态流转 | UPDATE status=新状态 |
| release | UPDATE status='idle', session_id=NULL |
| merge | UPDATE status='merged'（保持，不清除） |

## 验收标准

- [ ] SQLite 表创建成功，UNIQUE 约束生效
- [ ] 三个索引存在
- [ ] INSERT OR IGNORE 对重复 (repo, number) 不报错
- [ ] 所有核心查询可正确执行
