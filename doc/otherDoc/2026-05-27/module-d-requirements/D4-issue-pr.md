# D4: 001-7-issue-pr 改造

> 创建时间：2026-05-27
> 模块：claude-tap-plus / 模块 D / D4
> 简述：PR 创建后向后端标记 `pr-created` 状态

---

## 目标

在 `001-7-issue-pr` 技能中，PR 创建成功后调用后端更新 `pr-created`。

## 改造文件

| 文件 | 操作 |
|------|------|
| `.claude/skills/001-7-issue-pr/scripts/03UserPromptSubmit.sh` | **修改** |

## 依赖

- **D0**: 需要 `source backend.sh`（`update_issue_status`）
- **后端 API**: `/api/issue/status`

## 改造内容

在 `gh pr create` 成功后：

```bash
source "$CLAUDE_PROJECT_DIR/.claude/skills/backend.sh"

# ... 原有逻辑：gh pr create ...

# [新增] 标记 pr-created
update_issue_status "$ISSUE_NUM" "pr-created"
```

## 传入后端的数据

```json
{
  "repo_full_name": "xiaoheiDTF/claude-hk",
  "issue_number": 10,
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "status": "pr-created"
}
```

## 降级行为

后端不可用时静默忽略，继续原有流程。

## 验证标准

- [ ] PR 创建后后端 issue 状态变为 `pr-created`
- [ ] 后端不可用时不影响原有流程
