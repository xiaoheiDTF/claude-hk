# D6: 003-9-issue-review 改造

> 创建时间：2026-05-27
> 模块：claude-tap-plus / 模块 D / D6
> 简述：审核时标记 reviewing 中间状态，merge/reject 后向后端更新最终状态

---

## 目标

在 `003-9-issue-review` 技能中：
- 审核开始时先标记 `reviewing`
- merge 成功后标记 `merged`
- reject 后标记 `rejected`

## 改造文件

| 文件 | 操作 |
|------|------|
| `.claude/skills/003-9-issue-review/scripts/03UserPromptSubmit.sh` | **修改** |

## 依赖

- **D0**: 需要 `source backend.sh`（`update_issue_status`）
- **后端 API**: `/api/issue/status`

## 改造内容

```bash
source "$CLAUDE_PROJECT_DIR/.claude/skills/backend.sh"

# 审核开始，标记 reviewing
update_issue_status "$issue_num" "reviewing"

# merge 分支
merge_issue() {
  local issue_num="$1"

  # 原有逻辑：gh pr merge ...

  # [新增] 更新后端状态
  update_issue_status "$issue_num" "merged"
}

# reject 分支
reject_issue() {
  local issue_num="$1"

  # 原有逻辑：gh pr comment ... gh issue edit ...

  # [新增] 更新后端状态
  update_issue_status "$issue_num" "rejected"
}
```

## 传入后端的数据

merge 时：
```json
{
  "repo_full_name": "xiaoheiDTF/claude-hk",
  "issue_number": 10,
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "status": "merged"
}
```

reject 时：
```json
{
  "repo_full_name": "xiaoheiDTF/claude-hk",
  "issue_number": 10,
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "status": "rejected"
}
```

## 降级行为

后端不可用时静默忽略，继续原有流程。

## 验证标准

- [ ] 审核开始时 issue 状态变为 `reviewing`
- [ ] merge 后后端 issue 状态变为 `merged`
- [ ] reject 后后端 issue 状态变为 `rejected`
- [ ] rejected 状态的 issue 被 SessionEnd 释放后可重新 claim
- [ ] merged 状态的 issue 不被 SessionEnd 释放
- [ ] 后端不可用时不影响原有 merge/reject 流程
