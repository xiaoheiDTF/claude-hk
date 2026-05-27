# B3: 原子领取 API

> 创建时间：2026-05-27
> 模块：claude-tap-plus / Issue 管理
> 简述：实现 `/api/issue/claim` 接口，支持 session 原子领取 issue，防止并发冲突

---

## 需求描述

提供原子领取接口。Agent 确认要领取某个 issue 时，先调此 API 在后端加锁，成功后再操作 GitHub。如果 issue 已被其他 session 领取，则返回失败。

## API 定义

**端点**: `POST /api/issue/claim`

### 请求

```json
{
  "repo_full_name": "xiaoheiDTF/claude-hk",
  "issue_number": 10,
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "issue_title": "优化 issue 模板"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| repo_full_name | string | 是 | 仓库全名 |
| issue_number | int | 是 | Issue 编号 |
| session_id | string | 是 | 当前会话 ID |
| issue_title | string | 否 | Issue 标题（缓存用） |

### 成功响应

```json
{"success": true, "status": "claimed", "claimed_at": "2026-05-26T10:05:00Z"}
```

### 失败响应（已被领取）

```json
{"success": false, "error": "already_claimed", "claimed_by": "session_abc", "claimed_at": "2026-05-26T10:00:00Z"}
```

### 失败响应（参数缺失）

```json
{"success": false, "error": "invalid_request", "message": "session_id is required"}
```

## 业务逻辑

```
1. 校验请求参数（repo_full_name、issue_number、session_id 必填）
2. 查询 issue_claims 表：
   a. 如果不存在 → INSERT（首次出现）
3. 检查当前状态：
   a. 如果是 idle → UPDATE status='claimed', session_id=请求值, claimed_at=NOW → 返回成功
   b. 如果已被同一 session_id 领取 → 返回成功（幂等）
   c. 如果已被其他 session 领取 → 返回 already_claimed
   d. 如果状态是 merged/rejected → 返回 already_claimed
4. 返回结果
```

**关键**：第 3 步需要使用事务或行锁保证原子性：

```sql
-- SQLite 方案：利用 WHERE 条件实现乐观锁
UPDATE issue_claims
SET status = 'claimed', session_id = ?, claimed_at = CURRENT_TIMESTAMP
WHERE repo_full_name = ? AND issue_number = ? AND status = 'idle';

-- 检查 affected rows：
--   0 → 可能已被领取，查询当前状态返回失败
--   1 → 领取成功
```

## 技能脚本调用示例

```bash
session_id=$(json_get '.session_id')
repo=$(gh repo view --json nameWithOwner --jq '.nameWithOwner')
issue_title=$(gh issue view 10 --json title --jq '.title')

# 先调后端领取
claim_result=$(curl -s -X POST "$backend_url/api/issue/claim" \
  -H "Content-Type: application/json" \
  -d "{
    \"repo_full_name\":\"$repo\",
    \"issue_number\":10,
    \"session_id\":\"$session_id\",
    \"issue_title\":\"$issue_title\"
  }")

if echo "$claim_result" | jq -e '.success' > /dev/null; then
  # 后端成功，再操作 GitHub
  gh issue edit 10 --add-assignee @me --add-label "in-progress"
else
  claimed_by=$(echo "$claim_result" | jq -r '.claimed_by')
  echo "Issue #10 已被 $claimed_by 领取"
fi
```

## 冲突处理场景

```
T1: Agent A → POST /api/issue/claim (#10, session_A)
    后端: idle → claimed, session=session_A
    响应: {"success": true}

T2: Agent B → POST /api/issue/claim (#10, session_B)
    后端: 已 claimed, session=session_A ≠ session_B
    响应: {"success": false, "error": "already_claimed", "claimed_by": "session_A"}
```

## 验收标准

- [ ] 空闲 issue 可被成功领取
- [ ] 已被其他 session 领取的 issue 返回 already_claimed
- [ ] 同一 session 重复领取同一 issue 返回成功（幂等）
- [ ] 已合并/打回的 issue 不可领取
- [ ] 已打回的 issue 释放（status 恢复 idle）后可被重新领取
- [ ] 并发领取时只有一个成功（原子性）
- [ ] 首次出现的 issue 自动创建记录后领取
