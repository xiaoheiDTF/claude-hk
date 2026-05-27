# SR-5: 配置与集成测试

> 创建时间：2026-05-27
> 模块：claude-tap-plus / 配置与集成
> 简述：后端配置文件管理 + SR-1 ~ SR-4 的端到端集成验证

---

## 目标

1. 定义后端配置文件规范
2. 验证 SR-1 ~ SR-4 的端到端数据流

## 一、代理状态文件规范

### 文件：`$HOME/.claude-tap-plus/.proxy.json`

```json
{ "pid": 12345, "backend_url": "http://localhost:8080" }
```

### 写入与清理

| 时机 | 动作 | 说明 |
|------|------|------|
| 代理启动 | 创建文件，写入 pid + backend_url | `backend_url` 从代理配置或命令行参数获取 |
| 代理正常退出 | 删除文件 | `defer os.Remove()` 确保退出时清理 |
| 代理异常退出（kill -9） | 文件残留 | Hook 通过 `kill -0 $pid` 检测到 PID 已死，自动跳过 |

### Hook 检测逻辑

由 `hooks/base.sh` 中的 `check_proxy_active()` 共享函数完成（详见设计文档 5.1 节）。

### 行为规则

| 场景 | 行为 |
|------|------|
| 文件存在且 PID 存活、后端可达 | 正常注册/注销 |
| 文件存在且 PID 存活、后端不可达 | curl 静默失败，不阻塞 Claude Code |
| 文件不存在（用户直接运行 `claude`） | 跳过注册/注销，不发送请求 |
| 文件存在但 PID 已死（代理异常退出残留） | 跳过，等同文件不存在 |

### 渐进式启用

```
1. 部署后端服务（SR-4）
2. 启动 claude-tap-plus 代理，自动生成 .proxy.json
3. 代理启动 Claude Code 子进程
4. SessionStart hook 检测到代理运行，自动注册会话
5. 不使用 claude-tap-plus 的项目，.proxy.json 不存在，自动跳过
```

## 二、端到端数据流验证

### 测试步骤

```
1. 启动后端服务：go run ./cmd/claude-tap-server
   → 验证：服务监听在配置的端口

2. 确认代理状态文件存在
   → 验证：`$HOME/.claude-tap-plus/.proxy.json` 包含 pid 和 backend_url

3. 通过 claude-tap-plus 启动 Claude Code
   → 验证 SR-1：trace 文件路径包含 machine_id/project_slug 层级
   → 验证 SR-2：后端收到注册请求，sessions/machines/projects 表有记录

4. 进行几次用户交互（触发 API 调用）
   → 验证 SR-1：trace 文件正确追加写入

5. 退出 Claude Code
   → 验证 SR-3：后端收到注销请求，session 状态变为 closed

6. 查询后端
   → 验证 SR-4：GET /api/sessions 返回正确的会话列表
   → 验证：GET /api/session/:id 返回完整元数据
```

### 验证矩阵

| 验证项 | 对应子需求 | 验证方法 |
|--------|-----------|----------|
| trace 文件路径结构 | SR-1 | 检查 `.claude-tap-plus/.traces/` 下目录层级 |
| trace 内容追加写入 | SR-1 | 多次 API 调用后检查 JSONL 行数 |
| 会话注册 | SR-2 | 查询后端 DB sessions 表 |
| 机器/项目自动创建 | SR-2 | 查询后端 DB machines/projects 表 |
| 会话注销 | SR-3 | 查询后端 DB session status + closed_at |
| 静默失败（代理未启动） | SR-2/SR-3 | 直接运行 `claude`（不走代理），启动/退出无报错 |
| 静默失败（后端宕机） | SR-2/SR-3 | 停止后端，启动/退出无报错 |
| 残留文件自动忽略 | SR-2/SR-3 | 创建含过期 PID 的 `.proxy.json`，启动无报错 |
| 查询 API | SR-4 | curl 测试各过滤条件 |
| 端到端完整流程 | 全部 | 上述 6 步全部通过 |

## 三、集成后的数据流（完整）

```
用户启动 claude-tap-plus claude
  │
  ├─ 代理启动，监听本地端口
  ├─ Claude Code 启动
  │    │
  │    ├─ SessionStart hook 触发
  │    │    └─ POST /api/session/register → 后端记录会话元数据
  │    │
  │    ├─ 用户交互中...
  │    │    │
  │    │    ├─ 每次 API 请求 → 代理拦截
  │    │    │    ├─ 转发到上游 API
  │    │    │    └─ 本地 JSONL trace 文件写入（SR-1 路径结构）
  │    │    │
  │    │    └─ Issue 技能调用 → 查询后端状态
  │    │
  │    └─ SessionEnd hook 触发
  │         └─ POST /api/session/close → 后端注销会话
  │
  └─ 代理退出，打印汇总
```

## 四、异常场景测试

| 场景 | 预期行为 | 验证方法 |
|------|----------|----------|
| 后端启动前就开 Claude Code | 注册失败静默忽略，后续 API 调用 trace 正常写本地 | 直接运行 `claude`（不通过代理），检查无报错 |
| 会话中途后端重启 | 注销请求失败，session 状态停留在 active | 会话中途 kill 后端，退出后查询 DB |
| 同一 session_id 重复注册 | 后端返回 409，不重复创建 | 手动 curl 发两次相同注册请求 |
| 异常退出（kill -9） | SessionEnd 不触发，session 保持 active | kill -9 后查询后端 DB |
| 代理异常退出残留 `.proxy.json` | PID 已死，`check_proxy_active` 返回失败，跳过注册 | 手动创建含过期 PID 的 `.proxy.json`，启动无报错 |
| 不使用 claude-tap-plus 直接运行 `claude` | `.proxy.json` 不存在，完全跳过 | 直接运行 `claude`，检查 hook 日志无注册请求 |
| Ctrl+C 中断退出 | session → closed，reason 待观察 | 启动后按 Ctrl+C，查询后端 DB 检查状态 |

## 依赖

- SR-1 ~ SR-4 全部完成
- 这是最后一个子需求，负责端到端验证
