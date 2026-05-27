# claude-tap-plus 会话管理 — 子需求拆分总览

> 创建时间：2026-05-27
> 模块：claude-tap-plus / 会话管理
> 简述：将 claude-tap-plus-session-design.md 拆分为 5 个独立可开发的子需求

---

## 子需求清单

| 编号 | 名称 | 改造范围 | 依赖 |
|------|------|----------|------|
| [SR-1](sr1-trace-path-restructure.md) | Trace 存储路径重构 | 代理 `trace/writer.go` | 无 |
| [SR-2](sr2-session-start-register.md) | SessionStart Hook 会话注册 | Hook `01-session-start/base.sh` | SR-4（需要后端 API 就绪才能验证） |
| [SR-3](sr3-session-end-unregister.md) | SessionEnd Hook 会话注销 | Hook `29-session-end/base.sh` | SR-4（需要后端 API 就绪才能验证） |
| [SR-4](sr4-backend-session-service.md) | 后端会话管理服务 | 新建后端服务（Go/SQLite） | 无 |
| [SR-5](sr5-config-integration.md) | 配置与集成测试 | 配置文件 + 端到端验证 | SR-1 ~ SR-4 全部完成 |

## 建议开发顺序

```
SR-1（代理侧，独立） ──┐
                        ├──→ SR-5（集成）
SR-4（后端，独立）    ──┤
                        │
SR-2（注册 hook）     ──┤
                        │
SR-3（注销 hook）     ──┘
```

SR-1 和 SR-4 完全独立，可以并行开发。SR-2/SR-3 依赖 SR-4 的 API 就绪。SR-5 是最后集成验证。

## 设计原则（原文档第一节）

1. **消息存储在本地**：API 调用的请求/响应由代理写入本地 JSONL trace 文件，后端不存储消息内容
2. **后端只存关联表**：后端仅记录 session → 机器 → 项目 → trace 文件路径 的映射关系
3. **trace 路径跟 Claude Code 一致**：复用 Claude Code 的 transcript_path 结构，便于溯源

## 可获取的数据源（原文档第二节）

### SessionStart Hook 字段

| 字段 | 说明 | 用途 |
|------|------|------|
| `session_id` | UUID | 后端注册主键 |
| `transcript_path` | 本地会话 JSONL 文件路径 | 消息溯源、提取项目标识 |
| `cwd` | 当前项目根目录 | 项目识别 |
| `source` | 启动来源（startup/resume） | 区分首次启动和恢复 |
| `model` | 使用的模型名称 | 环境记录 |

### SessionEnd Hook 字段

| 字段 | 说明 | 用途 |
|------|------|------|
| `session_id` | UUID | 注销匹配 |
| `reason` | 退出原因 | 判断正常/异常退出 |
| `transcript_path` | 本地会话 JSONL 文件路径 | 可用但当前未使用 |
| `cwd` | 当前项目根目录 | 可用但当前未使用 |
| `hook_event_name` | 事件名称 | 可用但当前未使用 |

### 平台信息（hooks/platform.sh 已有）

| 数据 | 获取方式 | 示例值 |
|------|----------|--------|
| OS 类型 | `uname -s` | windows / linux / macos |
| 机器名 | `hostname` | DESKTOP-XXX |
| 用户 | `whoami` | Administrator |

### 代理已有数据

| 数据 | 说明 |
|------|------|
| request/response body | 完整请求/响应 JSON |
| session_id | 从请求元数据提取 |
| timestamp / duration_ms | 每次调用时间和耗时 |
| token 统计 | input/output/cache tokens |
| model | 实际调用的模型 |

### project_id 组合标识符

trace 路径中使用 `{machine_id}/{project_slug}` 组合作为目录层级：

```
project_id = "{hostname}/{project_slug}"
```

示例：`DESKTOP-ABC123/D--CodeDevelopment-CodeProject-claude-hk`

后端数据模型中 `machine_id` 和 `project_slug` 保持为独立字段，不作为单独的数据模型字段存储。

## Issue 管理 API 交叉引用

后端服务同时提供 Issue 管理 API（`/api/issue/*`），由独立的 `issue-management-reqs/` 模块定义。Session 会话管理和 Issue 管理共享同一个后端服务实例。
