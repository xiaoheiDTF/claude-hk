# 会话同步功能与配置

> 最后更新：2026-06-06

---

## 功能概述

会话同步功能提供 Claude Code 会话文件在不同设备/环境之间的收集与恢复能力，通过 `session-push`、`session-pull`、`session-status` 三个子命令实现。

---

## 子命令

### session-push — 收集会话

从 `~/.claude/projects/{slug}/` 收集 `{uuid}.jsonl` 会话文件到本地存储目录。

```bash
go run ./cmd/claude-tap session-push [选项]

选项:
  --all       收集所有项目（默认只收集当前项目）
  --force     强制覆盖已存在的文件
  --dry-run   预览模式，不实际复制
```

**功能特性**：
- 增量收集：跳过已存在的文件（除非 `--force`）
- 同时收集 `subagents` 子目录
- 维护 `meta.json` 元数据文件
- 自动生成项目 slug（路径中 `:`, `/`, `\` 替换为 `-`）

**存储路径**：

```
{exe所在目录}/sessions/{slug}/
├── meta.json                  # 元数据
├── {uuid}.jsonl               # 会话文件
└── subagents/
    └── {uuid}.jsonl           # 子 Agent 会话
```

### session-pull — 恢复会话

从本地存储恢复会话文件到 `~/.claude/projects/{slug}/`。

```bash
go run ./cmd/claude-tap session-pull [选项]

选项:
  --all       恢复所有项目
  --dry-run   预览模式
```

**功能特性**：
- 恢复后自动更新 `~/.claude.json` 中的 `lastSessionId`、`lastSessionModified` 等字段
- 支持全项目恢复

### session-status — 查看同步状态

显示每个项目的本地存储 vs Claude 目录的同步状态。

```bash
go run ./cmd/claude-tap session-status
```

**输出示例**：

```
Project: claude-hk
  Claude dir:  ~/.claude/projects/D-CodeDevelopment-CodeProject-claude-hk/
  Local dir:   ./sessions/D-CodeDevelopment-CodeProject-claude-hk/
  Claude:      15 sessions, 3 subagents
  Local:       12 sessions, 2 subagents
  Status:      3 sessions not synced
```

---

## 数据结构

### SessionMeta（`meta.json`）

| 字段 | 类型 | 说明 |
|------|------|------|
| Project | string | 项目名 |
| GitRemote | string | Git remote origin URL |
| Slug | string | 项目标识 |
| Sessions | []SessionEntry | 会话条目列表 |
| LastUpdated | time.Time | 最后更新时间 |

### SessionEntry

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | string | 会话 UUID |
| StartTime | time.Time | 最早记录时间 |
| EndTime | time.Time | 最新记录时间 |
| Models | []string | 使用过的模型列表 |
| FileSize | int64 | 文件大小 |
| FileName | string | 文件名 |

### Slug 生成规则

将绝对路径的 `:`, `/`, `\` 替换为 `-`：

```
D:\CodeDevelopment\CodeProject\claude-hk → D--CodeDevelopment-CodeProject-claude-hk
```

---

## 配置说明

| 配置项 | 来源 | 说明 |
|--------|------|------|
| 源目录 | `~/.claude/projects/` | Claude Code 会话文件存放位置 |
| 目标目录 | `{exe所在目录}/sessions/` | 本地存储位置 |
| slug | 自动生成 | 路径转换后的项目标识 |
| `~/.claude.json` | 自动更新 | 恢复后更新 `lastSessionId` 等字段 |

> 无需额外配置文件，所有路径自动推断。
