# 模块 6：Skill 集成

> 阶段：M4 | 依赖：M3（Issue 状态镜像）

## 目标

现有 Issue 工作流 Skill 需要调用本地 Issue Mirror 服务，实现 claim/fix/pr 前的状态检查。

## 集成点

| Skill | 集成点 |
|-------|--------|
| `001-4-issue-claim` | claim 前查 issue 确认状态，成功后更新 last_activity |
| `001-5-issue-fix` | 开始前查依赖，done 时更新 last_activity |
| `001-6-issue-pr` | PR 前查 issue 状态、rebase 状态、Test Plan |

## 功能

### 功能 1：Skill 状态查询

为 issue 相关 skill 提供统一状态查询。

1. claim 前查询 issue 是否 ready。
2. fix 前查询 issue 是否 blocked。
3. done 前查询是否有未完成检查。
4. pr 前查询 issue、branch、test plan、PR 状态。
5. review 前查询 PR 与 issue 关联状态。

### 功能 2：Skill 决策响应

返回给 skill 的结果必须结构化。

1. 返回 `allow/warn/block`。
2. 返回原因列表。
3. 返回建议下一步命令。
4. 返回相关 issue/PR/依赖信息。

### 功能 3：Skill 活动写入

Skill 成功执行关键动作后，需要写入活动日志。

1. claim 成功后记录 assignee 和时间。
2. fix 开始后记录 branch/sandbox。
3. pr 创建后记录 PR number。
4. test 完成后记录 Test Plan 状态。

### 功能 4：失败兜底

当本地服务不可用时，skill 需要有明确兜底。

1. 本地 Issue Mirror 不可用时给出错误。
2. 可提示用户执行同步或启动服务。
3. P0 风险检查失败时不应静默通过。
