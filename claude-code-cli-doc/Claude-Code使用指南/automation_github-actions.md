# Claude-Code / Automation / Github-Actions

> 来源: claudecn.com

# GitHub Actions

Claude Code GitHub Actions 将 AI 自动化引入 GitHub 工作流。只需在 PR 或 Issue 中 `@claude` 提及，Claude 就能分析代码、创建 PR、实现功能、修复 Bug。

## 功能特点

- 即时创建 PR：描述需求，Claude 完成所有必要更改
- 自动代码实现：一条命令将 Issue 转为可工作的代码
- 遵循团队标准：Claude 遵守 CLAUDE.md 指南和现有代码模式
- 安全默认：代码在 GitHub 运行器上执行，不离开 GitHub
---

## 快速设置

### 方式一：自动安装（推荐）

在 Claude Code 终端中运行：

```bash
/install-github-app
```

此命令引导你完成 GitHub App 安装和必要的 secrets 配置。

需要仓库管理员权限。GitHub App 会请求 Contents、Issues、Pull requests 的读写权限。

### 方式二：手动设置

- 安装 Claude GitHub App：https://github.com/apps/claude
- 添加 API Key：在仓库 Settings → Secrets and variables → Actions 中添加 ANTHROPIC_API_KEY
- 创建工作流文件 .github/workflows/claude.yml：
```yaml
name: Claude Code
on:
  issue_comment:
    types: [created]
  pull_request_review_comment:
    types: [created]
jobs:
  claude:
    runs-on: ubuntu-latest
    steps:
      - uses: anthropics/claude-code-action@v1
        with:
          anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
```

---

## 使用示例

### 在 Issue 或 PR 评论中

```
@claude implement this feature based on the issue description
@claude how should I implement user authentication for this endpoint?
@claude fix the TypeError in the user dashboard component
```

Claude 会自动分析上下文并做出响应。

### 使用斜杠命令

```yaml
- uses: anthropics/claude-code-action@v1
  with:
    anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
    prompt: "/review"
    claude_args: "--max-turns 5"
```

### 自定义自动化

```yaml
name: Daily Report
on:
  schedule:
    - cron: "0 9 * * *"
jobs:
  report:
    runs-on: ubuntu-latest
    steps:
      - uses: anthropics/claude-code-action@v1
        with:
          anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
          prompt: "生成昨天提交和待处理 Issue 的摘要"
          claude_args: "--model claude-opus-4-5-20251101"
```

---

## 进阶案例：PR 自动审查 + 定时维护
下面是一套更接近“团队默认工作流”的 GitHub Actions 组合思路：把审查标准写成 `.claude/agents/code-reviewer.md`，再在 CI 里调用它；同时用定时任务做文档同步、质量巡检、依赖审计。

在 Actions 里跑 Claude Code 时，务必把权限收敛到最小：

- permissions 只给需要的范围（例如只读 contents + 可写 pull-requests）
- claude_args --allowedTools 也只放“完成任务所需”的命令集合
PR 审查工作流可以把工具限制在 `git` 与 `gh pr ...` 相关命令，避免动作越界。

### 1) PR 自动审查（按自定义 checklist 输出分级建议）

工作流文件示例：`your-repo/.github/workflows/pr-claude-code-review.yml`。核心片段（节选）：

```yaml
- name: Claude Code Review
  uses: anthropics/claude-code-action@beta
  with:
    anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
    model: claude-opus-4-5-20251101
    prompt: |
      Review this Pull Request using the standards in .claude/agents/code-reviewer.md

      1. Read .claude/agents/code-reviewer.md for the full checklist
      2. Run `git diff origin/${{ steps.pr-info.outputs.base_ref }}...HEAD` to see changes
      3. Apply the review checklist to modified files
      4. Provide feedback organized by severity (Critical, Warning, Suggestion)
    claude_args: |
      --max-turns 10
      --allowedTools "Read,Glob,Grep,Bash(git:*),Bash(gh pr comment:*),Bash(gh pr diff:*),Bash(gh pr view:*)"
```

落地建议：

- permissions 与 --allowedTools 越小越安全；先从“只读审查 + 评论”开始，再考虑“自动修复 + 开 PR”。
- 把审查 checklist 放进 .claude/agents/，方便团队版本控制与复用。
### 2) 定时文档同步（只修“实际错误”的文档）

工作流文件示例：`your-repo/.github/workflows/scheduled-claude-code-docs-sync.yml`。它先筛出一段时间内变更过的代码文件，再让 Claude 检查“相关文档是否已经不对”，**只在确实不对时才创建 PR**（节选）：

```yaml
on:
  schedule:
    - cron: "0 9 1 * *"

- name: Claude Documentation Sync
  uses: anthropics/claude-code-action@beta
  with:
    base_branch: main
    branch_prefix: claude/docs-sync-
    prompt: |
      - Fix ONLY what's broken
      - ONLY update docs when they are incorrect or misleading
      - If no problems are found, report that and DO NOT create a PR
```

### 3) 其他定时维护任务（参考）

- scheduled-claude-code-quality.yml：随机抽取目录做质量巡检与修复，并在通过 npm run lint 后发 PR
- scheduled-claude-code-dependency-audit.yml：依赖过期与安全审计，保守更新并跑 npm test
---

## 配置参数

| 参数 | 说明 | 必填 |
| --- | --- | --- |
| `prompt` | 给 Claude 的指令（文本或斜杠命令） | 否* |
| `claude_args` | 传递给 Claude Code 的 CLI 参数 | 否 |
| `anthropic_api_key` | Claude API 密钥 | 是** |
| `github_token` | GitHub 令牌 | 否 |
| `trigger_phrase` | 自定义触发词（默认 “@claude”） | 否 |
| `use_bedrock` | 使用 AWS Bedrock | 否 |
| `use_vertex` | 使用 Google Vertex AI | 否 |
*当省略 prompt 时，Claude 响应触发词
**使用 Bedrock/Vertex 时不需要

### 常用 claude_args

```yaml
claude_args: "--max-turns 5 --model claude-sonnet-4-5-20250929"
```

- --max-turns：最大对话轮数（默认 10）
- --model：使用的模型
- --mcp-config：MCP 配置路径
- --allowed-tools：允许的工具列表
- --debug：启用调试输出
---

## CLAUDE.md 配置
在仓库根目录创建 `CLAUDE.md` 定义：

- 代码风格指南
- 审查标准
- 项目特定规则
- 偏好的模式
Claude 会在创建 PR 和响应请求时遵循这些指南。

---

## 使用云服务提供商

### AWS Bedrock

需要先配置：

- 在 Amazon Bedrock 中启用 Claude 模型访问
- 配置 GitHub 作为 AWS 的 OIDC 身份提供者
- 创建具有 Bedrock 权限的 IAM 角色
```yaml
name: Claude PR Action
permissions:
  contents: write
  pull-requests: write
  issues: write
  id-token: write

on:
  issue_comment:
    types: [created]

jobs:
  claude-pr:
    if: contains(github.event.comment.body, '@claude')
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Configure AWS Credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_ROLE_TO_ASSUME }}
          aws-region: us-west-2
      
      - uses: anthropics/claude-code-action@v1
        with:
          use_bedrock: "true"
          claude_args: '--model us.anthropic.claude-sonnet-4-5-20250929-v1:0'
```

### Google Vertex AI
需要先配置：

- 启用 Vertex AI API
- 配置 Workload Identity Federation
- 创建具有 Vertex AI 权限的服务账号
```yaml
jobs:
  claude-pr:
    if: contains(github.event.comment.body, '@claude')
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Authenticate to Google Cloud
        uses: google-github-actions/auth@v2
        with:
          workload_identity_provider: ${{ secrets.GCP_WORKLOAD_IDENTITY_PROVIDER }}
          service_account: ${{ secrets.GCP_SERVICE_ACCOUNT }}
      
      - uses: anthropics/claude-code-action@v1
        with:
          use_vertex: "true"
          claude_args: '--model claude-sonnet-4@20250514'
        env:
          ANTHROPIC_VERTEX_PROJECT_ID: ${{ steps.auth.outputs.project_id }}
          CLOUD_ML_REGION: us-east5
```

---

## 成本考量
**GitHub Actions 费用**：

- 消耗 GitHub Actions 分钟数
- 具体计费与配额以 GitHub 仓库的 Billing 页面为准
**API 费用**：

- 每次交互消耗 API token
- 复杂任务和大型代码库消耗更多
**优化建议**：

- 使用明确的 @claude 命令减少不必要的 API 调用
- 通过 --max-turns 限制迭代次数
- 设置工作流超时避免失控任务
- 使用并发控制限制并行运行
---

## 故障排除

### Claude 不响应 @claude 命令

- 验证 GitHub App 已正确安装
- 检查工作流是否启用
- 确认 API Key 已设置在仓库 secrets 中
- 确认评论包含 @claude（不是 /claude）
### Claude 的提交未触发 CI

- 确保使用 GitHub App（非 Actions 用户）
- 检查工作流触发器包含必要事件
- 验证 App 权限包含 CI 触发
### 认证错误

- 确认 API Key 有效且有足够权限
- 对于 Bedrock/Vertex，检查凭证配置
- 确保 secrets 名称正确
---

## 下一步
[Headless 模式无交互运行 Claude Code
](../headless/)[Hooks 系统自定义自动化逻辑
](../../advanced/hooks/)
