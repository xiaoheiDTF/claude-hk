# API 能力缺口清单

> 拆分自：`2026-06-01-可视化页面线框图设计-v2.md` §六
> 相关参考：[共享参考：API 清单与数据模型](00-共享参考-API清单与数据模型.md)
> 校验日期：2026-06-06 — 已对照 `router.go` 逐条验证路由注册

---

## 仍需后端扩展的 API

> 2026-06-06 校验：以下 6 项在 `router.go` 中均未注册，确认全部仍未实现。

| 缺失 API | 影响页面 | 功能描述 | 当前替代方案 |
|----------|----------|----------|-------------|
| `GET /api/issue/{repo}/{number}` | Issue 详情 | 无法查看单个 Issue 的完整历史信息 | 用 `GET /api/issues?repo=X` 过滤后前端筛选 |
| `GET /api/issues/stats` | Issue 统计 | 无法按仓库或全局统计 Issue 状态分布 | 前端多次调用 `GET /api/issues?status=X` 计数 |
| `GET /api/machine/{id}` | 机器详情 | 无法查看单台机器的关联会话和统计 | 用 `GET /api/sessions?machine_id=X` 间接获取 |
| `GET /api/machine/{id}/stats` | 机器统计 | 无法查看机器的会话数、Issue 处理数等统计 | 无直接替代，需聚合多个 API |
| `GET /api/project/{slug}` | 项目详情 | 无法查看项目的关联会话和 Issue 统计 | 用 `GET /api/sessions?project_slug=X` 间接获取 |
| `POST /api/session/{id}/heartbeat` | 心跳检测 | 无法检测异常断连的会话 | 无替代，依赖 session-end 事件被动关闭 |

---

## 已实现的 API（之前在缺口清单中）

> 2026-06-06 校验：以下 9 项在 `router.go` 中均已注册 handler，确认全部可用。

| API | 实现版本 | 说明 |
|-----|---------|------|
| `GET /api/issues` | ✅ 已实现 | 支持按仓库、状态、会话过滤 + 分页 |
| `GET /api/machines` | ✅ 已实现 | 支持按 OS、hostname 过滤 |
| `GET /api/projects` | ✅ 已实现 | 按 last_seen_at 倒序 |
| `GET /api/session/{id}/issues` | ✅ 已实现 | 返回会话关联的所有 Issue |
| `GET /api/session/{id}/tokens` | ✅ 已实现 | 解析 trace 文件汇总 token（使用 `domain.TokenStats`） |
| `GET /api/session/{id}/traces` | ✅ 已实现 | 返回 trace 文件元数据 |
| `GET /api/logs` | ✅ 已实现 | 支持级别、日期过滤 |
| `GET /api/config` | ✅ 已实现 | 返回所有配置项（`config` 表） |
| `PUT /api/config` | ✅ 已实现 | 支持部分更新配置 |
