# B5: 释放机制 API

> 创建时间：2026-05-27
> 模块：claude-tap-plus / Issue 管理
> 简述：实现 `/api/issue/release` 和 `/api/issue/release-session` 接口，支持手动释放和 SessionEnd 自动释放

---

## 需求描述

提供两种释放机制：
1. **单个释放**：手动放弃某个 issue 时调用
2. **批量释放**：SessionEnd hook 触发时，自动释放该 session 领取的所有 issue

## API 1: 单个释放

**端点**: `POST /api/issue/release`

### 请求

```json
{
  "repo_full_name": "xiaoheiDTF/claude-hk",
  "issue_number": 10,
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| repo_full_name | string | 是 | 仓库全名 |
| issue_number | int | 是 | Issue 编号 |
| session_id | string | 是 | 当前会话 ID |

### 成功响应

```json
{"success": true, "released": true}
```

### 失败响应（非领取者）

```json
{"success": false, "error": "not_owner"}
```

### 业务逻辑

```
1. 校验参数
2. 查询当前领取者
3. 检查 session_id 是否是当前领取者
4. UPDATE status='idle', session_id=NULL, claimed_at=NULL
   WHERE status NOT IN ('merged', 'rejected')
5. 返回结果
```

---

## API 2: Session 批量释放

**端点**: `POST /api/issue/release-session`

### 请求

```json
{"session_id": "bf15cac4-7235-48ce-8853-5d4598547f31"}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| session_id | string | 是 | 要释放的会话 ID |

### 响应

```json
{"released": [10, 12], "count": 2}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| released | int[] | 被释放的 issue 编号列表 |
| count | int | 释放数量 |

### 业务逻辑

```
1. 校验参数
2. 查询该 session_id 领取的所有 issue
3. UPDATE status='idle', session_id=NULL, claimed_at=NULL
   WHERE session_id=? AND status NOT IN ('merged', 'rejected')
4. 返回被释放的 issue 编号列表
```

**注意**：`merged` 和 `rejected` 是终态，不释放。

## SessionEnd hook 调用示例

```bash
# .claude/hooks/29-session-end/base.sh 末尾
release_session_issues() {
  local session_id=$(json_get '.session_id')
  local backend_url
  backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)
  [ -z "$backend_url" ] && return 0

  local result
  result=$(curl -s -X POST "$backend_url/api/issue/release-session" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$session_id\"}")

  local count
  count=$(echo "$result" | jq -r '.count // 0')
  [ "$count" -gt 0 ] && log "INFO" "Released $count issues for session $session_id"
}

release_session_issues
```

## 异常处理：Agent 崩溃未触发 SessionEnd

```
场景：Agent A 领取 #10 后崩溃，SessionEnd hook 未触发

解决方案（按优先级）：
1. 后端设置 claim 超时（如 24 小时），超时自动释放
2. 管理员手动调 POST /api/issue/release 释放
3. 其他 Agent 联系管理员确认后强制释放
```

## 验收标准

- [x] 单个释放：领取者可释放自己的 issue
- [x] 单个释放：非领取者无法释放（返回 not_owner）
- [x] 单个释放：merged/rejected 状态的 issue 不被释放
- [x] 批量释放：释放该 session 所有未终态的 issue
- [x] 批量释放：已合并/打回的 issue 不受影响
- [x] 批量释放：无领取记录的 session 返回空列表
- [ ] （可选）claim 超时自动释放机制
