---
title: 当前 Issue 闭环机制缺陷分析（结合 Claude Code Hooks 能力评估）
labels: skill-system,enhancement
assignee: 
priority: P1
status: draft
created: 2026-05-21
---

## 描述

基于对 Claude Code CLI Hooks 官方文档（`07-Hooks参考.md`、`advanced_hooks.md`、`advanced_hooks-recipes.md`）的研读，重新评估当前 Issue 闭环机制的缺陷。**关键发现：Hooks 只能在 Claude Code 生命周期事件（如 PreToolUse、PostToolUse、Stop、SessionStart 等）触发时执行，无法主动监听 GitHub 事件，也无法在没有活跃会话时执行定时任务。** 因此，之前的缺陷清单需要按 "Hooks 可解决 / Skill 可解决 / 需 GitHub Actions / 当前不可行" 重新分类。

---

## Claude Code Hooks 能力边界

### Hooks 能做什么

| 能力 | 说明 | 适用场景 |
|------|------|---------|
| **PreToolUse 拦截** | 工具执行前拦截、阻止（exit 2）、修改参数、注入上下文 | 阻断危险/不合规操作，如 main 分支直接编辑、无 test plan 的 PR 合并 |
| **PostToolUse 自动化** | 工具执行后自动执行脚本 | commit 后自动关联 issue、merge 后自动清理分支、edit 后自动格式化 |
| **PostToolBatch** | 一批工具调用完成后触发 | 批量 edit 后统一跑 lint/test |
| **Stop / SessionEnd** | 每轮/会话结束时触发 | 会话结束时检查未完成任务、生成交接文档 |
| **SessionStart** | 会话开始时触发 | 恢复环境变量、检查 in-progress issue 状态 |
| **SubagentStart/Stop** | 子代理开始/结束时 | 子代理任务前后注入规则 |
| **TaskCreated/TaskCompleted** | 任务创建/完成时 | 阻止不合规的任务标记完成 |
| **FileChanged** | 监视文件变更时 | 本地文件与 GitHub 状态同步检测 |

### Hooks 不能做什么

| 限制 | 说明 | 影响 |
|------|------|------|
| **无法主动监听 GitHub 事件** | Hooks 不监听 GitHub Webhook，无法感知 issue 被评论、PR 被 review 等 | 无法做真正的"实时状态同步" |
| **无法跨会话定时执行** | 没有 cron 能力，SessionEnd 后不再触发 | 超时释放、stale 清理等需要 GitHub Actions |
| **无法持久化复杂状态** | 只能通过写文件/环境变量持久化 | 状态机流转需要借助文件锁或外部存储 |
| **无法直接与其他 Agent 通信** | TeammateIdle 可以感知队友空闲，但无法主动推送消息 | 多 Agent 协作需要轮询 GitHub 状态 |

> 官方文档来源：`07-Hooks参考.md` 第 1~42 行 Hook 生命周期事件表

---

## 缺陷重分类与可落地方案

### 第一类：Hooks + Skill 可解决（推荐优先落地）

#### #1 PR 合并后无分支清理 ❌ → ✅ 可通过 PostToolUse 解决

**当前缺陷**：003-6-issue-pr merge 后直接结束，不清理分支。

**Hooks 方案**：
```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "#!/bin/bash\ninput=$(cat)\ncmd=$(echo \"$input\" | jq -r '.tool_input.command // \"\"')\nif echo \"$cmd\" | grep -qE '^gh pr merge'; then\n  branch=$(git branch --show-current)\n  echo \"[Hook] PR merged, cleaning up branch: $branch\" >&2\n  git push origin --delete \"$branch\" 2>/dev/null || true\n  git checkout main 2>/dev/null || true\n  git branch -d \"$branch\" 2>/dev/null || true\nfi\necho \"$input\""
          }
        ]
      }
    ]
  }
}
```

> 参考：`advanced_hooks-recipes.md` 第 176~189 行 "禁止在 main 分支直接编辑" 配方思路

---

#### #2 commit 与 issue 无强制关联 ❌ → ✅ 可通过 PostToolUse 解决

**当前缺陷**：005-git-commit / 004-git-push 无 issue 编号要求。

**Hooks 方案**：
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "#!/bin/bash\ninput=$(cat)\ncmd=$(echo \"$input\" | jq -r '.tool_input.command // \"\"')\n# 检查 git commit 命令\nif echo \"$cmd\" | grep -qE '^git commit'; then\n  branch=$(git branch --show-current)\n  if echo \"$branch\" | grep -qE 'issue-[0-9]+'; then\n    issue_num=$(echo \"$branch\" | grep -oE 'issue-[0-9]+' | sed 's/issue-//')\n    if ! echo \"$cmd\" | grep -q \"Refs #$issue_num\"; then\n      echo \"[Hook] WARNING: commit message should include 'Refs #$issue_num'\" >&2\n      echo \"[Hook] Current branch $branch maps to issue #$issue_num\" >&2\n    fi\n  fi\nfi\necho \"$input\""
          }
        ]
      }
    ]
  }
}
```

**效果**：不阻断，仅提醒。如需强制阻断，改用 `exit 2`。

> 参考：`advanced_hooks.md` 第 289~325 行 "阻止危险命令" 示例

---

#### #3 禁止在 main 分支直接编辑 ❌ → ✅ 可通过 PreToolUse 阻断

**当前缺陷**：当前机制没有任何分支保护，agent 可能直接在 main 上修改。

**Hooks 方案**（官方已有现成配方）：
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit|MultiEdit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "[ \"$(git branch --show-current)\" != \"main\" ] || { echo '{\"hookSpecificOutput\":{\"hookEventName\":\"PreToolUse\",\"permissionDecision\":\"deny\",\"permissionDecisionReason\":\"Cannot edit files on main branch. Create a feature branch first via 003-5-issue-fix.\"}}' >&2; exit 2; }",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
```

> 参考：`advanced_hooks-recipes.md` 第 176~189 行现成配方

---

#### #4 fix 阶段 main 更新后无 rebase 提醒 ❌ → ✅ 可通过 PreToolUse 提醒

**当前缺陷**：分支开发期间 main 可能有新提交，提 PR 时容易产生冲突。

**Hooks 方案**：
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "#!/bin/bash\ninput=$(cat)\ncmd=$(echo \"$input\" | jq -r '.tool_input.command // \"\"')\nif echo \"$cmd\" | grep -qE '^gh pr create'; then\n  branch=$(git branch --show-current)\n  if [ \"$branch\" != \"main\" ]; then\n    behind=$(git rev-list --count HEAD..origin/main 2>/dev/null || echo 0)\n    if [ \"$behind\" -gt 0 ]; then\n      echo \"[Hook] WARNING: Your branch is $behind commits behind origin/main\" >&2\n      echo \"[Hook] Recommend: git fetch origin && git rebase origin/main\" >&2\n    fi\n  fi\nfi\necho \"$input\""
          }
        ]
      }
    ]
  }
}
```

---

#### #5 issue 创建前重复检测 ❌ → ✅ 可通过 PreToolUse 提醒

**当前缺陷**：003-2-issue 直接创建，不检查相似 issue。

**Hooks 方案**：在 `gh issue create` 执行前搜索相似标题：
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "#!/bin/bash\ninput=$(cat)\ncmd=$(echo \"$input\" | jq -r '.tool_input.command // \"\"')\nif echo \"$cmd\" | grep -qE '^gh issue create'; then\n  title=$(echo \"$cmd\" | grep -oP '(?<=--title \")[^\"]+' | head -1)\n  if [ -n \"$title\" ]; then\n    echo \"[Hook] Checking for duplicate issues...\" >&2\n    gh issue list --search \"$title\" --limit 5 --json number,title --jq '.[] | \"  #\(.number): \(.title)\"' >&2 || true\n  fi\nfi\necho \"$input\""
          }
        ]
      }
    ]
  }
}
```

---

### 第二类：需要 GitHub Actions 解决（Hooks 无法定时执行）

#### #6 in-progress 超时释放 ❌ → 需 GitHub Actions

**原因**：Hooks 没有 cron 能力，SessionEnd 后不会触发。

**方案**：`.github/workflows/stale-issue.yml`
```yaml
name: Stale Issue Release
on:
  schedule:
    - cron: '0 0 * * *'  # 每天运行
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - run: |
          gh issue list --label in-progress --search "updated:<$(date -d '7 days ago' +%Y-%m-%d)" --json number | \
          jq -r '.[].number' | \
          xargs -I {} sh -c 'gh issue edit {} --remove-label in-progress --add-label stale && gh issue comment {} --body "自动释放：超过7天无活动，移除 assignee"'
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

#### #7 stale issue 清理 ❌ → 需 GitHub Actions

**原因**：同上，需要定时任务。

**方案**：使用官方 `actions/stale` action：
```yaml
- uses: actions/stale@v9
  with:
    stale-issue-message: 'This issue has been inactive for 30 days.'
    stale-issue-label: 'stale'
    days-before-stale: 30
    days-before-close: 7
```

---

### 第三类：Skill 层面可解决（无需 Hooks）

#### #8 claim 前不检查 issue 状态

**改进**：在 003-4-issue-claim 的 SKILL.md 中增加前置检查步骤：
```bash
# 新增步骤
ght issue view <N> --json state,labels --jq '.state'  # 必须为 open
gh issue view <N> --json labels --jq '.labels[].name' | grep -q resolved && exit 1
```

#### #9 claim 失败后的回退指引

**改进**：003-4-issue-claim 验证失败后：
```bash
# 当前：仅输出领取结果
# 改进：输出当前 assignee → 列出 "ready-for-claim" 标签的 issue → 建议切换
```

#### #10 fix 无"解决问题完成"标记

**改进**：003-5-issue-fix 增加子命令 `done`，已在 #15 中规划。

#### #11 PR 打回后的分支处理规则

**改进**：003-6-issue-pr 打回流程中明确：
- 保留原分支继续修改（推荐）
- 删除后重建（如需要重新设计）

---

### 第四类：当前机制不可行（需要架构变更或人工介入）

#### #12 issue 依赖关系管理

**评估**：Hooks 无法感知 GitHub 上其他 issue 的状态变更。需要 GitHub Projects 或外部系统。当前不可行。

#### #13 多 Agent 实时协作通知

**评估**：TeammateIdle 可以感知队友空闲，但无法主动推送消息。Agent 之间只能通过 GitHub issue 状态轮询实现间接通信。当前不可行。

#### #14 里程碑/迭代规划

**评估**：纯 Hooks/Skill 无法实现里程碑管理，需要人工或 GitHub Projects。当前不可行。

#### #15 issue 关闭后知识沉淀自动化

**评估**：PostToolUse 可以在 `gh issue close` 时触发，生成摘要文件。但"知识沉淀"的质量需要人工审核，全自动可能产生垃圾文档。

**折中方案**：关闭时生成草稿摘要到 `doc/otherDoc/closed-issues/`，人工审核后归档。

---

## 改进优先级（结合 Hooks 能力）

| 优先级 | 缺陷 | 解决方式 | 工作量 |
|--------|------|---------|--------|
| **P0（立即）** | #3 禁止 main 分支编辑 | PreToolUse 阻断 | 1 个 Hook |
| **P0** | #1 PR 合并后分支清理 | PostToolUse 自动化 | 1 个 Hook |
| **P1（本轮）** | #2 commit 关联 issue | PostToolUse 提醒 | 1 个 Hook |
| **P1** | #4 rebase 提醒 | PreToolUse 提醒 | 1 个 Hook |
| **P1** | #5 重复检测 | PreToolUse 提醒 | 1 个 Hook |
| **P1** | #8~#11 Skill 改进 | 修改 SKILL.md | 4 个文件 |
| **P2（后续）** | #6 超时释放 | GitHub Actions | 1 个 workflow |
| **P2** | #7 stale 清理 | GitHub Actions | 1 个 workflow |
| **P3（暂不）** | #12~#14 架构级需求 | 需要 GitHub Projects | 超出当前范围 |

---

## 建议：建立 Hooks 护栏体系

基于官方文档 `advanced_hooks-recipes.md` 的"把经验变成自动化护栏"思路，建议为 issue 机制建立三层护栏：

### 第一层：阻断型（exit 2）
- ✅ 禁止 main 分支直接编辑
- ✅ 禁止无 test plan 的 PR 合并（配合 #15）
- ✅ 禁止 claim 已关闭/已解决的 issue

### 第二层：提醒型（stderr 输出）
- ⚠️ commit 缺少 issue 关联时提醒
- ⚠️ PR 创建前分支落后 main 时提醒
- ⚠️ issue 创建前发现相似 issue 时提醒
- ⚠️ fix 阶段检测到未提交变更时提醒

### 第三层：自动化型（PostToolUse 后台执行）
- 🤖 PR merge 后自动清理分支
- 🤖 edit 后自动格式化/lint（已有官方配方）
- 🤖 测试文件修改后自动跑相关测试（已有官方配方）

> 参考：`advanced_hooks-recipes.md` 第 7~11 行 "建议先用'提醒型'，再逐步上'阻断型'"

---

## 涉及文件

| 文件 | 改动说明 |
|------|---------|
| `.claude/settings.json` | 新增 Hooks 配置（三层护栏） |
| `.claude/hooks/validate-branch.sh` | 阻断 main 分支编辑 |
| `.claude/hooks/cleanup-branch.sh` | PR merge 后清理分支 |
| `.claude/hooks/check-issue-ref.sh` | commit 关联 issue 提醒 |
| `.claude/hooks/check-rebase.sh` | PR 前 rebase 提醒 |
| `.claude/hooks/check-duplicate-issue.sh` | issue 重复检测 |
| `.github/workflows/stale-issue.yml` | 超时释放 + stale 清理 |
| `.claude/skills/003-4-issue-claim/SKILL.md` | 增加前置状态检查、失败回退 |
| `.claude/skills/003-5-issue-fix/SKILL.md` | 增加 done 子命令（#15） |
| `.claude/skills/003-6-issue-pr/SKILL.md` | 明确打回分支处理规则 |

## 验收标准

- [ ] main 分支编辑被 PreToolUse 阻断（已测试验证）
- [ ] PR merge 后远程分支自动删除
- [ ] commit 时如分支含 issue 编号且 message 未关联，输出提醒
- [ ] 提 PR 前如分支落后 main，输出 rebase 提醒
- [ ] claim 前检查 issue state 为 open
- [ ] claim 失败时输出回退指引和可领取 issue 列表
- [ ] GitHub Actions 定时释放超 7 天 in-progress issue
- [ ] 所有 Hooks 脚本有日志记录（`/tmp/claude-hooks.log`）

## 发布记录

- Issue #18: https://github.com/xiaoheiDTF/claude-hk/issues/18 (发布于 2026-05-21 22:42)

