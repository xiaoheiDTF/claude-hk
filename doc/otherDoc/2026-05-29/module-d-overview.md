# 模块 D 总览：Issue 技能集成后端服务

> 创建时间：2026-05-29
> 模块：claude-tap-plus / 模块 D
> 简述：汇总 001-x-issue 系列技能接入后端服务的实现与测试状态

---

## 设计原则

1. 后端优先，GitHub 次之 — 状态变更先调后端，再操作 GitHub
2. 最小侵入改造 — 现有流程不变，只在关键节点插入后端调用
3. 失败静默 — 后端不可用时降级为原有行为
4. 配置驱动 — 通过 `.claude/backend.conf` 控制启用

---

## D0-D7 实现状态

| 子需求 | 改造文件 | 后端调用 | 状态 |
|--------|----------|----------|------|
| D0: 公共基础设施 | `.claude/skills/backend.sh` | `_backend_available()`, `_call_backend()`, `_get_session_id()`, `update_issue_status()` | ✅ |
| D1: issue-claim | `.claude/skills/001-4-issue-claim/scripts/03UserPromptSubmit.sh` | `/api/issue/check` 去重 + `/api/issue/claim` 原子领取 | ✅ |
| D2: issue-fix | `.claude/skills/001-5-issue-fix/scripts/03UserPromptSubmit.sh` | `update_issue_status("fixing")` | ✅ |
| D3: issue-done | `.claude/skills/001-6-issue-done/scripts/03UserPromptSubmit.sh` | `update_issue_status("ready-for-pr")` | ✅ |
| D4: issue-pr | `.claude/skills/001-7-issue-pr/scripts/03UserPromptSubmit.sh` | `update_issue_status("pr-created")` | ✅ |
| D5: issue-test | `.claude/skills/001-8-issue-test/scripts/03UserPromptSubmit.sh` | `update_issue_status("testing")` | ✅ |
| D6: issue-review | `.claude/skills/001-9-issue-review/scripts/03UserPromptSubmit.sh` | `update_issue_status("reviewing"/"merged"/"rejected")` | ✅ |
| D7: SessionEnd | `.claude/hooks/29-session-end/base.sh` | `/api/issue/release-session` 自动释放 | ✅ |

---

## 集成测试覆盖

`claude_tap_plus/tests/integration/backend_skill_flow_test.go` 包含 8 个测试函数：

| 测试函数 | 覆盖子需求 | 子测试数 |
|----------|-----------|----------|
| `TestSessionEndHook` | D7 | — |
| `TestClaimAPI` | D1 (B3 验收) | 6 |
| `TestIssueFixFlow` | D2 | 5 |
| `TestIssueDoneFlow` | D3 | 3 |
| `TestIssuePRFlow` | D4 | 3 |
| `TestIssueTestFlow` | D5 | 3 |
| `TestIssueReviewFlow` | D6 | 6 |
| `TestSessionEndRelease` | D7 | 5 |

---

## 数据流

```
/001-4-issue-claim → gh issue list → POST /api/issue/check 过滤 idle
                  → 用户选择 → POST /api/issue/claim 原子领取 → gh issue edit

/001-5-issue-fix  → 创建分支后 → POST /api/issue/status (fixing)
/001-6-issue-done → 标记完成 → POST /api/issue/status (ready-for-pr)
/001-7-issue-pr   → PR 创建后 → POST /api/issue/status (pr-created)
/001-8-issue-test → 开始测试 → POST /api/issue/status (testing)
/001-9-issue-review → 审核 → POST /api/issue/status (reviewing/merged/rejected)

SessionEnd hook → POST /api/issue/release-session 自动释放
```

---

## 原始设计文档索引

| 文档 | 位置 |
|------|------|
| 模块 D 总览设计 | `doc/otherDoc/2026-05-27/module-d-requirements/00-overview.md` |
| D0 后端基础设施 | `doc/otherDoc/2026-05-27/module-d-requirements/D0-backend-infra.md` |
| D1 issue-claim 改造 | `doc/otherDoc/2026-05-27/module-d-requirements/D1-issue-claim.md` |
| D2 issue-fix 改造 | `doc/otherDoc/2026-05-27/module-d-requirements/D2-issue-fix.md` |
| D3 issue-done 改造 | `doc/otherDoc/2026-05-27/module-d-requirements/D3-issue-done.md` |
| D4 issue-pr 改造 | `doc/otherDoc/2026-05-27/module-d-requirements/D4-issue-pr.md` |
| D5 issue-test 改造 | `doc/otherDoc/2026-05-27/module-d-requirements/D5-issue-test.md` |
| D6 issue-review 改造 | `doc/otherDoc/2026-05-27/module-d-requirements/D6-issue-review.md` |
| D7 SessionEnd 自动释放 | `doc/otherDoc/2026-05-27/module-d-requirements/D7-session-end-release.md` |
| 降级测试 | `doc/otherDoc/2026-05-27/module-d-requirements/degradation-testing.md` |
