# claude-hk

Claude Code 配置与文档项目。为 Claude Code CLI 构建了一套完整的 Hooks 管道、Skill 技能系统和自动化初始化流程。

## 功能概览

- **Hooks 管道** — 29 个生命周期事件自动调度，两层架构可扩展
- **Skill 系统** — 13 个内置技能，覆盖 Issue 管理、Git 工作流、测试脚本等
- **自动初始化** — 一键配置 Python、UTF-8、目录结构，支持 Windows/macOS/Linux
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
/003-2-issue #              # 创建 GitHub Issue
/004-git-push                # 规范化提交并推送
```

## 安装要求

| 依赖 | 必需 | 说明 |
|------|------|------|
| Claude Code CLI | 是 | [安装指南](https://docs.anthropic.com/en/docs/claude-code) |
| Git | 是 | 版本控制 |
| Bash | 是 | Windows 自带（Git Bash） |
| GitHub CLI (`gh`) | 部分 | Issue/PR 相关 Skill 需要 |
| Python 3.x | 可选 | 无环境时自动下载嵌入版 |

## 文档导航

| 层面 | 目录 | 回答的问题 |
|------|------|-----------|
| 功能介绍 | [`doc/features/`](doc/features/) | 有哪些功能？怎么用？ |
| 功能机制 | [`doc/mechanisms/`](doc/mechanisms/) | 怎么实现的？设计原理是什么？ |
| 项目治理 | [`doc/project/`](doc/project/) | CHANGELOG、贡献指南 |
| 中文翻译 | [`claude-code-cli-doc/`](claude-code-cli-doc/) | Claude Code CLI 完整中文文档 |

## 项目结构

```
claude-hk/
├── .claude/                # 自动化机制核心
│   ├── hooks/              # 29 个生命周期事件
│   ├── skills/             # 13 个内置 Skill
│   └── settings.json       # Hooks 注册入口
├── .github/                # Issue/PR/Discussion 模板
├── claude-code-cli-doc/    # Claude Code 中文文档（130+ 篇）
├── doc/                    # 文档 + Skill 运行时输出
└── CLAUDE.md               # Claude Code 项目指令
```

详细结构解析见 [目录结构文档](doc/mechanisms/project-structure.md)。

## 许可证

MIT License
