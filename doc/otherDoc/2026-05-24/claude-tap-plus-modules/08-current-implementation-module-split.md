# 模块 8：当前实现模块划分与补充需求

> 阶段：现状梳理 | 来源：`claude_tap_plus` 当前 Go 代码结构

## 目标

梳理当前已实现的模块能力，补充遗漏的功能需求，为后续规划提供基线。

## 已实现模块总览

| 模块 | 当前状态 | 说明 |
|------|---------|------|
| CLI 入口与子命令分发 | 已实现 | 默认 proxy 模式，支持 session 子命令 |
| Client 配置与 Claude 启动适配 | 已实现 | 解析命令、环境变量、配置注入 |
| Reverse Proxy | 已实现 | 代理 API 请求、path 白名单、header 脱敏 |
| SSE 重组 | 已实现 | 将流式响应重组成完整响应 |
| Trace 写入与统计 | 已实现 | JSONL trace 写入、token 用量统计 |
| Usage 归一化 | 已实现 | 统一多 provider token 字段 |
| Session 快照同步 | 已实现 | session-push/pull/status |
| 测试辅助 | 已实现 | 单测和 E2E 支撑 |

## 功能

### 功能 1：代理启动

用户可以通过一个命令启动 Claude Code，并让 API 流量经过本地代理。

1. 支持自动检测上游 API 地址。
2. 支持用户显式指定上游 API 地址。
3. 支持随机端口或指定端口。
4. 支持把剩余参数透传给 Claude Code。
5. 支持退出时输出 API 调用和 token 用量摘要。

### 功能 2：Trace 记录

代理模式下必须记录 API 请求与响应。

1. 每次 API 调用写入一条 trace 记录。
2. streaming 响应需要尽可能重组成完整响应。
3. 记录请求方法、路径、状态码、耗时、usage。
4. 敏感 header 必须脱敏。
5. trace 写入失败时要输出明确错误。

### 功能 3：Session 快照命令

当前已存在的 session 命令定义为"快照同步"能力。

1. `session-push` 收集 Claude session 到本地副本。
2. `session-pull` 将本地副本恢复到 Claude 目录。
3. `session-status` 查看本地副本状态。
4. 写入 Claude 原始目录的命令必须显式提示风险。
5. 后续只读索引能力不能和快照恢复混淆。

### 功能 4：测试与验收

当前功能需要保持可测试。

1. proxy path/header 有测试。
2. SSE 重组有测试。
3. trace 写入和 usage 统计有测试。
4. session JSONL 解析有测试。
5. 后续新增 CLI 参数路由和 sandbox 需要测试。

## 需要补齐的需求

1. **命令统一**：当前存在 `claude-tap-plus`、`claude_tap_plus`、`claude-tap` 三种叫法，需统一。
2. **目录写入边界**：哪些命令会写当前项目、哪些只写固定存储、哪些会写 Claude 原始目录，必须清楚标注。
3. **项目识别一致性**：trace、session、sandbox、issue 不能各自实现 project name 检测。
4. **跨平台路径**：Windows drive、WSL path、macOS/Linux path 的 slug 规则要稳定。
5. **并发运行**：多个实例同时启动时，端口、trace、db lock、worktree 不应冲突。
6. **失败可恢复**：proxy 启动成功但 Claude 启动失败、trace 写入失败、worktree 创建失败都要能清理中间状态。
7. **安全脱敏**：body、headers、settings、env 中的 token 都需要统一脱敏策略。
8. **Dry-run**：涉及 GitHub、Git worktree、session-pull、apply/sync 的命令都应支持 dry-run。
