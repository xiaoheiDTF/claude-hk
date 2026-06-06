# claude-hk

Claude Code 配置与文档项目。为 Claude Code CLI 构建了一套完整的 Hooks 管道、Skill 技能系统和自动化初始化流程，搭配 Go 语言实现的反向代理与后端服务。

## 功能概览

- **Hooks 管道** — 29 个生命周期事件自动调度，两层架构可扩展
- **Skill 系统** — 24 个内置技能，覆盖 Issue 管理、开发流程（BDD/TDD/CDD）、Git 工作流等
- **Go 后端服务** — 反向代理（API 流量拦截 + JSONL 追踪 + Token 统计）+ HTTP 后端（Issue/Session 管理，15 个端点）
- **自动初始化** — 一键配置 Python、UTF-8、目录结构，支持 Windows/macOS/Linux
- **开发流程** — 003 系列技能实现从功能树分析到 E2E 测试的完整 BDD/TDD 流水线
- **Git 工作流** — 规范化 commit + push，中文提交信息，按类型分组
- **中文文档** — Claude Code CLI 完整中文翻译文档（130+ 篇）

更多功能详见 [功能文档](doc/features/README.md)。

## 快速开始

```bash
# 1. 克隆仓库
git clone https://github.com/xiaoheiDTF/claude-hk.git
cd claude-hk

# 2. 用 Claude Code 打开（自动触发初始化）
claude

# 3. 使用内置 Skill
/001-2-issue              # 创建 GitHub Issue
/003-1-develop-feature-tree  # 功能点与功能树分析
/999-2-git-push            # 规范化提交并推送
```

### Go 后端服务（可选）

```bash
cd claude_tap_plus
go build -o claude-tap-plus ./cmd/claude-tap

# 代理模式（拦截 API 流量，记录 JSONL 追踪）
claude-tap-plus --tap-profile glm

# 后端服务（Issue 管理 + 会话管理）
go run ./cmd/claude-tap backend
```

## 安装要求

| 依赖 | 必需 | 说明 |
|------|------|------|
| Claude Code CLI | 是 | [安装指南](https://docs.anthropic.com/en/docs/claude-code) |
| Git | 是 | 版本控制 |
| Bash | 是 | Windows 自带（Git Bash） |
| GitHub CLI (`gh`) | 部分 | Issue/PR 相关 Skill 需要 |
| Python 3.x | 可选 | 无环境时自动下载嵌入版 |
| Go 1.21+ | 可选 | 仅编译 claude_tap_plus 时需要 |

## 文档导航

| 层面 | 目录 | 回答的问题 |
|------|------|-----------|
| 架构设计 | [`doc/redeme/`](doc/redeme/) | 系统架构、流程图、数据模型 |
| 功能介绍 | [`doc/features/`](doc/features/) | 有哪些功能？怎么用？ |
| 功能机制 | [`doc/mechanisms/`](doc/mechanisms/) | 怎么实现的？设计原理是什么？ |
| 项目治理 | [`doc/project/`](doc/project/) | CHANGELOG、贡献指南 |
| 中文翻译 | [`doc/claude-code-cli-doc/`](doc/claude-code-cli-doc/) | Claude Code CLI 完整中文文档 |

## 项目结构

```
claude-hk/
├── .claude/                # 自动化机制核心
│   ├── hooks/              # 29 个生命周期事件
│   ├── skills/             # 24 个内置 Skill
│   └── settings.json       # Hooks 注册入口
├── .github/                # Issue/PR/Discussion 模板
├── claude_tap_plus/        # Go 后端服务
│   ├── cmd/                # CLI 入口（代理/后端/会话管理）
│   └── internal/           # 代理管线 + 后端 API + 持久化
├── doc/                    # 文档 + Skill 运行时输出
│   ├── redeme/             # 架构设计文档与流程图
│   ├── claude-code-cli-doc/  # Claude Code 中文文档（130+ 篇）
│   ├── otherDoc/           # 归档文档（按日期）
│   └── testcode/           # Python 测试脚本
└── CLAUDE.md               # Claude Code 项目指令
```

详细结构解析见 [目录结构文档](doc/mechanisms/project-structure.md)。

## Skill 概览

### 001 Issue 工作流（9 个）

| Skill | 命令 | 说明 |
|-------|------|------|
| 001-1-issue-init | `/001-1-issue-init` | 初始化 issue 标签体系（一次性） |
| 001-2-issue | `/001-2-issue` | 创建 GitHub Issue |
| 001-3-issue-discuss | `/001-3-issue-discuss` | 讨论 Issue 内容 |
| 001-4-issue-claim | `/001-4-issue-claim` | 原子领取 Issue |
| 001-5-issue-fix | `/001-5-issue-fix` | 创建分支并开始开发 |
| 001-6-issue-done | `/001-6-issue-done` | 标记开发完成 |
| 001-7-issue-pr | `/001-7-issue-pr` | 创建 PR |
| 001-8-issue-test | `/001-8-issue-test` | 执行 Test Plan |
| 001-9-issue-review | `/001-9-issue-review` | 审核并合并/打回 |

### 002 文档工具（2 个）

| Skill | 命令 | 说明 |
|-------|------|------|
| 002-1-doc-otherdoc | `/002-1-doc-otherdoc` | 文档归档（按日期） |
| 002-2-doc-testcode-python | `/002-2-doc-testcode-python` | Python 测试脚本 |

### 003 开发流程（8 个）

| Skill | 命令 | 说明 |
|-------|------|------|
| 003-1-develop-feature-tree | `/003-1-develop-feature-tree` | 功能点与功能树分析 |
| 003-2-develop-bdd-scenario | `/003-2-develop-bdd-scenario` | BDD 场景规范 |
| 003-3-1-backend-bdd | `/003-3-1-backend-bdd` | 后端 BDD |
| 003-3-2-frontend-bdd | `/003-3-2-frontend-bdd` | 前端 BDD |
| 003-4-api-contract | `/003-4-api-contract` | API 契约 |
| 003-5-1-backend-tdd-java | `/003-5-1-backend-tdd-java` | 后端 TDD（Java/SpringBoot） |
| 003-6-1-ui-state-definition | `/003-6-1-ui-state-definition` | UI 状态定义 |
| 003-6-2-frontend-cdd | `/003-6-2-frontend-cdd` | 前端 CDD |
| 003-7-e2e-test | `/003-7-e2e-test` | 端到端测试 |

### 999 Git 与工具（4 个）

| Skill | 命令 | 说明 |
|-------|------|------|
| 999-1-git-commit | `/999-1-git-commit` | 规范化提交（仅 commit） |
| 999-2-git-push | `/999-2-git-push` | 规范化提交并推送 |
| 999-other-110-requirement-planning | `/999-other-110-requirement-planning` | 需求规划（PRD 文档） |
| 999-other-120-learn | `/999-other-120-learn` | 学习进化 |

## 许可证

MIT License
