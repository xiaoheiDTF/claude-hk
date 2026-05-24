# 模块 3：Issue 状态镜像服务

> 阶段：M3 | 依赖：GitHub CLI 或 token

## 目标

在 `claude-tap-plus` 中提供本地服务和状态库，维护 GitHub Issue 本地镜像，使 hooks、skills、scheduler 可以快速查询 issue 状态而不用每次访问 GitHub。

## 功能

### 功能 1：Issue 同步

把 GitHub Issue 状态同步到本地状态库。

1. 支持手动触发同步。
2. 支持按 repo 同步。
3. 支持增量同步和全量同步。
4. 同步 issue、labels、assignee、milestone、PR 关联、更新时间。
5. 同步失败时保留旧数据并记录错误。
6. Webhook 本地不可达时提供手动同步 fallback。

### 功能 2：Issue 查询

提供本地查询能力，避免每次都访问 GitHub。

1. 查询全部 issue。
2. 查询单个 issue。
3. 查询 ready issue（open、无 assignee、无 blocked）。
4. 查询 in-progress issue。
5. 查询 blocked issue 和依赖链。
6. 支持 label、state、assignee、project 过滤。

### 功能 3：终端看板

快速查看工作状态。

1. 展示 ready、in-progress、blocked、recently closed 分区。
2. 展示 issue number、title、assignee、labels、last activity。
3. 默认输出终端友好文本。
4. 支持 JSON 输出给 skill 或其他工具消费。

### 功能 4：活动日志

所有 issue 状态变化都需要记录活动日志。

1. 记录动作来源：sync、webhook、skill、scheduler、user。
2. 记录动作类型：claim、release、blocked、unblocked、closed、comment。
3. 记录原始 payload 摘要。
4. 活动日志可按 issue 查询。
