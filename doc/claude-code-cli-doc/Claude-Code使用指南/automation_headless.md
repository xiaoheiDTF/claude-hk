# Claude-Code / Automation / Headless

> 来源: claudecn.com

# Headless 模式

Headless 模式让 Claude Code 在无交互环境中运行，适用于 CI/CD 管道、自动化[ 脚本](#)和批处理任务。

## 基本用法

使用 `-p` 参数传递提示，Claude 处理任务后退出：

```bash
# 基本查询
claude -p "解释这个项目的架构"

# JSON 输出
claude -p "列出所有 TODO 注释" --output-format json

# 流式输出
claude -p "审查最近的更改" --output-format stream-json
```

---

## 输出格式

### 文本（默认）

```bash
claude -p "总结 README.md"
```

输出纯文本响应。

### JSON

```bash
claude -p "分析代码质量" --output-format json
```

输出结构化 JSON：

```json
{
  "type": "result",
  "result": "分析结果...",
  "cost": 0.0123,
  "duration": 5.2
}
```

### 流式 JSON

```bash
claude -p "重构这个函数" --output-format stream-json
```

实时输出每个步骤的 JSON 对象。

---

## 工具限制

在自动化环境中应限制可用工具：

```bash
# 只允许只读工具
claude -p "分析代码库" --allowedTools "Read,Glob,Grep"

# 禁止特定工具
claude -p "审查代码" --disallowedTools "Write,Edit,Bash"
```

### 常用只读工具列表

| 工具 | 说明 |
| --- | --- |
| `Read` | 读取文件 |
| `Glob` | 文件模式匹配 |
| `Grep` | 内容搜索 |
| `Bash(git log:*)` | 允许 git log 命令 |
| `Bash(git diff:*)` | 允许 git diff 命令 |

---

## 权限模式

```bash
# 计划模式（只读）
claude --permission-mode plan -p "分析认证系统的潜在问题"

# 自动接受编辑（谨慎使用）
claude --permission-mode acceptEdits -p "添加类型注解"
```

---

## 会话管理

### 恢复会话

```bash
# 恢复最近会话
claude --resume -p "继续之前的工作"

# 恢复特定会话
claude --session-id abc123 -p "完成剩余任务"
```

### 会话超时

```bash
# 设置最大执行时间
timeout 600 claude -p "生成测试用例"
```

---

## 环境变量

| 变量 | 说明 |
| --- | --- |
| `ANTHROPIC_API_KEY` | API 密钥 |
| `CLAUDE_CODE_USE_BEDROCK` | 使用 AWS Bedrock |
| `CLAUDE_CODE_USE_VERTEX` | 使用 Google Vertex AI |
| `MAX_TURNS` | 最大对话轮数 |
| `NO_COLOR` | 禁用颜色输出 |

---

## 管道集成

### 与其他命令组合

```bash
# 分析 git diff 输出
git diff HEAD~5 | claude -p "审查这些更改"

# 处理文件内容
cat error.log | claude -p "解释这些错误"
```

### 脚本示例

```bash
#!/bin/bash

# 自动代码审查脚本
review_code() {
  local file="$1"
  
  claude -p "审查 $file 的代码质量和安全性" \
    --allowedTools "Read,Glob,Grep" \
    --output-format json \
    | jq -r '.result'
}

# 批量审查
for file in src/*.py; do
  echo "审查: $file"
  review_code "$file" >> review_report.md
done
```

---

## CI/CD 示例

### GitHub Actions

```yaml
- name: 代码审查
  run: |
    claude -p "审查 PR #${{ github.event.pull_request.number }} 的代码" \
      --allowedTools "Read,Glob,Grep" \
      --output-format json > review.json
```

### Shell 脚本

```bash
#!/bin/bash
set -e

# 生成测试
claude -p "为 src/auth.py 生成单元测试" \
  --permission-mode plan \
  --output-format json > tests.json

# 验证结果
if [ "$(jq -r '.type' tests.json)" = "result" ]; then
  echo "测试生成成功"
else
  echo "测试生成失败"
  exit 1
fi
```

---

## 最佳实践

- 限制工具访问：始终使用 --allowedTools 限制为必要的只读工具
- 设置超时：使用 timeout 或 --max-turns 防止无限运行
- 使用 JSON 输出：便于程序处理结果
- 隔离环境：在容器或沙箱中运行
- 记录日志：保存所有 Claude 操作用于审计
---

## 下一步
[GitHub Actions集成到 GitHub 工作流
](../github-actions/)[Hooks 系统自定义自动化逻辑
](../../advanced/hooks/)
