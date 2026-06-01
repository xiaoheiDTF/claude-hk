# B8: 统一降级策略

> 创建时间：2026-05-27
> 模块：claude-tap-plus / Issue 管理
> 简述：定义后端服务不可用时各场景的降级行为，确保技能主流程不中断

---

## 需求描述

后端服务可能因未配置、未启动、网络故障等原因不可用。需要一个统一的降级策略，明确每种场景下技能脚本的行为，保证即使后端完全不可用，原有 GitHub 操作流程仍可正常执行。

## 设计原则

1. **后端是辅助，GitHub 是主链路**：后端只负责状态追踪和冲突防护，不替代 GitHub 操作
2. **失败静默**：后端调用失败不应阻塞或报错到用户层面（claim 除外）
3. **配置开关**：`.claude/backend.conf` 不存在时，完全不调用后端

## 降级行为表

| 场景 | 触发条件 | 降级行为 | 影响范围 |
|------|----------|----------|----------|
| 未配置后端 | `backend.conf` 不存在或 `BACKEND_URL` 为空 | 所有后端调用跳过，走原有逻辑 | 全部技能 |
| 后端不可用 | `GET /health` 超时或连接失败 | 所有后端调用跳过，走原有逻辑 | 全部技能 |
| `/api/issue/check` 失败 | 请求超时或返回非 JSON | 返回完整 issue 列表（不过滤） | 001-4-issue-claim |
| `/api/issue/claim` 失败 | 请求超时或返回失败 | **阻止领取**，提示用户 | 001-4-issue-claim |
| `/api/issue/status` 失败 | 请求超时或返回失败 | 静默忽略，继续操作 GitHub | 001-5 ~ 001-9 |
| `/api/issue/release-session` 失败 | 请求超时或返回失败 | 静默忽略 | SessionEnd hook |

## 各 API 降级详细说明

### check API 降级

```
正常：gh issue list → check API 过滤 → 展示空闲 issue
降级：gh issue list → 跳过过滤 → 展示全部 open issue
```

用户可能看到已被领取的 issue，但领取时会通过 claim API 兜底校验。

### claim API 降级

```
正常：用户选择 → claim API 原子领取 → 成功 → gh issue edit
降级：用户选择 → claim API 超时 → 提示"后端不可用，无法确认领取状态" → 阻止操作
```

**注意**：claim 是唯一一个降级时**阻止**而非静默的操作。原因：没有后端锁的保护，多 Agent 可能同时领取同一 issue，造成冲突。

可选策略：如果用户确认单 Agent 环境，可以提供 `--force` 参数跳过后端直接操作 GitHub。

### status API 降级

```
正常：update_issue_status() → 后端 API 更新 → 响应含 previous_status → 同步 GitHub label
降级：update_issue_status() → 后端 API 超时 → 无响应 → GitHub label 不变 → 继续技能主流程
```

后端失败时 `update_issue_status` 返回空，`_sync_github_label` 不会被调用，GitHub label 保持不变。技能主流程继续执行（如 `gh pr create` 等操作不受影响）。

### release-session API 降级

```
正常：SessionEnd → release-session API → 释放 issue
降级：SessionEnd → release-session API 超时 → 静默跳过
```

未释放的 issue 会在后端侧通过超时机制（B5 的 claim 超时自动释放）最终被释放。

## 实现方式

各技能脚本通过 `backend.sh` 的 `_backend_available()` 判断后端状态：

```bash
source "$CLAUDE_PROJECT_DIR/.claude/skills/backend.sh"

if _backend_available; then
  # 走后端逻辑
else
  # 走原有逻辑
fi
```

对于不需要预检查的场景（status/release），直接调用 `_call_backend`，失败时静默返回。

## 与各子需求的关联

| 子需求 | 降级影响 | 处理方式 |
|--------|----------|----------|
| B2 check API | check 失败不过滤 | `filter_claimed_issues` 返回原始列表 |
| B3 claim API | claim 失败阻止领取 | `claim_issue_backend` 返回非零，主流程跳过 gh edit |
| B4 status API | status 失败静默 | `update_issue_status` 输出丢弃 |
| B5 release API | release 失败静默 | hook 中 `> /dev/null 2>&1` |
| B6-1 claim 技能 | 同 B2+B3 | 组合处理 |
| B6-2 状态技能 | 同 B4 | 同 B4 |

## 验收标准

- [x] `backend.conf` 不存在时，所有技能走原有逻辑，无报错
- [x] 后端未启动时，所有技能走原有逻辑，无报错
- [x] check API 超时时，展示全部 issue 列表
- [x] claim API 超时时，阻止领取并提示用户
- [x] status API 超时时，技能主流程不受影响
- [x] release API 超时时，SessionEnd hook 不报错不卡住
