# D5: 003-8-issue-test 改造

> 创建时间：2026-05-27
> 模块：claude-tap-plus / 模块 D / D5
> 简述：开始执行测试时向后端标记 `testing` 状态

---

## 目标

在 `003-8-issue-test` 技能中，找到关联 PR 并开始测试前调用后端更新 `testing`。

## 改造文件

| 文件 | 操作 |
|------|------|
| `.claude/skills/003-8-issue-test/scripts/03UserPromptSubmit.sh` | **修改** |

## 依赖

- **D0**: 需要 `source backend.sh`（`update_issue_status`）
- **后端 API**: `/api/issue/status`

## 改造内容

在找到关联 PR、开始执行测试前：

```bash
source "$CLAUDE_PROJECT_DIR/.claude/skills/backend.sh"

# ... 原有逻辑：gh pr view 找到关联 PR ...

# [新增] 标记 testing
update_issue_status "$ISSUE_NUM" "testing"

# 继续原有逻辑：执行测试项 ...
```

## 传入后端的数据

```json
{
  "repo_full_name": "xiaoheiDTF/claude-hk",
  "issue_number": 10,
  "session_id": "bf15cac4-7235-48ce-8853-5d4598547f31",
  "status": "testing"
}
```

## 降级行为

后端不可用时静默忽略，继续原有流程。

## 验证标准

- [ ] 开始测试后后端 issue 状态变为 `testing`
- [ ] 后端不可用时不影响原有流程
