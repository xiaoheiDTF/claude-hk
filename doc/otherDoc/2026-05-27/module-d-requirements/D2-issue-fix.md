# D2: 003-5-issue-fix 改造

> 创建时间：2026-05-27
> 模块：claude-tap-plus / 模块 D / D2
> 简述：开发分支创建后向后端标记 `fixing` 状态

---

## 目标

在 `003-5-issue-fix` 技能中，创建分支后调用后端标记 `fixing` 状态。

## 改造文件

| 文件 | 操作 |
|------|------|
| `.claude/skills/003-5-issue-fix/scripts/03UserPromptSubmit.sh` | **修改** |

## 依赖

- **D0**: 需要 `source backend.sh`（`update_issue_status`）
- **后端 API**: `/api/issue/status`

## 改造内容

在创建分支后、评论前的位置插入一行调用：

```bash
source "$CLAUDE_PROJECT_DIR/.claude/skills/backend.sh"

# ... 原有逻辑：检查 assignee、创建分支 ...

# [新增] 标记 fixing 状态
update_issue_status "$ISSUE_NUM" "fixing"

# 继续原有逻辑：gh issue comment ...
```

## 传入后端的数据

```json
{
  "repo_full_name": "xiaoheiDTF/claude-hk",
  "issue_number": 10,
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "status": "fixing"
}
```

## 降级行为

后端不可用时静默忽略，继续执行原有逻辑（`update_issue_status` 内部 return 0）。

## 验证标准

- [ ] 创建分支后后端 issue 状态变为 `fixing`
- [ ] 后端不可用时不影响原有流程
