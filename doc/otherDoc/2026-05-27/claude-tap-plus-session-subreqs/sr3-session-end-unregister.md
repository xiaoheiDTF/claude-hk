# SR-3: SessionEnd Hook 会话注销

> 创建时间：2026-05-27
> 模块：claude-tap-plus / Hooks
> 简述：改造 SessionEnd hook，在 Claude Code 退出时向后端注销会话

---

## 目标

在 Claude Code 会话结束时（SessionEnd hook），向后端发送会话注销请求，记录退出时间和原因。

## 改造范围

### 文件：`.claude/hooks/29-session-end/base.sh`

新增 `unregister_session()` 函数，在会话结束时调用。

## 实现逻辑

```bash
unregister_session() {
  # 检测代理是否在运行，不在则跳过
  check_proxy_active || return 0

  local session_id=$(json_get '.session_id')
  local reason=$(json_get '.reason')

  curl -s -X POST "$PROXY_BACKEND_URL/api/session/close" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$session_id\",\"reason\":\"$reason\"}" \
    > /dev/null 2>&1

  log "INFO" "Session unregistered from backend: $session_id"
}
```

## 发送的数据

| 字段 | 来源 | 示例值 |
|------|------|--------|
| session_id | `json_get '.session_id'` | `bf15cac4-7235-...` |
| reason | `json_get '.reason'` | `prompt_input_exit` |

> **注**：SessionEnd hook 还提供 `transcript_path`、`cwd`、`hook_event_name` 等字段，当前注册/注销流程不使用。

## 退出原因（reason）可能值

| reason | 含义 |
|--------|------|
| `prompt_input_exit` | 用户主动输入 exit 退出 |
| Ctrl+C 中断退出 | SessionEnd hook 是否触发取决于 Claude Code 实现 |
| 其他值 | 待观察实际行为，可能包括异常中断等 |

## 关键设计决策

### 同 SR-2 的代理状态检测策略

与注册一样，通过 `check_proxy_active()` 检测代理是否在运行：
- `.proxy.json` 不存在或 PID 已死 → 跳过
- 后端不可达 → 静默失败
- **原因**：退出流程更不能阻塞，即使注销失败也不应影响 Claude Code 的正常退出

### 异常退出场景

如果 Claude Code 异常崩溃（kill -9、系统断电等），SessionEnd hook 不会触发，后端的 session 状态将停留在 `active`。

| 场景 | 后端状态 | 处理方式 |
|------|----------|----------|
| 用户输入 `exit` 正常退出 | session → `closed`，记录 `close_reason` | SessionEnd hook 正常触发 |
| Ctrl+C 中断退出 | session → `closed`，reason 待观察 | SessionEnd hook 是否触发取决于 Claude Code 实现 |
| kill -9 强杀进程 | session 保持 `active` | SessionEnd 不触发，需要后端清理 |
| 系统断电/崩溃 | session 保持 `active` | SessionEnd 不触发，需要后端清理 |

**超时清理策略**（后端可选实现）：

```sql
UPDATE sessions
SET status = 'closed',
    closed_at = datetime('now'),
    close_reason = 'timeout_cleanup'
WHERE status = 'active'
  AND registered_at < datetime('now', '-24 hours');
```

建议作为后端定时任务或启动时执行一次。

## 验证标准

- [x] 正常退出 Claude Code 后，后端会话状态变为 `closed`
- [x] 后端记录了 `close_reason` 和 `closed_at` 时间
- [x] 代理未运行时，退出无报错
- [x] 后端不可达时，退出无报错
- [x] 异常退出后，后端 session 状态保持 `active`（符合预期）

## 依赖

- SR-4（后端 API `/api/session/close` 需要就绪才能端到端验证）
- SR-5（代理状态文件 `.proxy.json` 的管理规范与集成验证，与 SR-2 共用）
