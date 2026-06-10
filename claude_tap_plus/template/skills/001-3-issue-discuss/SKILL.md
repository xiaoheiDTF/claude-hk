---
name: 001-3-issue-discuss
description: 拉取 GitHub Issue 内容进行讨论，支持评论互动
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Write
---

# Issue 讨论 Skill

拉取指定 issue 的完整内容（标题、描述、评论、标签），注入到当前会话作为上下文，用于方案讨论。

## 操作流程

1. 用户输入 `/001-3-issue-discuss #N`
2. 拉取 issue 内容：`gh issue view N`
3. 拉取评论：`gh api repos/{owner}/{repo}/issues/N/comments`
4. 将完整内容注入会话上下文
5. 用户在会话中讨论方案
6. 讨论结论通过 `gh issue comment N --body "..."` 写入

## 规则

1. 必须指定 issue 编号
2. 拉取内容包括：title、body、labels、assignees、state、所有 comments
3. 讨论结论以 comment 形式写回 issue

## gh 命令参考

```bash
# 查看详情
gh issue view <编号>

# 查看评论
gh api repos/{owner}/{repo}/issues/<编号>/comments --jq '.[].body'

# 发表评论
gh issue comment <编号> --body "<内容>"
```
