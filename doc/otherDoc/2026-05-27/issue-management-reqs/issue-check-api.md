# B2: 批量状态查询 API

> 创建时间：2026-05-27
> 模块：claude-tap-plus / Issue 管理
> 简述：实现 `/api/issue/check` 接口，支持批量查询 issue 状态并过滤已被领取的 issue

---

## 需求描述

提供一个 API 端点，让 Agent 脚本传入一组 issue 编号，返回每个 issue 的当前状态。主要用于 `003-4-issue-claim` 技能中过滤已被其他 session 领取的 issue。

## API 定义

**端点**: `POST /api/issue/check`

### 请求

```json
{
  "repo_full_name": "xiaoheiDTF/claude-hk",
  "issue_numbers": [9, 10, 11, 12]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| repo_full_name | string | 是 | 仓库全名 |
| issue_numbers | int[] | 是 | 要查询的 issue 编号列表 |

### 响应

```json
{
  "issues": [
    {"number": 9,  "status": "idle",    "session_id": null,         "claimed_at": null},
    {"number": 10, "status": "claimed", "session_id": "session_abc", "claimed_at": "2026-05-26T10:00:00Z"},
    {"number": 11, "status": "merged",  "session_id": null,         "claimed_at": null},
    {"number": 12, "status": "idle",    "session_id": null,         "claimed_at": null}
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| issues[].number | int | Issue 编号 |
| issues[].status | string | 当前状态 |
| issues[].session_id | string? | 领取者的 session_id |
| issues[].claimed_at | string? | 领取时间（ISO 8601） |

### 错误响应

```json
{"error": "invalid_request", "message": "issue_numbers is required"}
```

## 业务逻辑

```
1. 校验请求参数（repo_full_name、issue_numbers 必填）
2. 对每个 issue_number：
   a. 查询 issue_claims 表
   b. 如果不存在 → 自动 INSERT OR IGNORE（首次出现，状态 idle）
   c. 返回当前状态
3. 组装响应返回
```

## 技能脚本调用示例

```bash
# 获取 GitHub open issues
gh_issues=$(gh issue list --state open --json number,title,labels)
numbers=$(echo "$gh_issues" | jq '[.[].number]')

# 调后端查询
repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner')
backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)

check_result=$(curl -s -X POST "$backend_url/api/issue/check" \
  -H "Content-Type: application/json" \
  -d "{\"repo_full_name\":\"$repo\",\"issue_numbers\":$numbers}")

# 过滤出 idle 的 issue
idle_numbers=$(echo "$check_result" | jq '[.issues[] | select(.status == "idle") | .number]')
echo "$gh_issues" | jq --argjson idle "$idle_numbers" '[.[] | select(.number | IN($idle[]))]'
```

## 验收标准

- [x] 传入多个 issue 编号，返回每个的状态
- [x] 首次出现的 issue 自动创建记录，状态为 idle
- [x] 已被领取的 issue 返回 claimed 及 session_id
- [x] 已合并的 issue 返回 merged
- [x] 参数缺失时返回错误响应
- [x] 空 issue_numbers 数组返回空列表
