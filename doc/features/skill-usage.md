# Skill 使用教程

## 调用方式

在 Claude Code 对话中输入 `/技能名` 加参数：

```
/003-4-issue-claim #17          # 领取 issue 17
/003-5-issue-fix #17            # 创建分支开始解决
/004-git-push                   # 提交并推送
```

## 调用流程

```
1. 输入 /技能名 参数
   ↓
2. Hooks 管道捕获 UserPromptSubmit 事件
   ↓
3. skill-inject.sh 提取技能名，匹配 registry.conf
   ↓
4. 运行 skill 的 03UserPromptSubmit.sh
   → 输出动态上下文（issue 信息、分支状态等）
   ↓
5. 上下文注入到当前会话
   ↓
6. Claude 根据 SKILL.md 定义执行任务
   ↓
7. 完成后 16Stop.sh 清理会话状态
```

## 典型使用场景

### 场景 1：从零开始解决一个 Issue

```
/003-4-issue-claim #17        # 领取
/003-5-issue-fix #17          # 创建分支，开始开发
# ... 编码 ...
/003-6-issue-done #17         # 标记完成
/003-7-issue-pr #17           # 创建 PR
```

### 场景 2：写个 Python 测试脚本

```
/001-testcode-python 编写一个登录接口的自动化测试
```

### 场景 3：记录一个临时文档

```
/002-otherdoc 记录今天的会议纪要
```

### 场景 4：规范化提交代码

```
/005-git-commit    # 只提交到本地
/004-git-push      # 提交并推送到远程
```

## 常见问题

### Skill 未注册

如果输入 `/技能名` 没有反应，检查 `.claude/skills/registry.conf` 是否包含该技能名。Skill 在每次 Stop 事件时自动注册（`skill-register.sh` 扫描目录）。

### 命令不识别

确保以 `/` 开头，技能名精确匹配 registry.conf 中的条目。例如 `/003-2-issue` 而不是 `/issue`。

### 无参数调用

大部分 Skill 支持无参数调用，此时会列出可操作的列表供选择。
