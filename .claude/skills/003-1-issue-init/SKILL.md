---
name: 003-1-issue-init
description: 初始化 GitHub 项目的 issue 标签体系（一次性）
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Write
---

# Issue Init Skill

初始化当前 GitHub 项目的 issue 标签体系。

## 规则

1. 只能执行一次，重复执行提示已初始化
2. 标签来源：本目录下 `labels.conf`，不使用其他标签
3. 已存在的标签跳过，不覆盖颜色和描述
4. 后续更新：修改 `labels.conf` 后重新运行会增量添加

## 操作流程

1. 检查 `.github/.issue-initialized` 标记文件是否存在
2. 读取 `labels.conf` 中的标签定义
3. 获取 GitHub remote 信息（owner/repo）
4. 逐条创建标签：`gh label create <name> --color <color> --description <desc>`
5. 创建标记文件 `.github/.issue-initialized`
6. 输出初始化结果

## gh 命令参考

```bash
# 创建标签
gh label create "bug" --color "d73a4a" --description "Something isn't working"

# 列出已有标签
gh label list

# 删除多余标签
gh label delete "wontfix" --yes
```
