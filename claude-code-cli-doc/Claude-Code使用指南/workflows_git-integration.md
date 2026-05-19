# Claude-Code / Workflows / Git-Integration

> 来源: claudecn.com

# Git 集成

Claude Code 深度集成 Git，让版本控制变得对话化。从智能提交到 Pull Request，Claude 可以处理完整的 Git 工作流。

## 智能提交

使用 `claude commit` 创建带有描述性信息的提交：

```bash
claude commit
```

Claude 会：

- 分析暂存的更改
- 理解修改的意图
- 生成符合规范的提交信息
- 执行提交
**在交互模式中：**

```
> 用描述性信息提交我的改动
```

```
> 提交这些更改，使用 conventional commits 格式
```

## 查看修改
让 Claude 帮你理解当前的 Git 状态：

```
> git status
```

```
> 我修改了哪些文件？
```

```
> 显示 auth.ts 的 diff
```

```
> 解释我最近的改动做了什么
```

Claude 不仅显示 diff，还能解释代码变更的含义。

## 创建 Pull Request

使用 GitHub CLI 集成创建 PR：

```
> 为当前分支创建 Pull Request
```

```
> 创建 PR，标题为"添加用户认证功能"，并生成详细描述
```

Claude 会：

- 分析分支的所有提交
- 生成 PR 标题和描述
- 列出主要更改
- 调用 gh pr create 创建 PR
## 处理代码审查评论

收到 PR 审查意见后：

```
> 查看 PR #123 的审查评论
```

```
> 修复 PR 审查中指出的问题
```

Claude 会获取评论内容，理解审查者的要求，并进行相应修改。

## GitHub CLI 集成

Claude Code 可以使用 `gh` 命令完成各种 GitHub 操作：

**Issues：**

```
> 列出所有打开的 issues
```

```
> 创建一个 bug 报告 issue
```

**Pull Requests：**

```
> 列出待审查的 PR
```

```
> 检出 PR #456 到本地
```

**Releases：**

```
> 创建新版本发布 v1.2.0
```

## 分支管理

```
> 创建并切换到 feature/user-profile 分支
```

```
> 将 main 分支合并到当前分支
```

```
> 显示分支图
```

## 把 Git 规范固化成 Agent / Command（团队用法）
如果你希望团队的分支命名、提交信息、PR 标题与模板更一致，可以把这些规范固化到：

- .claude/agents/：例如定义一个 github-workflow agent（分支命名、Conventional Commits、PR 模板）
- .claude/commands/：例如 pr-summary（基于 git diff/git log 生成 PR 描述模板）
这种做法的好处是：规则跟着仓库走，减少“口口相传”；同时也方便在 CI（GitHub Actions）里复用同一套标准。

## 解决合并冲突

```
> 帮我解决合并冲突
```

Claude 会：

- 识别冲突文件
- 分析两边的更改意图
- 提出解决方案
- 应用修复
## 自动化工作流示例

### 功能开发完整流程

```
> 1. 创建 feature/payment 分支
> 2. [实现功能]
> 3. 提交所有更改
> 4. 推送到远程
> 5. 创建 Pull Request
```

### 修复紧急 Bug

```bash
# 一行命令完成修复和提交
claude "修复登录按钮不响应的问题，然后提交"
```

### 代码审查工作流

```
> 检出 PR #789
> 审查代码并提出改进建议
> 留下审查评论
```

## Git 别名和快捷方式
常用操作可以通过简短命令完成：

| 命令 | 作用 |
| --- | --- |
| `claude commit` | 智能提交 |
| `claude "status"` | 查看状态 |
| `claude "push"` | 推送更改 |
| `claude "log"` | 查看历史 |

## 最佳实践

- 小步提交：让 Claude 为每个逻辑更改创建单独提交
- 有意义的信息：使用 claude commit 生成描述性提交信息
- 分支策略：让 Claude 帮助管理分支命名和合并策略
- PR 描述：让 Claude 生成详细的 PR 描述，方便审查者理解
- 冲突处理：遇到复杂冲突时让 Claude 分析两边意图
## GitHub CLI 安装

Claude Code 的 Git 集成依赖 GitHub CLI (`gh`)。

### macOS

```bash
brew install gh
```

### Linux (Debian/Ubuntu)

```bash
sudo apt update
sudo apt install gh
```

### Windows

```powershell
winget install GitHub.cli
```

### 认证

```bash
gh auth login
```

## CI/CD 集成

### GitHub Actions 自动化
在 CI/CD 流程中使用 Claude Code 进行自动化任务。

#### 代码审查 Action

```yaml
name: Claude Code Review

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

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Install Claude Code
        run: npm install -g @anthropic/claude-code

      - name: Review PR Changes
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: |
          git diff origin/main...HEAD > changes.diff
          cat changes.diff | claude -p "审查这些代码更改，指出潜在问题" \
            --output-format json > review.json

      - name: Post Review Comment
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const review = JSON.parse(fs.readFileSync('review.json', 'utf8'));
            await github.rest.issues.createComment({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: context.issue.number,
              body: review.result
            });
```

#### 自动修复 Lint 错误

```yaml
name: Auto Fix Lint

on:
  push:
    branches: [main]

jobs:
  fix:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Install Dependencies
        run: |
          npm ci
          npm install -g @anthropic/claude-code

      - name: Run Lint and Fix
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: |
          npm run lint 2>&1 | tee lint-output.txt || true
          if grep -q "error" lint-output.txt; then
            cat lint-output.txt | claude -p "修复这些 lint 错误" \
              --dangerously-skip-permissions
          fi

      - name: Commit Fixes
        run: |
          git config user.name "Claude Bot"
          git config user.email "claude@example.com"
          git add -A
          git diff --staged --quiet || git commit -m "fix: auto-fix lint errors"
          git push
```

#### 生成发布说明

```yaml
name: Generate Release Notes

on:
  release:
    types: [created]

jobs:
  notes:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install Claude Code
        run: npm install -g @anthropic/claude-code

      - name: Generate Notes
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: |
          git log --oneline $(git describe --tags --abbrev=0 HEAD^)..HEAD > commits.txt
          cat commits.txt | claude -p "根据这些提交生成用户友好的发布说明" > notes.md

      - name: Update Release
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const notes = fs.readFileSync('notes.md', 'utf8');
            await github.rest.repos.updateRelease({
              owner: context.repo.owner,
              repo: context.repo.repo,
              release_id: context.payload.release.id,
              body: notes
            });
```

### GitLab CI 集成

```yaml
stages:
  - review
  - fix

code-review:
  stage: review
  image: node:20
  script:
    - npm install -g @anthropic/claude-code
    - git diff origin/main...HEAD | claude -p "审查代码更改" --output-format json > review.json
  artifacts:
    paths:
      - review.json
  only:
    - merge_requests

auto-fix:
  stage: fix
  image: node:20
  script:
    - npm install -g @anthropic/claude-code
    - npm run test 2>&1 | claude -p "修复测试失败" --dangerously-skip-permissions
    - git add -A && git commit -m "fix: auto-fix" || true
    - git push origin HEAD:$CI_COMMIT_REF_NAME
  only:
    - branches
  when: manual
```
