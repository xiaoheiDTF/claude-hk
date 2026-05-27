# D3: 003-6-issue-done 改造

> 创建时间：2026-05-27
> 模块：claude-tap-plus / 模块 D / D3
> 简述：开发完成后向后端标记 `ready-for-pr` 状态

---

## 目标

在 `003-6-issue-done` 技能中，标记完成后调用后端更新 `ready-for-pr`。

## 改造文件

| 文件 | 操作 |
|------|------|
| `.claude/skills/003-6-issue-done/scripts/03UserPromptSubmit.sh` | **修改** |

## 依赖

- **D0**: 需要 `source backend.sh`（`update_issue_status`）
- **后端 API**: `/api/issue/status`

## 改造内容

在移除 `in-progress` label、添加 `ready-for-pr` label 后：

```bash
source "$CLAUDE_PROJECT_DIR/.claude/skills/backend.sh"

# ... 原有逻辑：修改 labels ...

# [新增] 标记 ready-for-pr
update_issue_status "$ISSUE_NUM" "ready-for-pr"
```

## 传入后端的数据

```json
{
  "repo_full_name": "xiaoheiDTF/claude-hk",
  "issue_number": 10,
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "status": "ready-for-pr"
}
```

## 降级行为

后端不可用时静默忽略，继续原有流程。

## 验证标准

- [ ] 标记完成后后端 issue 状态变为 `ready-for-pr`
- [ ] 后端不可用时不影响原有流程
