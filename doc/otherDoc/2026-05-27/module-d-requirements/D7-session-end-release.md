# D7: SessionEnd hook 自动释放

> 创建时间：2026-05-27
> 模块：claude-tap-plus / 模块 D / D7
> 简述：会话结束时自动释放该 session 持有的所有 issue（排除 merged/rejected）

---

## 目标

在 `SessionEnd` hook 中调用后端 `/api/issue/release-session`，自动释放当前会话持有的 issue，防止 Agent 异常退出后 issue 永久锁定。

## 改造文件

| 文件 | 操作 |
|------|------|
| `.claude/hooks/29-session-end/base.sh` | **修改**（或新增释放逻辑） |

## 依赖

- **后端 API**: `/api/issue/release-session`
- **注意**：D7 是唯一不依赖 `backend.sh` 的模块，因为它运行在 hook 环境而非 skill 环境中，无法 source skill 目录下的文件，因此直接内联 curl 调用

## 改造内容

在 `29-session-end/base.sh` 末尾添加：

```bash
release_session_issues() {
  local session_id
  session_id=$(json_get '.session_id')
  [ -z "$session_id" ] && return 0

  local backend_url
  backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null | grep '^BACKEND_URL=' | cut -d= -f2)
  [ -z "$backend_url" ] && return 0

  local result
  result=$(curl -s --max-time 5 -X POST "$backend_url/api/issue/release-session" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$session_id\"}" 2>/dev/null)

  local count
  count=$(echo "$result" | jq -r '.count // 0')
  [ "$count" -gt 0 ] && log "INFO" "Released $count issues for session $session_id"
}

release_session_issues
```

## 传入后端的数据

```json
{
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31"
}
```

## 后端处理逻辑（模块 B 侧）

- 查找 `issue_claims` 表中 `session_id` 匹配的记录
- 排除 `status IN ('merged', 'rejected')` 的记录
- 将剩余记录的 `status` 更新为 `idle`，`session_id` 和 `claimed_at` 清空（`UPDATE status='idle', session_id=NULL, claimed_at=NULL`）
- 返回释放数量 `count`

## 降级行为

| 场景 | 行为 |
|------|------|
| 未配置 backend.conf | 跳过释放 |
| 后端不可用 | 静默忽略，不阻塞会话关闭 |
| 释放失败 | 静默忽略 |

## 验证标准

- [ ] 会话正常结束时，该 session 持有的 issue 被释放为 `idle`
- [ ] `merged` 状态的 issue 不被释放
- [ ] `rejected` 状态的 issue 被释放为 `idle`（可重新领取）
- [ ] 后端不可用时不影响会话关闭
- [ ] 释放后日志记录释放数量
