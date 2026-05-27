# SR-1: Trace 存储路径重构

> 创建时间：2026-05-27
> 模块：claude-tap-plus / 代理侧
> 简述：重构 trace writer 的文件输出路径，加入 machine_id 和 project_slug 层级

---

## 目标

将 trace 文件路径从当前结构改为四层目录结构，支持多机器、多项目的 trace 文件隔离。

## 当前路径（待确认）

代理现有的 trace 输出路径（需从 `trace/writer.go` 确认）。

## 目标路径

```
.claude-tap-plus/.traces/{machine_id}/{project_slug}/{session_id}.jsonl
```

示例：
```
.claude-tap-plus/.traces/Administrator@DESKTOP-ABC123/D--CodeDevelopment-CodeProject-claude-hk/bf15cac4-7235-48ce-8853-5d4598547f31.jsonl
```

### 四层目录说明

| 层级 | 路径 | 说明 |
|------|------|------|
| 1 | `.claude-tap-plus/` | 工具根目录 |
| 2 | `.traces/` | trace 存储目录 |
| 3 | `{machine_id}/` | 机器标识（`whoami@hostname`） |
| 4 | `{project_slug}/` | 项目标识（从 transcript_path 提取） |

### project_id 组合标识符

trace 路径中的第三层和第四层组合构成 `project_id`：

```
project_id = "{machine_id}/{project_slug}"
```

示例：`Administrator@DESKTOP-ABC123/D--CodeDevelopment-CodeProject-claude-hk`

此组合标识符仅用于 trace 路径构建，后端数据模型中 `machine_id` 和 `project_slug` 保持为独立字段。

## 改造范围

### 文件：`claude_tap_plus/internal/trace/writer.go`

**改造点：**

1. **`NewTracePath()`** — 加入 machine_id 层级目录
   - 获取 `whoami@hostname` 组合作为 machine_id
   - 从代理拦截的请求中提取 project_slug（复用 transcript_path 的解析逻辑）
   - 构建四层目录路径并自动创建不存在的目录

2. **`extractSessionID()`** — 从拦截的 API 请求中提取 session_id
   - 首次拦截到某个 session_id 时创建对应 trace 文件
   - 后续同一 session 的请求追加写入同一文件

### 从 transcript_path 解析标识

```
C:\Users\Administrator\.claude\projects\D--CodeDevelopment-CodeProject-claude-hk\bf15cac4-...-5d4598547f31.jsonl
│                                          │                                              │
│                                          │                                              └─ session_id
│                                          └─ project_slug（Claude Code 内置的项目标识）
│                                             由 cwd 的路径分隔符替换为 - 得到
└─ 用户主目录 → 机器/用户标识
```

解析逻辑（Go 实现）：

```go
// 从 transcript_path 提取 project_slug
// 输入: "C:\\Users\\Admin\\.claude\\projects\\D--xxx-yyy\\uuid.jsonl"
// 输出: "D--xxx-yyy"
func extractProjectSlug(transcriptPath string) string {
    // 按 .claude/projects/ 分割，取第二段再取第一个路径组件
}
```

## 验证标准

- [ ] trace 文件路径包含 machine_id 和 project_slug 层级
- [ ] 目录不存在时自动创建
- [ ] 同一 session 的多次 API 调用追加写入同一 trace 文件
- [ ] 跨平台路径分隔符处理正确（Windows `/` vs `\`）
- [ ] 现有 trace 写入内容格式不变（只改路径不改内容）

## 依赖

无外部依赖，代理侧独立改造。
