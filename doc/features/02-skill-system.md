# Skill 系统功能与配置

> 最后更新：2026-06-06

---

## 功能概述

Skill 是通过 `/skill-name` 触发的功能单元。每个 Skill 拥有完整的生命周期：触发 → 上下文注入 → Hook 事件转发 → 清理。Skill 通过 `.active` 文件绑定到当前会话，同一会话同时只激活一个 Skill。

### 核心机制

```
用户输入 /skill-name
  → 03 skill-inject.sh 匹配 registry.conf
    → 执行 03UserPromptSubmit.sh（上下文注入）
      → 写入 .active（session_id|skill_name）
        → 事件 05~29 转发到 skills/<skill>/scripts/<Event>.sh
          → 16Stop.sh 清理并移除 .active 条目
```

### Skill 目录结构

```
.claude/skills/
├── registry.conf              # Skill 注册表（每行一个 skill 名）
├── .active                    # 活跃标记文件（session_id|skill_name，多行）
├── active.sh                  # .active 文件 CRUD（加锁）
├── lock.sh                    # 跨平台文件锁（mkdir 原子操作）
├── log.sh                     # 双写日志：hooks/logs/ + skills/log/<tag>/
├── backend.sh                 # 后端 API 调用（001 系列共用）
├── enforce_boundary.sh        # 工具白名单解析与拦截
├── <XXX-skill-name>/
│   ├── SKILL.md               # 定义（frontmatter: name/description/allowed-tools）
│   └── scripts/
│       ├── 03UserPromptSubmit.sh  # 上下文注入
│       ├── 16Stop.sh              # 清理
│       ├── init.sh                # 项目首次初始化（可选）
│       └── init_check.sh          # 每次会话环境检查（可选）
└── log/
    └── <skill-tag>/
        └── YYYY-MM-DD.log     # Skill 运行日志
```

---

## 24 个 Skill 清单

### 001 系列 — Issue 工作流

完整的 Issue 驱动开发管线，支持原子领取、多 Agent 协调。

```
/001-1-issue-init → /001-2-issue → /001-3-issue-discuss
                                       │
/001-9-issue-review ← /001-8-issue-test ← /001-7-issue-pr ← /001-6-issue-done ← /001-5-issue-fix ← /001-4-issue-claim
```

| Skill | 功能 | allowed-tools | 输出 |
|-------|------|---------------|------|
| `001-1-issue-init` | 一次性初始化 GitHub 标签体系 | Bash, Read, Write | `.github/.issue-initialized`；GitHub 远程标签 |
| `001-2-issue` | 创建 Issue（支持草稿/模板/发布） | Bash, Read, Write, Glob, Grep | `doc/issues/drafts/`、`doc/issues/templates/` |
| `001-3-issue-discuss` | 拉取 Issue 及评论到会话讨论 | Bash, Read, Write | GitHub Issue Comment |
| `001-4-issue-claim` | **原子领取** Issue（后端 API 或 Label 降级） | Bash, Read | GitHub assignee + label 变更 |
| `001-5-issue-fix` | 根据 Issue 创建 `<type>/issue-<N>-<desc>` 分支 | Bash, Read, Edit, Write, Glob, Grep | Git 分支 |
| `001-6-issue-done` | 标记开发完成，label → ready-for-pr | Bash, Read | GitHub label 变更 |
| `001-7-issue-pr` | 创建 PR（**必须包含 `## Test plan` 区块**） | Bash, Read, Edit, Write, Glob, Grep | GitHub PR（含 `Closes #N`） |
| `001-8-issue-test` | 执行 Test Plan，逐项勾选 checkbox | Bash, Read, Edit, Glob, Grep | PR body checkbox `[ ]→[x]` |
| `001-9-issue-review` | 审核 PR：检查 Test Plan 完成度后合并或打回 | Bash, Read | GitHub merge 或 rejected label |

### 002 系列 — 文档管理

| Skill | 功能 | allowed-tools | 输出 |
|-------|------|---------------|------|
| `002-1-doc-otherdoc` | 按日期归档存储 Markdown 文档 | Bash, Read, Write, Edit, Glob, Grep | `doc/otherDoc/<YYYY-MM-DD>/<文件名>.md` |
| `002-2-doc-testcode-python` | 编写 Python 测试脚本和 API 自动化测试 | Bash, Read, Write, Edit, Glob, Grep | `doc/testcode/python/api/`、`doc/testcode/python/other/` |

### 003 系列 — 开发流程（BDD/TDD/CDD 管线）

从需求到 E2E 的完整开发流程驱动：

```
功能树 → BDD 场景 → 后端/前端 BDD → API 契约 → 后端 TDD / 前端 CDD → E2E 测试
```

| Skill | 功能 | allowed-tools |
|-------|------|---------------|
| `003-1-develop-feature-tree` | 功能点与功能树分析 | Bash, Read, Write, Edit, Glob, Grep |
| `003-2-develop-bdd-scenario` | BDD 场景规范（正向/异常/边界） | Bash, Read, Write, Edit, Glob, Grep |
| `003-3-1-backend-bdd` | 后端 BDD：拆解为后端可验证的行为场景 | Bash, Read, Write, Edit, Glob, Grep |
| `003-3-2-frontend-bdd` | 前端 BDD：拆解为 UI/交互场景 | Bash, Read, Write, Edit, Glob, Grep |
| `003-4-api-contract` | 定义前后端接口契约，回填更新 BDD | Bash, Read, Write, Edit, Glob, Grep |
| `003-5-1-backend-tdd-java` | 后端 TDD（Java/SpringBoot）Red-Green-Refactor | Bash, Read, Write, Edit, Glob, Grep |
| `003-6-1-ui-state-definition` | UI 状态定义 + 可交互 HTML 原型 | Bash, Read, Write, Edit, Glob, Grep |
| `003-6-2-frontend-cdd` | 前端 CDD（Vue/Storybook）Atomic Design | Bash, Read, Write, Edit, Glob, Grep |
| `003-7-e2e-test` | E2E 端到端链路验证 + 验收清单 | Bash, Read, Write, Edit, Glob, Grep |

### 999 系列 — 工具与辅助

| Skill | 功能 | allowed-tools | 输出 |
|-------|------|---------------|------|
| `999-1-git-commit` | 按规范格式提交代码（仅 commit，不推送） | Bash, Read, Edit, Glob, Grep | Git 本地 commit |
| `999-2-git-push` | 按规范格式提交并推送 | Bash, Read, Glob, Grep | Git commit + push |
| `999-other-110-requirement-planning` | 需求拆解为页面组文档和领域模块文档 | — | `M{N}P{N}-<名称>.md`、`domains/M{N}-<名称>.md` |
| `999-other-120-learn` | 将使用中发现的问题/修正写入技能补充文件 | Bash, Read, Write, Edit, Glob, Grep | `<skill>/03-user-prompt-submit/` 下补充文件 |

---

## 配置说明

### Skill 注册表（`registry.conf`）

每行一个 Skill 名称，`skill-register.sh` 在 `16-Stop` 事件时自动扫描并更新：

```
001-1-issue-init
001-2-issue
001-3-issue-discuss
...
999-2-git-push
```

### SKILL.md Frontmatter

每个 Skill 的 `SKILL.md` 定义元信息，控制 Skill 行为：

```yaml
---
name: 001-4-issue-claim
description: 原子领取 GitHub Issue，防止多 agent 冲突
allowed-tools:
  - Bash
  - Read
alwaysApply: false
---
```

| 字段 | 说明 |
|------|------|
| `name` | Skill 标识，与目录名一致 |
| `description` | 功能描述，用于 `/help` 展示 |
| `allowed-tools` | 工具白名单，`enforce_boundary.sh` 在 `05-PreToolUse` 时拦截未授权工具 |
| `alwaysApply` | 是否自动激活（当前未使用自动激活逻辑） |

### Issue 标签配置（`001-1-issue-init/labels.conf`）

```ini
[types]
bug, enhancement, documentation, performance, chore, good first issue, duplicate, help wanted, invalid, question, wontfix, skill-system

[workflow]
in-progress, fixing, ready-for-pr, pr-created, testing, reviewing, rejected

[priority]
P0 (Critical), P1 (High), P2 (Medium), P3 (Low)
```

### 活跃标记文件（`.active`）

多行格式，每行 `session_id|skill_name`：

```
abc123-def456|001-5-issue-fix
xyz789-ghi012|001-7-issue-pr
```

通过 `active.sh` 进行 CRUD 操作，使用 `lock.sh` 的 `mkdir` 原子操作实现跨进程文件锁。

### 共享模块

| 模块 | 功能 | 被谁引用 |
|------|------|---------|
| `skills/active.sh` | `.active` 文件 CRUD | 所有 Skill 脚本 |
| `skills/backend.sh` | 后端 API 调用（`_call_backend`、`_get_session_id`、`update_issue_status`） | 001 系列脚本 |
| `skills/log.sh` | 双写日志：统一日志 + Skill 专属日志 | 所有 Skill 脚本 |
| `skills/lock.sh` | 文件锁（`lock_acquire` / `lock_release`） | active.sh |
| `skills/enforce_boundary.sh` | 解析 SKILL.md 的 `allowed-tools`，拦截未授权工具 | 05-PreToolUse |

### 后端降级策略

`skills/backend.sh` 中的 `_require_backend()` 返回三级状态：

| 返回值 | 含义 | Skill 行为 |
|--------|------|-----------|
| 0 | 后端可用 | 使用后端 API |
| 1 | 后端不可用 | 降级到 GitHub Label 方案 |
| 2 | 后端未配置 | 降级到 GitHub Label 方案 |

> Shell 侧始终**静默降级**：Go 后端是增强（原子性、多 Agent 协调），不是依赖。
