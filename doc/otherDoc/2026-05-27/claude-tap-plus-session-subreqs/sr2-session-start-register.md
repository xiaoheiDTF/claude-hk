# SR-2: SessionStart Hook 会话注册

> 创建时间：2026-05-27
> 模块：claude-tap-plus / Hooks
> 简述：改造 SessionStart hook，在 Claude Code 启动时向后端注册会话元数据

---

## 目标

在 Claude Code 会话启动时（SessionStart hook），自动向后端发送会话注册请求，记录机器、项目、模型等信息。

## 改造范围

### 文件：`.claude/hooks/01-session-start/base.sh`

新增 `register_session()` 函数，在会话启动时调用。

## 实现逻辑

```bash
register_session() {
  # 检测代理是否在运行，不在则跳过
  check_proxy_active || return 0

  local session_id=$(json_get '.session_id')
  local transcript_path=$(json_get '.transcript_path')
  local cwd=$(json_get '.cwd')
  local model=$(json_get '.model')
  local source=$(json_get '.source')

  # 组装机器信息
  local machine_id="$(whoami)@$(hostname)"
  local os_type="$OS_TYPE"

  # 从 transcript_path 提取 project_slug
  local project_slug=""
  project_slug=$(echo "$transcript_path" | sed -E 's|.*/projects/([^/]+)/.*|\1|')

  # 发送注册请求（backend_url 由 check_proxy_active 设置到 PROXY_BACKEND_URL）
  curl -s -X POST "$PROXY_BACKEND_URL/api/session/register" \
    -H "Content-Type: application/json" \
    -d "{
      \"session_id\":\"$session_id\",
      \"machine_id\":\"$machine_id\",
      \"os\":\"$os_type\",
      \"project_slug\":\"$project_slug\",
      \"project_cwd\":\"$cwd\",
      \"trace_path\":\"$transcript_path\",
      \"model\":\"$model\",
      \"source\":\"$source\"
    }" > /dev/null 2>&1

  log "INFO" "Session registered to backend: $session_id"
}
```

## 关键设计决策

### 代理状态检测（取代配置文件）

Hook 通过 `check_proxy_active()` 检测 claude-tap-plus 代理是否在运行：

- **检测方式**：读取 `$HOME/.claude-tap-plus/.proxy.json`，检查 `pid` 是否存活
- **代理未运行**（`.proxy.json` 不存在或 PID 已死）→ 跳过注册，不发送请求
- **后端不可达** → curl 静默失败，不影响 Claude Code 正常使用
- **原因**：Hook 不应阻塞 Claude Code 的启动流程；用户直接运行 `claude` 时不应有任何副作用

**无需 `backend.conf`**：后端地址从代理状态文件的 `backend_url` 字段获取，代理启动时自动写入。

## 发送的数据

| 字段 | 来源 | 示例值 |
|------|------|--------|
| session_id | `json_get '.session_id'` | `bf15cac4-7235-...` |
| machine_id | `whoami@hostname` | `Administrator@DESKTOP-ABC123` |
| os | `platform.sh` 的 `$OS_TYPE` | `windows` |
| project_slug | 从 transcript_path 解析 | `D--CodeDevelopment-CodeProject-claude-hk` |
| project_cwd | `json_get '.cwd'` | `D:\CodeDevelopment\CodeProject\claude-hk` |
| trace_path | `json_get '.transcript_path'` | 完整的 JSONL 文件路径 |
| model | `json_get '.model'` | `GLM-5.1` |
| source | `json_get '.source'` | `startup` / `resume` |

## 验证标准

- [ ] 代理运行时（`.proxy.json` 存在且 PID 存活），启动 Claude Code 后能在后端查到会话记录
- [ ] 代理未运行时（`.proxy.json` 不存在或 PID 已死），启动无报错，不发送请求
- [ ] 后端不可达时，启动无报错，curl 静默失败
- [ ] 注册请求包含所有 8 个字段
- [ ] `source` 正确区分 `startup` 和 `resume`

## 依赖

- SR-4（后端 API `/api/session/register` 需要就绪才能端到端验证）
- SR-5（代理状态文件 `.proxy.json` 的管理规范与集成验证）
