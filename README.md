# claude-hk

Claude Code 配置与文档项目。为 Claude Code CLI 构建了一套完整的 Hooks 管道、Skill 技能系统和自动化初始化流程。

## 快速开始

1. 克隆仓库后用 Claude Code 打开
2. 首次启动自动执行初始化（创建目录、配置 Python、设置 UTF-8）
3. 使用 `/技能名` 调用内置 Skill

## 项目结构

```
claude-hk/
├── .claude/                          # 自动化机制核心
│   ├── settings.json                 # 29 个生命周期事件声明
│   ├── init.sh                       # 首次运行统一初始化
│   ├── dirs.conf                     # 项目目录声明
│   ├── hooks/                        # Hooks 管道
│   │   ├── base.sh                   # 公共基础（JSON 解析、日志、输出）
│   │   ├── platform.sh               # 平台检测 + Python 路径解析
│   │   ├── json_get.py               # JSON 解析 Python 回退
│   │   ├── logs/                     # 按日期的 hook 日志
│   │   ├── 01-session-start/         # 会话启动：初始化 + 环境巡检
│   │   ├── 03-user-prompt-submit/    # 用户提交 prompt：Skill 注入
│   │   ├── 16-stop/                  # 响应完成：Skill 清理 + 自动注册
│   │   └── 02,04~29/                 # 其余事件（纯日志）
│   ├── scripts/
│   │   ├── ensure_dirs.sh            # 目录创建与检查
│   │   └── ensure_python.sh          # Python 环境保障（嵌入版/系统/自动下载）
│   ├── skills/                       # Skill 技能系统
│   │   ├── registry.conf             # 注册表
│   │   ├── active.sh                 # 运行时状态管理
│   │   ├── lock.sh                   # 并发文件锁
│   │   ├── log.sh                    # 双路日志模块
│   │   ├── 001-testcode-python/      # Python 测试脚本
│   │   ├── 002-otherdoc/             # 通用文档归档
│   │   ├── 003-issues/               # GitHub Issues 管理
│   │   ├── 004-git-push/             # 规范化 commit + push
│   │   └── 005-git-commit/           # 规范化 commit（仅本地）
│   └── myRule/                       # 用户自定义规则
├── .github/                          # GitHub 原生模板
│   ├── ISSUE_TEMPLATE/               # Issue 模板
│   └── PULL_REQUEST_TEMPLATE/        # PR 模板
├── claude-code-cli-doc/              # Claude Code 中文文档
├── doc/                              # Skills 运行时输出
│   ├── testcode/python/              # Python 脚本
│   ├── otherDoc/                     # 按日期归档的文档
│   └── issues/                       # Issue 草稿与模板
└── CLAUDE.md                         # 项目指令
```

## 架构

### 三层设计

```
settings.json（声明入口）
       │
       ▼
Hooks 管道（29 个事件调度）
       │
       ▼
Skills 技能系统（5 个可扩展技能）
```

### Hooks 管道

`settings.json` 注册 29 个生命周期事件，每个事件指向 `XX-event-name/base.sh`。

**公共基础设施** (`hooks/base.sh`)：
- `json_get(key)` — JSON 解析，三级降级：jq → Python → sed
- `log(level, msg)` — 结构化日志写入 `hooks/logs/YYYY-MM-DD.log`
- `hook_output(code, json)` — 标准退出（0=继续，2=阻止）

**有业务逻辑的事件**：

| 事件 | 触发时机 | 作用 |
|------|---------|------|
| 01-session-start | 每会话一次 | 首次初始化 + 每次环境巡检 |
| 03-user-prompt-submit | 每轮用户输入 | 检测 `/skill名`，匹配注册表，注入上下文 |
| 16-stop | 每轮响应完成 | 自动注册新 Skill + 清理活跃 Skill |

**扩展方式**：在事件目录下新建 `.sh` 脚本，在 `base.sh` 中添加调用即可，无需改 `settings.json`。

### 初始化流程

**首次运行** (`init.sh`)：

```
检测 .initialized 不存在
  → 1. 读取 dirs.conf 创建目录
  → 2. 平台检测 (Linux/macOS/Windows)
  → 3. Python 配置（嵌入版 → 系统 → 自动下载）
  → 4. UTF-8 配置（~/.bashrc 标记块，幂等）
  → 5. 写入 .initialized 标记
```

**每次会话** (`01-session-start/base.sh`)：

```
ensure_utf8 → ensure_python_check → ensure_dirs
```

### Skill 技能系统

**生命周期**：

```
用户输入 /skill-name
  → skill-inject.sh 匹配注册表
    → 运行 03UserPromptSubmit.sh（注入上下文到会话）
    → active_add(session_id, skill_name)
  → Claude 加载 SKILL.md 执行任务
  → Claude 完成 → 16Stop.sh 清理
    → active_remove(session_id)
```

**自动注册**：`skill-register.sh` 在每次 Stop 事件时扫描 `skills/` 目录，发现新 Skill 自动追加到 `registry.conf`。

**并发安全**：`lock.sh` 使用 `mkdir` 原子操作实现文件锁，保证 `.active` 文件的并发安全。

**双路日志**：每个 Skill 的日志同时写入统一日志（`hooks/logs/`）和模块日志（`skills/log/<tag>/`）。

## 内置 Skill

| Skill | 命令 | 用途 |
|-------|------|------|
| `001-testcode-python` | `/001-testcode-python` | Python 测试脚本和 API 自动化 |
| `002-otherdoc` | `/002-otherdoc` | 按日期归档通用文档 |
| `003-issues` | `/003-issues` | GitHub Issue 草稿、模板、发布管理 |
| `004-git-push` | `/004-git-push` | 规范化 commit + push |
| `005-git-commit` | `/005-git-commit` | 规范化 commit（仅本地） |

### Git 提交规范 (004/005)

格式：`<type>: <主描述>` + `- 子描述`

Type：fix / feat / update / style / refactor / perf / test / docs / revert / build / chore

分组优先级：type → 目录/模块 → 功能关联 → 影响范围。禁止 `git add .` 或 `git add -A`，必须按分组 add 具体文件。

## 新增 Skill

```
1. 创建 skills/XXX-name/ 目录
2. 编写 SKILL.md（frontmatter + 使用说明）
3. 编写 scripts/03UserPromptSubmit.sh（上下文注入）
4. 编写 scripts/16Stop.sh（清理）
5. 在 dirs.conf 中添加输出目录（如有）
6. 完成 — skill-register.sh 会自动注册
```

## 设计亮点

- **两层调度** — `base.sh` 只调度，业务脚本独立，扩展不改配置
- **自动注册** — 新 Skill 放入目录即被发现
- **渐进降级** — JSON: jq → Python → sed；Python: 嵌入版 → 系统 → 自动下载
- **幂等初始化** — 标记块防重复，状态文件持久化
- **并发安全** — mkdir 原子锁，僵尸锁自动清理
- **双路日志** — 全局 + 模块，便于排查

## 文档

- [Claude Code CLI 中文文档](claude-code-cli-doc/README.md)
- [Agent 技能体系](claude-code-cli-doc/Agent技能体系/)
- [Claude API 实战手册](claude-code-cli-doc/Claude-API实战手册/)
- [Claude Code 使用指南](claude-code-cli-doc/Claude-Code使用指南/)
- [Claude Code 源码分析](claude-code-cli-doc/Claude-Code源码分析/)

## Open Issues

- [#1](https://github.com/xiaoheiDTF/claude-hk/issues/1) — 16-stop .active 只增不删 bug
- [#2](https://github.com/xiaoheiDTF/claude-hk/issues/2) — 优化 003-issues Skill 闭环
- [#3](https://github.com/xiaoheiDTF/claude-hk/issues/3) — Skill 级 init.sh / init_check.sh
