# 目录结构

## 一级目录

```
claude-hk/
├── .claude/                # 自动化机制核心
├── .github/                # GitHub 模板
├── claude-code-cli-doc/    # Claude Code 中文文档
├── doc/                    # 文档 + Skill 运行时输出
└── CLAUDE.md               # Claude Code 项目指令
```

## .claude/

自动化机制的核心，包含 Hooks 管道、Skill 系统和初始化脚本。

| 文件/目录 | 职责 |
|----------|------|
| `settings.json` | Hook 事件注册入口，29 个生命周期事件声明 |
| `settings.local.json` | 本地权限白名单 |
| `init.sh` | 首次运行统一初始化 |
| `dirs.conf` | 项目目录声明 |
| `.initialized` | 首次运行标记（JSON：os/python/utf8/时间） |

### hooks/

| 文件/目录 | 职责 |
|----------|------|
| `base.sh` | 公共基础：JSON 读取、日志、输出 |
| `platform.sh` | 平台检测（linux/macos/windows）+ Python 路径 |
| `json_get.py` | 无 jq 时的 JSON 解析回退 |
| `logs/` | 按日期的 hook 日志（YYYY-MM-DD.log） |
| `01-session-start/` | 会话启动：首次→init.sh，每次→环境巡检 |
| `03-user-prompt-submit/` | 用户输入：调度器 + skill-inject.sh |
| `05-pre-tool-use/` | 工具调用前：边界检查 + Skill 分发 |
| `16-stop/` | 响应完成：Skill 清理 + 自动注册 |
| `02,04,06~15,17~29/` | 其余事件（纯日志记录） |

### skills/

| 文件/目录 | 职责 |
|----------|------|
| `registry.conf` | Skill 注册表（skill-inject.sh 读取） |
| `active.sh` | 运行时状态管理（session ↔ skill 映射） |
| `lock.sh` | 并发文件锁（mkdir 原子操作） |
| `log.sh` | 双路日志模块 |
| `enforce_boundary.sh` | 工具边界检查（读取 SKILL.md 白名单） |
| `skill-register.sh` | 自动注册（扫描 SKILL.md 目录） |
| `001-testcode-python/` | Python 测试脚本 Skill |
| `002-otherdoc/` | 通用文档归档 Skill |
| `003-1~003-9/` | Issue 工作流 9 个 Skill |
| `004-git-push/` | 规范化 commit + push |
| `005-git-commit/` | 规范化 commit（仅本地） |

## .github/

GitHub 原生社区模板。

| 文件 | 职责 |
|------|------|
| `ISSUE_TEMPLATE/bug-report.md` | Bug 报告模板（Markdown 格式） |
| `ISSUE_TEMPLATE/bug_report.yml` | Bug 报告模板（YAML 表单） |
| `ISSUE_TEMPLATE/feature-request.md` | 功能请求模板 |
| `ISSUE_TEMPLATE/feature_request.yml` | 功能请求模板（YAML 表单） |
| `ISSUE_TEMPLATE/config.yml` | 模板配置（启用空白 issue） |
| `DISCUSSION_TEMPLATE/ideas.yml` | 功能讨论模板 |
| `PULL_REQUEST_TEMPLATE/default.md` | PR 模板（含 Test Plan） |

## claude-code-cli-doc/

Claude Code 中文文档库，从 `https://code.claude.com/docs/zh-CN/` 翻译。

| 子目录 | 内容 |
|--------|------|
| `01~09` | CLI 参考、命令、环境变量、工具、交互、Hooks、Plugins、Channels |
| `Agent技能体系/` | Skill 体系文档 |
| `Claude-API实战手册/` | Claude API 实战手册 |
| `Claude-Code使用指南/` | Claude Code 使用指南 |
| `Claude-Code源码分析/` | Claude Code 源码分析 |

## doc/

文档和 Skill 运行时输出。

| 子目录 | 类型 | 说明 |
|--------|------|------|
| `features/` | 文档 | 功能介绍文档 |
| `mechanisms/` | 文档 | 功能机制文档 |
| `project/` | 文档 | 项目治理文档 |
| `testcode/python/` | 运行时 | 001-testcode-python 输出 |
| `otherDoc/` | 运行时 | 002-otherdoc 输出（按日期） |
| `issues/` | 运行时 | 003-issues 输出（草稿、模板） |

## CLAUDE.md

Claude Code 项目指令文件，分为两部分：
- **Part 1**：项目特定指令（目录结构、架构、Skill 列表、提交规范）
- **Part 2**：行为准则（编码原则）

此文件由 Claude Code 自动加载，不属于用户文档体系。
