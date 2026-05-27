# B6-3: SessionEnd Hook 改造

> 创建时间：2026-05-27
> 模块：claude-tap-plus / Issue 管理 / 技能改造
> 简述：改造 SessionEnd hook，在会话结束时自动释放该 session 领取的所有 issue

---

## 需求描述

在 `.claude/hooks/29-session-end/base.sh` 末尾增加后端调用，当 session 结束时自动释放该 session 领取的所有未终态 issue。

## 改造内容

### 新增函数

在 `29-session-end/base.sh` 末尾追加：

```bash
release_session_issues() {
  local session_id
  session_id=$(json_get '.session_id')
  [ -z "$session_id" ] && return 0

  local backend_url
  backend_url=$(cat "$CLAUDE_PROJECT_DIR/.claude/backend.conf" 2>/dev/null \
    | grep '^BACKEND_URL=' | cut -d= -f2)
  [ -z "$backend_url" ] && return 0

  local result
  result=$(curl -s --max-time 5 -X POST "$backend_url/api/issue/release-session" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$session_id\"}")

  local count
  count=$(echo "$result" | jq -r '.count // 0' 2>/dev/null)
  [ "$count" -gt 0 ] && log "INFO" "Released $count issues for session $session_id"
}

release_session_issues
```

### 关键设计点

| 点 | 说明 |
|-----|------|
| `--max-time 5` | 超时 5 秒，避免 session 结束时卡住 |
| `> /dev/null 2>&1` 不使用 | curl 结果需要解析 count，但 jq 失败不报错 |
| `backend.conf` 不存在时静默跳过 | 兼容未配置后端的场景 |
| `merged`/`rejected` 不释放 | 后端 API 保证终态不被释放 |

## 前置依赖

- `.claude/backend.conf` 文件存在且配置了 `BACKEND_URL=xxx`
- 后端 `/api/issue/release-session` 接口可用

## 验收标准

- [ ] session 正常结束时，领取的 issue 被自动释放
- [ ] 已合并/打回的 issue 不被释放
- [ ] 后端不可用时，hook 不报错、不卡住（5 秒超时）
- [ ] 未配置 backend.conf 时，静默跳过
- [ ] 无领取记录的 session 不报错
