# B4: 状态流转 API

> 创建时间：2026-05-27
> 模块：claude-tap-plus / Issue 管理
> 简述：实现 `/api/issue/status` 接口，支持技能脚本驱动 issue 状态在各阶段间流转

---

## 需求描述

提供状态更新接口，供 003-5 至 003-9 各技能在对应阶段调用，更新 issue 在后端的状态。后端只记录状态变更，不主动操作 GitHub。

## API 定义

**端点**: `POST /api/issue/status`

### 请求

```json
{
  "repo_full_name": "xiaoheiDTF/claude-hk",
  "issue_number": 10,
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "status": "fixing"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| repo_full_name | string | 是 | 仓库全名 |
| issue_number | int | 是 | Issue 编号 |
| session_id | string | 是 | 当前会话 ID |
| status | string | 是 | 目标状态 |

### 成功响应

```json
{"success": true, "status": "fixing", "previous_status": "claimed", "updated_at": "2026-05-26T11:00:00Z"}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| success | bool | 是否成功 |
| status | string | 更新后的新状态 |
| previous_status | string | 更新前的旧状态（用于同步 GitHub label） |
| updated_at | string | 更新时间（ISO 8601） |

`previous_status` 用于技能脚本调用 `_sync_github_label()` 时知道该移除哪个旧 GitHub label。

### 失败响应（无权限）

```json
{"success": false, "error": "not_owner", "message": "Issue is owned by session_abc"}
```

### 失败响应（非法流转）

```json
{"success": false, "error": "invalid_transition", "message": "Cannot transition from merged to fixing"}
```

## 各技能调用映射

`update_issue_status()` 内部自动完成双写：调后端 API + 同步 GitHub label。

| 技能 | 调用时机 | status 值 | GitHub Label 变更（自动） |
|------|----------|-----------|--------------------------|
| 003-5-issue-fix | 创建分支后 | `fixing` | 移除 `in-progress`，添加 `fixing` |
| 003-6-issue-done | 开发完成后 | `ready-for-pr` | 移除 `fixing`，添加 `ready-for-pr` |
| 003-7-issue-pr | PR 创建后 | `pr-created` | 移除 `ready-for-pr`，添加 `pr-created` |
| 003-8-issue-test | 开始测试 | `testing` | 移除 `pr-created`，添加 `testing` |
| 003-9-issue-review | 审核开始 | `reviewing` | 移除 `testing`，添加 `reviewing` |
| 003-9-issue-review merge | 合并后 | `merged` | 移除 `reviewing`，关闭 issue |
| 003-9-issue-review reject | 打回后 | `rejected` | 移除 `reviewing`，添加 `rejected` |

> 技能脚本只需调用 `update_issue_status "$ISSUE_NUM" "fixing"`，不需要手动执行 `gh issue edit` label 操作。

## 合法状态流转

```
idle → claimed        (仅通过 claim API)
claimed → fixing
fixing → ready-for-pr
fixing → claimed      (reject 回退)
ready-for-pr → pr-created
pr-created → testing
testing → reviewing
reviewing → merged    (终态)
reviewing → fixing    (reject 回退)
```

> **注意**：是否严格校验流转路径可选。初期可不校验，仅记录状态；后期按需加入校验。

## 业务逻辑

```
1. 校验请求参数
2. 查询 issue_claims 表，记录当前 status 作为 previous_status
3. 检查权限：session_id 必须是当前领取者（或管理员可强制更新）
4. （可选）校验状态流转合法性
5. UPDATE status 和 updated_at
6. 返回结果（含 previous_status）
```

## 技能脚本调用示例

```bash
# 003-5-issue-fix 中
session_id=$(json_get '.session_id')
repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner')

curl -s -X POST "$backend_url/api/issue/status" \
  -H "Content-Type: application/json" \
  -d "{
    \"repo_full_name\":\"$repo\",
    \"issue_number\":$ISSUE_NUM,
    \"session_id\":\"$session_id\",
    \"status\":\"fixing\"
  }"
```

## 验收标准

- [ ] 合法状态可成功更新
- [ ] 非领取者 session 无法更新状态（返回 not_owner）
- [ ] （可选）非法状态流转返回 invalid_transition
- [ ] updated_at 自动更新
- [ ] 响应中包含 previous_status 字段
- [ ] 不存在的 issue 返回错误
