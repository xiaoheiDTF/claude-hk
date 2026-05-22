# Skill 完整清单

项目内置 13 个 Skill，通过 `/技能名` 调用。

## Skill 列表

### 通用工具

| Skill | 命令 | 输出目录 | 用途 |
|-------|------|---------|------|
| `001-testcode-python` | `/001-testcode-python` | `doc/testcode/python/{api,other}/` | Python 测试脚本、API 自动化测试 |
| `002-otherdoc` | `/002-otherdoc` | `doc/otherDoc/YYYY-MM-DD/` | 按日期归档通用文档 |

### Issue 工作流

| Skill | 命令 | 用途 |
|-------|------|------|
| `003-1-issue-init` | `/003-1-issue-init` | 初始化 issue 标签体系（一次性） |
| `003-2-issue` | `/003-2-issue` | 创建 GitHub Issue（本地草稿、模板、发布） |
| `003-3-issue-discuss` | `/003-3-issue-discuss #N` | 拉取 Issue 内容进行讨论 |
| `003-4-issue-claim` | `/003-4-issue-claim #N` | 原子领取 Issue |
| `003-5-issue-fix` | `/003-5-issue-fix #N` | 根据 issue 创建分支并开始开发 |
| `003-6-issue-done` | `/003-6-issue-done #N` | 标记开发完成，准备提 PR |
| `003-7-issue-pr` | `/003-7-issue-pr #N` | 创建 PR 关联 issue |
| `003-8-issue-test` | `/003-8-issue-test #N` | 执行 PR 的 Test Plan |
| `003-9-issue-review` | `/003-9-issue-review merge/reject #N` | 审核合并或打回 PR |

### Git 操作

| Skill | 命令 | 用途 |
|-------|------|------|
| `004-git-push` | `/004-git-push` | 规范化 commit + push（分组，中文信息） |
| `005-git-commit` | `/005-git-commit` | 规范化 commit（仅本地，不推送） |

## Issue 工作流完整链路

```
claim(003-4) → fix(003-5) → [编码] → done(003-6) → pr(003-7) → test(003-8) → review(003-9)
                                                                ├─ merge ✓
                                                                └─ reject ✗ → 回到 fix(003-5)
```

## 权限说明

每个 Skill 在 `SKILL.md` 的 `allowed-tools` 字段声明了最小权限集。例如：
- `003-6-issue-done` 只需 Bash + Read（收敛权限）
- `003-9-issue-review` 只需 Bash + Read
- `003-5-issue-fix` 需要 Bash + Read + Edit + Write + Glob + Grep（需要创建和编辑文件）
