---
name: 001-5-issue-fix
description: 根据 issue 创建分支并开始开发
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Edit
  - Write
  - Glob
  - Grep
---

# Issue 解决 Skill

根据 issue 创建对应分支，开始开发。

## 操作流程

1. 用户输入 `/001-5-issue-fix #N`
2. 检查当前用户是否为 issue 的 assignee
3. 根据 issue labels 判断分支类型（bug → fix，enhancement → feat，其他 → chore）
4. 从 issue title 自动生成分支名
5. 创建分支：`<type>/issue-<N>-<简短描述>`
6. 在 issue 中 comment："开始解决，分支: `<branch>`"
7. 输出分支信息，开始开发
8. 开发完成后执行 `/001-6-issue-done #N` 标记完成

## 分支命名规则

```
<type>/issue-<编号>-<kebab-case-简短描述>

示例:
  bug: 登录页面白屏     → fix/issue-9-login-white-screen
  feat: 新增用户注册    → feat/issue-10-user-register
  chore: 更新依赖版本   → chore/issue-11-update-deps
```

- title 中的中文翻译为英文拼音或关键词
- 全部小写，空格替换为 `-`
- 长度控制在 60 字符以内

## 规则

1. 必须先 claim issue 才能执行 fix
2. 分支名自动生成，用户也可以手动指定
3. 如果分支已存在则切换到该分支
4. 确保在 main 分支上创建新分支

## gh 命令参考

```bash
# 获取 issue 信息
gh issue view <编号> --json title,labels,assignees

# 创建分支
git checkout main
git pull origin main
git checkout -b <branch-name>

# 记录开始
gh issue comment <编号> --body "开始解决，分支: \`<branch>\`"
```
