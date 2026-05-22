# PRD: Claude Code Session 跨设备同步

> 版本：v0.2 | 日期：2026-05-21 | 作者：xiaoheiDTF

---

## 一、背景与目标

### 1.1 问题

Claude Code 的 session（对话记录）存储在本地 `~/.claude/projects/{slug}/` 目录下。slug 由工作目录的绝对路径生成（将 `:` `/` `\` 替换为 `-`），导致：

- 机器 A 路径 `D:\code\claude-tap` → slug `D--code-claude-tap`
- 机器 B 路径 `/home/user/claude-tap` → slug `-home-user-claude-tap`

`claude --resume` 按 slug 查找 session，不同机器 slug 不同 → **跨设备无法恢复会话**。

Claude Code 官方未实现此功能（[GitHub #31992](https://github.com/anthropics/claude-code/issues/31992) 仍然 Open）。

### 1.2 目标

实现一个 **claude-tap-plus 内置的 session 同步工具**，让用户在不同机器间：

1. **上传**当前机器的 Claude session 到云端
2. **下载**其他机器的 session 到本地
3. **自动适配**路径映射，使 `claude --resume` 能正确找到跨设备的 session
4. 可选：**端到端加密**，确保云端无法读取对话内容

### 1.3 非目标

- 不修改 Claude Code 本身的行为
- 不实现实时协同编辑
- 不做 Claude Code 到其他 AI 工具（Codex/Gemini）的 session 迁移

---

## 二、术语定义

| 术语 | 定义 |
|------|------|
| **session** | Claude Code 一次完整对话，对应一个 `{sessionId}.jsonl` 文件 |
| **slug** | 由 cwd 绝对路径生成的目录名，如 `D--code-claude-tap` |
| **JSONL** | 每行一个 JSON 的文件格式，Claude Code 用于存储流式对话记录 |
| **UUID 链** | JSONL 记录通过 `parentUuid → uuid` 形成的单向链表 |
| **resume** | `claude --resume` 或 `claude -c` 恢复历史会话 |
| **远端** | 云存储后端（R2/S3） |
| **slug 映射** | 原始 slug → 目标机器 slug 的对应关系 |

---

## 三、Claude Code 本地存储结构

> 详细分析见 [[claude-session-analysis]](claude-session-analysis.md)

### 3.1 核心文件

```
~/.claude/
├── projects/{slug}/
│   ├── {sessionId}.jsonl              ← 对话记录（必须同步）
│   ├── {sessionId}/
│   │   ├── subagents/*.jsonl          ← 子 agent 对话（推荐同步）
│   │   └── tool-results/*.txt         ← 大型工具输出（可选同步）
│   └── memory/                        ← 项目级 memory（推荐同步）
├── tasks/{sessionId}/                 ← 任务列表（可选同步）
├── file-history/{sessionId}/          ← 文件编辑备份（可选同步）
├── history.jsonl                      ← 全局提示历史（可选同步）
├── plans/*.md                         ← 计划文件（可选同步）
└── .claude.json                       ← 项目元数据（必须同步）
```

### 3.2 需要同步的文件优先级

| 优先级 | 文件 | 必要性 | 体积估算 | 说明 |
|--------|------|--------|----------|------|
| P0 | `projects/{slug}/{id}.jsonl` | 必须 | ~1MB/30轮 | 对话记录，resume 的核心 |
| P0 | `.claude.json` 中该项目条目 | 必须 | <1KB | lastSessionId、lastCost 等 |
| P1 | `projects/{slug}/{id}/subagents/` | 推荐 | 较小 | 子 agent 对话 |
| P1 | `projects/{slug}/memory/` | 推荐 | 极小 | 项目 memory |
| P2 | `tasks/{id}/` | 可选 | 极小 | 任务列表 |
| P2 | `file-history/{id}/` | 可选 | 较大 | 文件编辑备份 |
| P2 | `history.jsonl` | 可选 | ~450KB | 全局提示历史 |
| P2 | `plans/*.md` | 可选 | 极小 | 计划文件 |

### 3.3 slug 生成算法

```
输入: cwd 绝对路径
算法: 将所有 ":" "/" "\" 替换为 "-"
示例:
  D:\code\claude-tap    → D--code-claude-tap
  /home/user/claude-tap → -home-user-claude-tap
  C:\Users\Admin        → C--Users-Admin
```

**关键特性**：
- 不可逆：路径中自然包含 `-` 时有歧义（`my-project` → 来自 `my/project` 还是 `my-project`？）
- 大小写敏感：`D--` 和 `d--` 在 Linux 上是不同目录，Windows 上指向同一目录
- 子目录是独立项目：`/code/app` 和 `/code/app/sub` 是两个 slug

---

## 四、本地固定存储设计

### 4.1 设计原则

**所有 trace 和 session 数据统一存储到固定目录，与调用位置无关。**

无论用户在哪个目录执行 `claude-tap-plus`，数据都写入同一个固定位置。类似 Claude Code 自身的 `~/.claude/` 设计。

### 4.2 存储根目录

**存储路径跟着可执行文件位置走，不跟着 cwd 走。**

```
可执行文件位置: C:\tools\claude-tap-plus.exe
trace 存储路径:  C:\tools\.traces\{project}\

可执行文件位置: D:\dev\claude-tap\claude_tap_plus\claude-tap-plus.exe
trace 存储路径:  D:\dev\claude-tap\claude_tap_plus\.traces\{project}\
```

实现方式：`os.Executable()` 获取可执行文件路径，取其目录下的 `.traces/`。

### 4.3 完整目录结构

```
{executable-dir}/
├── claude-tap-plus.exe                   ← 可执行文件
├── .traces/                              ← trace 数据（跟着可执行文件走）
│   └── {project-name}/                   ← 按项目名隔离
│       ├── 2026-05-20_100000_a3f2c1.jsonl
│       └── 2026-05-20_143000_b7e9d4.jsonl
├── sessions/                             ← Claude Code session 数据（Phase 2）
│   └── {project-name}/
│       ├── {sessionId}.jsonl
│       └── meta.json
└── certs/                                ← TLS 证书（forward proxy 用）
```

### 4.4 与旧方案的区别

| 维度 | 旧方案 | 新方案 |
|------|--------|--------|
| trace 存储位置 | `{cwd}/.traces/` | `{exe-dir}/.traces/{project}/` |
| session 存储位置 | 直接读 `~/.claude/projects/{slug}/` | 拷贝到 `{exe-dir}/sessions/{project}/` |
| 调用位置依赖 | 是（存在项目目录下） | **否（跟着可执行文件走）** |
| 安装位置依赖 | 否 | **是（装 C 盘存 C 盘，装 D 盘存 D 盘）** |
| 项目隔离 | 按 slug | 按项目名（git remote repo name 优先） |

### 4.5 项目名识别优先级

```
1. git remote URL 的 repo 名（如 claude-tap）
2. cwd 的 filepath.Base（如 claude-tap）
3. 用户 --tap-project 参数手动指定
```

git remote URL 最可靠（跨机器一致），cwd 的 Base name 作为 fallback。

### 4.6 config.json 结构

```json
{
  "version": 1,
  "machine_id": "machine-A",
  "traces": {
    "enabled": true
  },
  "sync": {
    "backend": {
      "type": "r2",
      "account_id": "",
      "bucket": "claude-sessions",
      "access_key_id": "",
      "secret_access_key": ""
    },
    "encryption": {
      "enabled": false,
      "method": "aes-256-gcm"
    },
    "scope": {
      "include_subagents": true,
      "include_memory": true,
      "include_tasks": false,
      "include_file_history": false,
      "include_plans": false,
      "include_history": false
    }
  }
}
```

---

## 五、核心问题：slug 路径映射

### 5.1 问题描述

```
机器 A: D:\code\claude-tap  → slug: D--code-claude-tap
机器 B: /home/user/claude-tap → slug: -home-user-claude-tap
```

同一个代码库，两台机器 slug 不同。Claude Code 只按 slug 查找 session。

### 5.2 解决方案：slug 映射表

维护一个映射表，记录「项目身份 → 各机器 slug」的对应关系：

```json
{
  "projects": {
    "claude-tap": {
      "identifiers": ["git_remote:https://github.com/xiaoheiDTF/claude-tap.git"],
      "machines": {
        "machine-A": {
          "slug": "D--code-claude-tap",
          "cwd": "D:\\code\\claude-tap",
          "os": "windows"
        },
        "machine-B": {
          "slug": "-home-user-claude-tap",
          "cwd": "/home/user/claude-tap",
          "os": "linux"
        }
      }
    }
  }
}
```

### 5.3 项目身份识别

slug 不可逆，需要用其他方式确认「两台机器上的项目是同一个」。识别方式：

| 方式 | 可靠性 | 说明 |
|------|--------|------|
| **git remote URL** | 高 | 同一仓库的 remote URL 相同 |
| **项目目录名** | 中 | `filepath.Base(cwd)` 大多数情况相同 |
| **用户手动指定** | 高 | 首次同步时交互确认 |

优先使用 git remote URL，fallback 到目录名，最后让用户确认。

---

## 六、本地 session 存储设计

### 6.1 session 本地副本

`session-push` 的工作分为两步：

1. **本地收集**：从 `~/.claude/projects/{slug}/` 读取 session 文件，拷贝到 `~/.claude-tap/sessions/{project}/`
2. **远端推送**：从 `~/.claude-tap/sessions/{project}/` 上传到云端（Phase 2+）

Phase 1 只做本地收集和索引，不做云同步。

### 6.2 session 索引文件

每个项目维护一个 `meta.json`：

```json
{
  "project": "claude-tap",
  "git_remote": "https://github.com/xiaoheiDTF/claude-tap.git",
  "local_slug": "D--development-code-claude-tap",
  "local_cwd": "D:\\development\\code\\claude-tap",
  "machine_id": "machine-A",
  "sessions": [
    {
      "session_id": "731bcd6d-b7eb-401f-a9f9-4ed73ce85b38",
      "file": "731bcd6d-b7eb-401f-a9f9-4ed73ce85b38.jsonl",
      "file_size": 940784,
      "record_count": 762,
      "first_timestamp": "2026-05-20T01:20:56.622Z",
      "last_timestamp": "2026-05-20T14:44:00.000Z",
      "models_used": ["glm-5.1"],
      "git_branch": "claude-tap-plus",
      "source_slug": "D--development-code-claude-tap",
      "collected_at": "2026-05-20T14:50:00Z"
    },
    {
      "session_id": "4a430bc7-cdd0-4010-adf2-70a4ddaacd36",
      "file": "4a430bc7-cdd0-4010-adf2-70a4ddaacd36.jsonl",
      "file_size": 892405,
      "record_count": 547,
      "first_timestamp": "2026-05-18T10:16:21.345Z",
      "last_timestamp": "2026-05-20T09:21:00.000Z",
      "models_used": ["claude-sonnet-4-6"],
      "git_branch": "claude-tap-plus",
      "source_slug": "D--development-code-claude-tap",
      "collected_at": "2026-05-20T14:50:00Z"
    }
  ]
}
```

### 6.3 JSONL 中绝对路径的处理

**实测数据**：JSONL 中 87% 的记录（666/762）包含 `cwd` 字段，硬编码了绝对路径。

**Phase 1 策略**：不修改 JSONL 内容。保留原始路径。

**原因**：
- 修改 JSONL 内容可能误伤消息体中引用的路径
- Phase 1 只做本地收集和索引，不涉及跨机器 resume
- 跨机器 resume 的路径问题在 Phase 2 通过 symlink 解决

### 6.4 .claude.json 合并策略

`.claude.json` 是所有项目的全局配置文件。合并规则：

| 字段 | 策略 |
|------|------|
| `lastSessionId` | 取时间戳最新的 |
| `lastHintSessionId` | 取时间戳最新的 |
| `lastCost` | 取时间戳最新的 |
| `lastTotalInputTokens` | 取时间戳最新的 |
| `lastTotalOutputTokens` | 取时间戳最新的 |
| `lastSessionFirstPrompt` | 取时间戳最新的 |
| `lastSessionModified` | 取时间戳最大的 |
| `lastGracefulShutdown` | 取远端值（如果本地是 false 说明中断了） |
| `lastModelUsage` | 合并所有模型 |
| 其他项目条目 | 不触碰（只修改当前 slug 对应的条目） |

**注意**：`.claude.json` 的项目 key 格式是**原始路径**（`D:/development/code/claude-tap`，正斜杠），不是 slug 格式。pull 时需要按目标机器路径生成正确的 key。

---

## 七、命令行接口

### 7.1 新增命令

```bash
# 收集当前项目的 Claude session 到本地存储
claude-tap-plus session-push

# 从本地存储恢复 session 到 ~/.claude/（使 claude --resume 可用）
claude-tap-plus session-pull

# 查看本地存储状态
claude-tap-plus session-status

# 交互式配置（首次使用）
claude-tap-plus session-sync-setup
```

### 7.2 参数

```
session-push:
  --all              收集所有项目（默认只收集当前项目）
  --force            强制覆盖已有文件
  --dry-run          只显示将要收集的内容，不实际执行

session-pull:
  --all              恢复所有项目
  --project NAME     指定项目名（而非使用当前 cwd）
  --dry-run          只显示将要恢复的内容

session-status:
  --verbose          显示详细的文件级对比

session-sync-setup:
  （交互式引导，配置 machine_id 等）
```

### 7.3 已有参数变更

| 参数 | 变更 |
|------|------|
| `--tap-output-dir` | 废弃。trace 固定存储到 `~/.claude-tap/traces/{project}/` |
| `--tap-project` | 新增。手动指定项目名（覆盖自动检测） |
| `--claude` | 已支持。`--claude resume` 等同于 `-- resume` |

---

## 八、可行性验证结论

### 8.1 PRD 核心假设验证

| 假设 | 验证结果 | 数据来源 |
|------|---------|---------|
| session_id 全局唯一 | **确认**。102 个 session，0 个跨 slug 重复 | 本机实测 |
| slug 由路径替换生成 | **确认**。4/4 路径映射完全匹配 | 本机实测 |
| 同一 session 不会被两台机器同时写 | **确认**。session 追加写入，同一时间只有一个进程持有 | Claude Code 架构分析 |
| JSONL 中硬编码了绝对路径 | **确认**。87% 的记录包含 cwd 字段 | 本机实测 666/762 |
| `.claude.json` 项目 key 用原始路径 | **确认**。用的是 `D:/development/...`（正斜杠），不是 slug | 本机实测 |

### 8.2 关键风险点

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| JSONL 中的 cwd 绝对路径 | 跨机器 resume 时路径不一致 | Phase 2 通过 symlink 解决，Phase 1 不处理 |
| `.claude.json` key 格式差异 | pull 时生成错误的 key | 统一用目标机器的 cwd 生成 key |
| `last-prompt` 缺失 | resume 找不到链表末尾 | 扫描整个链表找最后 uuid 作为 fallback |
| slug 不可逆 | 无法从 slug 还原原始路径 | 通过 mappings.json 显式记录 |

### 8.3 已有方案对比

| 方案 | 做法 | 跨机器 resume | 状态 |
|------|------|--------------|------|
| Claude Sync（开源） | `age` 加密同步 `~/.claude/` 到 S3/R2/GCS | 路径不同则不行 | 作者承认待修复 |
| CLAUDE_CONFIG_DIR 指向网盘 | 所有机器共享同一目录 | JSONL 并发写入冲突 | 可行但不稳定 |
| Hook 自动保存 session_id | SessionEnd Hook 写 `.claude_session` | 仅同机器 | 不适用 |
| **claude-tap-plus（本方案）** | **本地收集 + slug 映射 + 云同步** | **symlink 路径适配** | **设计中** |

---

## 九、分阶段交付

### Phase 1：本地固定存储（当前阶段）

- [ ] 确定存储根目录 `~/.claude-tap/`
- [ ] trace 文件存储到 `~/.claude-tap/traces/{project}/`
- [ ] 修改 `NewTracePath` 使用固定目录
- [ ] `session-push` 收集 Claude session 到 `~/.claude-tap/sessions/{project}/`
- [ ] 生成 `meta.json` session 索引
- [ ] 自动检测项目名（git remote → cwd basename）
- [ ] `session-status` 显示本地存储状态

### Phase 2：本地恢复

- [ ] `session-pull` 从 `~/.claude-tap/sessions/` 恢复到 `~/.claude/projects/{slug}/`
- [ ] slug 映射表自动生成
- [ ] `.claude.json` 项目元数据合并
- [ ] symlink 路径适配（让 `claude --resume` 跨机器可用）
- [ ] subagents/ 同步

### Phase 3：云同步

- [ ] `session-sync-setup` 交互配置云后端
- [ ] 上传 `~/.claude-tap/sessions/` 到 R2/S3
- [ ] 下载远端 session 到本地
- [ ] 增量同步（JSONL 追加）
- [ ] 上传前自动脱敏

### Phase 4：安全与高级功能

- [ ] 客户端加密（AES-256-GCM）
- [ ] 密钥管理（系统 keychain）
- [ ] `--all` 多项目批量同步
- [ ] 自动同步（session 结束后自动上传）
- [ ] 同步历史记录和回滚

---

## 十、验收标准

### P0（Phase 1 必须）

1. 执行 `claude-tap-plus` 代理 Claude Code，trace 写入 `~/.claude-tap/traces/{project}/`
2. 在任意目录执行 `claude-tap-plus`，trace 都写入同一固定目录
3. `session-push` 收集当前项目的 Claude session 到 `~/.claude-tap/sessions/{project}/`
4. `meta.json` 索引正确生成，包含 session_id、时间戳、模型、分支等元数据
5. 项目名自动检测正确（git remote URL 优先，cwd basename fallback）

### P1（Phase 2）

6. `session-pull` 将 session 恢复到 `~/.claude/projects/{slug}/` 正确位置
7. 恢复后 `claude --resume` 能找到并恢复 session
8. `.claude.json` 元数据正确合并
9. slug 映射表正确生成

### P2（Phase 3+）

10. session 数据上传到 R2/S3
11. 从其他机器下载 session 并恢复
12. 客户端加密后云端无法读取明文
