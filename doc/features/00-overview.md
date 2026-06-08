# claude-hk 项目功能概览

> 最后更新：2026-06-06

---

## 项目定位

claude-hk 是 Claude Code CLI 的增强扩展层，由两个耦合的子系统组成：

| 子系统 | 位置 | 语言 | 角色 |
|--------|------|------|------|
| **Shell 自动化层** | `.claude/` | Bash + Python | 29 个生命周期 Hook + 24 个 Skill，注入工作流自动化 |
| **Go 服务层** | `claude_tap_plus/` | Go | 反向代理（记录 Trace）+ 后端 API（SQLite 存储） |

两者仅通过 HTTP（`127.0.0.1`）通信，配置桥接文件为 `~/.claude-tap-plus/backend.json`。

---

## 功能模块索引

| 模块 | 文档 | 核心能力 |
|------|------|---------|
| Hooks 管线 | [01-hooks-pipeline.md](01-hooks-pipeline.md) | 29 个生命周期事件拦截、Skill 分发、工具边界管控 |
| Skill 系统 | [02-skill-system.md](02-skill-system.md) | 24 个技能（Issue 工作流 / 文档管理 / 开发流程 / Git 操作） |
| 代理服务 | [03-proxy-service.md](03-proxy-service.md) | 反向代理、API 流量录制（JSONL Trace）、SSE 流处理、Model 改写 |
| 后端服务 | [04-backend-service.md](04-backend-service.md) | REST API（24 端点）、SQLite 存储、Session/Issue/Proxy 管理 |
| 会话同步 | [05-session-sync.md](05-session-sync.md) | 会话文件 push/pull/status、跨设备同步 |

---

## 快速启动

```bash
# 1. 启动代理（自动拉起后端）
cd claude_tap_plus
go run ./cmd/claude-tap claude

# 2. 在 Claude Code 中使用 Skill
/001-2-issue          # 创建 Issue
/001-4-issue-claim    # 领取 Issue
/001-5-issue-fix      # 创建分支开发
/999-2-git-push       # 提交并推送
```

---

## 数据流总览

```
Claude Code CLI
    │
    │  ① Hook 事件（stdin JSON）
    ▼
.claude/hooks/XX-event/base.sh ──→ skills/<skill>/scripts/<Event>.sh
    │                                         │
    │  ② HTTP 调用                             │  ③ gh CLI / git 操作
    ▼                                         ▼
claude_tap_plus/backend (SQLite)       GitHub / Git
    ▲
    │  ④ API 流量拦截
    │
claude_tap_plus/proxy ──→ 上游 API (Anthropic/OpenAI/Gemini)
```
