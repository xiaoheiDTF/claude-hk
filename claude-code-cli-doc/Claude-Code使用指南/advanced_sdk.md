# Claude-Code / Advanced / Sdk

> 来源: claudecn.com

# Claude Code SDK

Claude Code SDK 允许以[ 编程](#)方式与 Claude Code 集成，构建自动化工作流和自定义工具。

## 应用场景

- 自动化工作流 - 批量处理代码任务
- 自定义工具集成 - 与现有开发工具链集成
- CI/CD 集成 - 在持续集成流程中使用
- 构建自定义界面 - 开发定制化的交互界面
## Headless Mode

使用 `-p` 标志以非交互模式运行：

```bash
# 基本用法
claude -p "解释这个函数的作用"

# 指定工作目录
claude -p "分析项目结构" --cwd /path/to/project
```

## JSON 输出
使用 `--output-format stream-json` 获取结构化输出：

```bash
claude -p "列出所有 TODO 注释" --output-format stream-json
```

输出格式：

```json
{"type":"text","content":"找到以下 TODO 注释："}
{"type":"text","content":"1. src/main.ts:15 - TODO: 添加错误处理"}
{"type":"result","result":"完成扫描，共发现 3 个 TODO"}
```

## 限制工具
使用 `--allowedTools` 限制可用工具：

```bash
# 只允许读取和搜索
claude -p "查找所有测试文件" --allowedTools Read,Grep,glob

# 禁止文件修改
claude -p "代码审查" --allowedTools Read,Grep,finder
```

## 实用模式示例

### 自定义 Linter

```bash
#!/bin/bash
# custom-lint.sh

claude -p "检查以下代码的潜在问题并以 JSON 格式输出：
$(cat $1)" --output-format stream-json | \
  jq -r 'select(.type=="result") | .result'
```

### 自动 Issue 分流

```bash
#!/bin/bash
# triage-issue.sh

ISSUE_BODY="$1"
RESULT=$(claude -p "分析以下 Issue 并分类：
$ISSUE_BODY

输出格式：
- 类型：bug/feature/question
- 优先级：high/medium/low
- 相关模块：<module-name>" --output-format stream-json)

echo "$RESULT" | jq -r 'select(.type=="result") | .result'
```

### CI 集成
GitHub Actions 示例：

```yaml
name: Code Review

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install Claude Code
        run: npm install -g @anthropic/claude-code

      - name: Run Code Review
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: |
          # 获取变更文件
          CHANGED_FILES=$(git diff --name-only origin/main...HEAD)
          
          # 对每个文件进行审查
          for file in $CHANGED_FILES; do
            if [[ -f "$file" ]]; then
              echo "Reviewing: $file"
              claude -p "Review this file for issues: $file" \
                --allowedTools Read,Grep \
                --output-format stream-json >> review-results.json
            fi
          done

      - name: Upload Review Results
        uses: actions/upload-artifact@v4
        with:
          name: review-results
          path: review-results.json
```

## 安全最佳实践

- API Key 管理使用环境变量存储 API Key
- 在 CI 中使用 Secrets 管理
- 避免在脚本中硬编码
- 工具限制生产环境限制可用工具
- 禁用不必要的文件写入权限
- 输入验证清理用户输入
- 避免命令注入风险
- 审计日志记录所有 SDK 调用
- 监控异常使用模式
## 错误处理

```bash
#!/bin/bash

RESULT=$(claude -p "$PROMPT" --output-format stream-json 2>&1)
EXIT_CODE=$?

if [ $EXIT_CODE -ne 0 ]; then
  echo "Error: Claude Code failed with exit code $EXIT_CODE"
  echo "$RESULT"
  exit 1
fi

echo "$RESULT" | jq -r 'select(.type=="result") | .result'
```
