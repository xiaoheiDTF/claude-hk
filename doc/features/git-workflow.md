# Git 工作流

## 提交规范

所有 commit 使用中文格式：

```
<type>: <主描述>

- 具体修改描述1
- 具体修改描述2
```

### Type 列表

| Type | 用途 |
|------|------|
| `fix` | 修复 bug |
| `feat` | 新功能 |
| `update` | 功能更新 |
| `style` | 格式调整（不影响逻辑） |
| `refactor` | 重构 |
| `perf` | 性能优化 |
| `test` | 测试相关 |
| `docs` | 文档 |
| `revert` | 回退 |
| `build` | 构建相关 |
| `chore` | 杂项 |

## 分组提交规则

分组优先级：type → 目录/模块 → 功能关联 → 影响范围

**禁止** `git add .` 或 `git add -A`，必须按分组 `git add` 具体文件。

示例：

```bash
# 按类型分组
git add .claude/skills/003-5-issue-fix/
git add .claude/skills/registry.conf

# 按目录分组
git add doc/features/
git add doc/mechanisms/
```

## Skill 004：提交并推送

```
/004-git-push
```

自动执行：
1. 检查当前分支状态
2. 按分组规则 add 文件
3. 生成符合规范的中文 commit message
4. commit + push

## Skill 005：仅本地提交

```
/005-git-commit
```

与 004 相同的提交规范，但不推送。适用于：
- 需要多次提交后再统一 push
- 本地暂存，等待验证后再推送
