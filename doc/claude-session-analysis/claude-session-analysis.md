# Claude Code Session 存储机制分析

> 讨论日期：2026-05-20

## 背景

分析 Claude Code 的 session 存储机制，评估跨设备同步的可行性，为 claude-tap-plus 的云同步功能提供设计依据。

---

## 一、Claude Code Session 存储全景

### 1.1 目录结构

```
~/.claude/
├── sessions/{pid}.json                    ← 活跃进程映射（临时）
├── projects/{path-slug}/                  ← 核心：按项目路径隔离
│   ├── {sessionId}.jsonl                  ← 完整对话记录（流式粒度）
│   └── {sessionId}/subagents/             ← 子 agent 对话
├── tasks/{sessionId}/                     ← 任务列表
├── file-history/{sessionId}/              ← 文件编辑备份
├── session-env/{sessionId}/               ← 会话环境
├── history.jsonl                          ← 全局提示历史
├── plans/{name}.md                        ← 计划文件
├── shell-snapshots/                       ← Shell 环境快照
├── transcripts/ses_{id}.jsonl             ← 子系统 transcript
├── backups/.claude.json.backup.{ts}       ← 配置备份
├── ide/{pid}.lock                         ← IDE 连接锁
├── paste-cache/{hash}.txt                 ← 粘贴缓存
├── .claude.json                           ← 用户配置 + 项目元数据
├── settings.json                          ← 全局设置
└── settings.local.json                    ← 本地权限
```

### 1.2 路径映射规则

Claude Code 将项目的绝对路径转换为 slug 作为目录名：

**算法**：将 `:`、`/`、`\` 全部替换为 `-`

| 实际路径 | 映射目录 |
|---------|---------|
| `D:\development\code\claude-tap` | `D--development-code-claude-tap` |
| `D:\development\code\saas-rd\tripmax\database` | `D--development-code-saas-rd-tripmax-database` |
| `C:\Users\EDY` | `C--Users-EDY` |

**可逆性**：理论可逆，但路径中自然包含 `-` 的情况有歧义。

**子目录是独立项目**：不同子目录会生成不同的 slug。

### 1.3 Session ID

- **格式**：UUID v4（随机生成）
- **生成位置**：客户端本地生成
- **唯一性**：UUID v4 有 2^122 种可能，碰撞概率极低，跨机器也是唯一的
- **传播**：嵌入在 JSONL 每条记录的 `sessionId` 字段，同时嵌入在 API 请求的 `metadata.user_id` JSON 内

当前活跃的 session 示例：

| PID | Session ID | 项目 | 状态 | 入口 |
|-----|-----------|------|------|------|
| 12780 | `731bcd6d-b7eb-401f-a9f9-4ed73ce85b38` | claude-tap | busy | cli |
| 32464 | `8a7f5db9-3caf-4151-9a17-10e97c33c007` | claude-hk | idle | cli |
| 24316 | `e437782f-6487-46f7-b942-1ef819d229a5` | tripmax | idle | claude-vscode |

### 1.4 活跃进程文件（sessions/{pid}.json）

```json
{
  "pid": 12780,
  "sessionId": "731bcd6d-b7eb-401f-a9f9-4ed73ce85b38",
  "cwd": "D:\\development\\code\\claude-tap",
  "startedAt": 1779240046530,
  "version": "2.1.143",
  "peerProtocol": 1,
  "kind": "interactive",
  "entrypoint": "cli",
  "status": "busy",
  "updatedAt": 1779258755068
}
```

- 进程退出后文件清除
- `entrypoint`：`cli`（终端）或 `claude-vscode`（IDE 扩展）
- `status`：`busy` 或 `idle`

### 1.5 项目元数据（.claude.json）

每个项目在 `.claude.json` 中有以下 session 相关字段：

```json
{
  "lastSessionId": "4a430bc7-cdd0-4010-adf2-70a4ddaacd36",
  "lastHintSessionId": "731bcd6d-b7eb-401f-a9f9-4ed73ce85b38",
  "lastGracefulShutdown": true,
  "lastSessionFirstPrompt": "用户提示文本",
  "lastSessionModified": 1779240129725,
  "lastCost": 11.028336,
  "lastDuration": 140715999,
  "lastTotalInputTokens": 913667,
  "lastTotalOutputTokens": 37577,
  "lastModelUsage": {"GLM-5.1": {...}}
}
```

**关键**：`lastSessionId` 和 `lastHintSessionId` 可以不同。前者是上一次完成的 session，后者是当前活跃的。

---

## 二、Session JSONL 记录详解

### 2.1 记录类型清单

| 类型 | 作用 | 有 UUID 链 | 频率 |
|------|------|-----------|------|
| `permission-mode` | 标记权限模式 | 否 | 每轮开始 |
| `file-history-snapshot` | 文件编辑快照 | 否 | 每轮 |
| `user` | 用户消息 | 是 | 每轮 |
| `attachment` | 附件（7 种子类型） | 是 | 多次/轮 |
| `assistant` | AI 回复（thinking/text/tool_use） | 是 | 多次/轮 |
| `system/stop_hook_summary` | Hook 执行摘要 | 是 | 每轮结束 |
| `system/turn_duration` | 耗时统计 | 是 | 每轮结束 |
| `ai-title` | AI 生成的会话标题 | 否 | 偶尔 |
| `last-prompt` | 最后一条用户提示 | 否 | 偶尔 |
| `queue-operation` | 提示队列操作 | 否 | 偶尔 |

### 2.2 UUID 链结构

记录通过 `parentUuid → uuid` 形成单向链表：

```
user (parentUuid=null, uuid=A)
  → attachment:skill_listing (parentUuid=A, uuid=B)
    → assistant:thinking (parentUuid=B, uuid=C)
      → assistant:tool_use (parentUuid=C, uuid=D)
        → user:tool_result (parentUuid=D, uuid=E)
          → assistant:text (parentUuid=E, uuid=F)
            → system/stop_hook_summary (parentUuid=F, uuid=G)
              → system/turn_duration (parentUuid=G, uuid=H)
```

`permission-mode`、`file-history-snapshot`、`ai-title`、`last-prompt` 不参与链表。

### 2.3 关键记录结构

**user（普通消息）**：
```json
{
  "parentUuid": null,
  "isSidechain": false,
  "promptId": "6fabf3db-...",
  "type": "user",
  "message": {"role": "user", "content": "用户提示"},
  "uuid": "140347fe-...",
  "timestamp": "2026-05-20T01:20:56.622Z",
  "permissionMode": "default",
  "userType": "external",
  "entrypoint": "cli",
  "cwd": "D:\\development\\code\\claude-tap",
  "sessionId": "731bcd6d-...",
  "version": "2.1.143",
  "gitBranch": "claude-tap-plus"
}
```

**user（工具结果）**：额外有 `toolUseResult`、`sourceToolAssistantUUID`

**assistant（流式拆分）**：一个 API 消息（同一个 `message.id`）产生多条 JSONL 记录：
- thinking 一条
- text 一条
- 每个 tool_use 各一条

**assistant message 结构**：
```json
{
  "id": "msg_20260520092057502198d6104548b8",
  "type": "message",
  "role": "assistant",
  "model": "glm-5.1",
  "content": [...],
  "stop_reason": "tool_use",
  "usage": {
    "input_tokens": 55199,
    "cache_read_input_tokens": 256,
    "output_tokens": 54
  }
}
```

### 2.4 attachment 子类型（7 种）

| 子类型 | 作用 | 关键字段 |
|--------|------|---------|
| `skill_listing` | 首次提示时发送 skill 列表 | `isInitial`, `skillCount` |
| `async_hook_response` | Hook 输出 | `hookName`, `stdout`, `stderr`, `exitCode` |
| `task_reminder` | 任务列表提醒 | `itemCount` |
| `queued_command` | 提示队列 | `prompt`, `commandMode` |
| `nested_memory` | 文件级 memory/rules 加载 | `path`, `content` |
| `plan_mode` | Plan 模式激活 | `planFilePath`, `planExists` |
| `plan_mode_exit` | Plan 模式退出 | `planFilePath`, `planExists` |

### 2.5 流式粒度（关键发现）

JSONL 文件保存的是**流式级别**的细节，不是简单的对话摘要。一个 API 调用可能产生 5+ 条 JSONL 记录。

文件大小参考：
- `731bcd6d-b7eb-...jsonl`：649 行，940KB（约 30 轮对话）
- `4a430bc7-cdd0-...jsonl`：547 行，892KB

---

## 三、Resume 机制

### 3.1 工作流程

```
claude --resume / claude -c
  ↓
1. 计算当前 cwd 的 path-slug
  ↓
2. 读取 .claude.json 中该项目的 lastSessionId
  ↓
3. 定位 ~/.claude/projects/{slug}/{sessionId}.jsonl
  ↓
4. 重放 JSONL 重建完整对话状态
  ↓
5. 在链表末尾（last-prompt 的 leafUuid）继续
```

### 3.2 相关 CLI 参数

| 参数 | 作用 |
|------|------|
| `-c` / `--continue` | 继续当前目录最近的 session |
| `-r` / `--resume` | 恢复指定 session（交互选择器或传入 ID） |
| `--fork-session` | resume 时创建新 session 而非覆盖原 session |
| `--session-id` | 指定一个 UUID 作为 session ID |
| `--from-pr` | 从 PR 上下文恢复 |
| `--no-session-persistence` | 不持久化 session（仅 print 模式） |

### 3.3 关键环境变量

| 变量 | 作用 |
|------|------|
| `CLAUDE_CONFIG_DIR` | 覆盖 `~/.claude` 路径（可指向网盘） |
| `CLAUDE_CODE_RESUME_INTERRUPTED_TURN` | 自动恢复中断的轮次 |
| `CLAUDE_CODE_REMOTE` | 云端 session 标记 |
| `CLAUDE_CODE_REMOTE_SESSION_ID` | 云端 session ID |

---

## 四、跨设备同步方案对比

### 4.1 核心障碍：路径映射

```
机器 A: D:\development\code\claude-tap  → D--development-code-claude-tap
机器 B: /home/user/projects/claude-tap → -home-user-projects-claude-tap
```

Claude Code 按 cwd 计算 slug，不同机器路径不同 → `claude --resume` 找不到。

### 4.2 已有方案

| 方案 | 做法 | 跨机器 resume | 状态 |
|------|------|--------------|------|
| Claude Sync（开源） | `age` 加密同步 `~/.claude/` 到 S3/R2/GCS | 路径不同则不行 | 作者承认待修复 |
| CLAUDE_CONFIG_DIR 指向网盘 | 所有机器共享同一目录 | JSONL 并发写入冲突 | 可行但不稳定 |
| Hook 自动保存 session_id | SessionEnd Hook 写 `.claude_session` | 仅同机器 | 不适用 |

**Claude Sync 的致命问题（评论区指出）**：

> 即使同步了 `~/.claude/projects/`，机器 A 路径是 `/Users/me/Downloads/app`，机器 B 是 `/Users/me/Documents/app`，`claude --resume` 在机器 B 上找不到机器 A 的 session，因为 slug 不同。

### 4.3 Claude Code 官方态度

- GitHub Issue [#42219](https://github.com/anthropics/claude-code/issues/42219)：跨设备 session 同步需求 → 已关闭（duplicate）
- GitHub Issue [#31992](https://github.com/anthropics/claude-code/issues/31992)：CLI-to-CLI session 移交 → Open，Anthropic 未实现
- **结论：Claude Code 目前不支持跨设备 session resume**

---

## 五、对 claude-tap 的建议

### 5.1 trace 自身同步（已完成基础设施）

```
.traces/
└── claude-tap/                                    ← 项目名目录
    ├── 2026-05-20_100000_a3f2c1.jsonl             ← trace 文件
    └── 2026-05-20_143000_b7e9d4.jsonl             ← resume 产生的新 trace
```

- 文件名含项目名 + 时间 + 随机 ID，天然隔离不冲突
- `session_id` 在 record 顶层，跨文件可关联
- 同步 `.traces/{project}/` 目录即可

### 5.2 Claude session 同步（可选增强）

如果需要让 Claude Code 跨机器 resume：

1. 上传时记录 slug 映射表
2. 下载时按目标机器路径重算 slug
3. 将 JSONL 放到正确的 `~/.claude/projects/{new-slug}/` 下
4. 更新 `.claude.json` 的 `lastSessionId`

### 5.3 需要同步的 Claude 文件清单

| 文件 | 必要性 | 体积 | 说明 |
|------|--------|------|------|
| `projects/{slug}/{id}.jsonl` | 必须 | ~1MB/30轮 | 核心对话记录 |
| `projects/{slug}/{id}/subagents/` | 推荐 | 较小 | 子 agent 数据 |
| `.claude.json` 项目元数据 | 必须 | 极小 | lastSessionId 等 |
| `tasks/{id}/` | 可选 | 极小 | 任务列表 |
| `file-history/{id}/` | 可选 | 较大 | 文件编辑备份 |
| `history.jsonl` | 可选 | ~450KB | 全局提示历史 |

## 六、可行性验证结论

### 6.1 核心假设验证

| 假设 | 结果 | 数据 |
|------|------|------|
| session_id 全局唯一 | 确认 | 102 个 session，0 跨 slug 重复 |
| slug 算法一致 | 确认 | 4/4 路径映射匹配 |
| JSONL 含绝对路径 | 确认 | 87% 记录有 cwd 字段（666/762） |
| `.claude.json` key 用原始路径 | 确认 | `D:/development/...`（正斜杠） |

### 6.2 关键风险

| 风险 | 缓解 |
|------|------|
| JSONL 中 cwd 绝对路径 | Phase 1 不处理，Phase 2 用 symlink |
| `.claude.json` key 格式差异 | pull 时按目标机器路径生成 key |
| `last-prompt` 缺失 | 扫描链表找最后 uuid fallback |
| slug 不可逆 | mappings.json 显式记录 |

### 6.3 决策结果

- [x] 存储方案：本地固定目录 `~/.claude-tap/`，与调用位置无关
- [x] Phase 1 范围：本地收集 + 索引，不做云同步
- [x] 云同步后端：R2/S3（Phase 3）
- [x] 加密：SSE-S3 先行，客户端加密 Phase 4
- [ ] 详细设计见 [PRD v0.2](prd-session-sync.md)
