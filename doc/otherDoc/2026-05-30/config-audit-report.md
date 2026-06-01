# 配置全景诊断与整理方案

> 创建时间：2026-05-30
> 模块：.claude 配置体系
> 简述：梳理所有 sh 脚本读取的配置文件、使用的环境变量，识别孤儿配置，提出统一整理方案

---

## 一、配置文件全清单（按存储位置分组）

### 1.1 项目内配置 — `.claude/`

| 文件路径 | 格式 | 当前状态 | 读者 | 写者 | 问题诊断 |
|----------|------|----------|------|------|----------|
| `.claude/backend.conf` | KEY=VALUE | ❌ **孤儿配置** | **无人读取** | 手动维护 | `BACKEND_URL=http://127.0.0.1:8080` 存在但没有任何脚本 `source` 或 `grep` 它 |
| `.claude/dirs.conf` | 每行一个目录路径 | ✅ 正常 | `scripts/ensure_dirs.sh` | 手动维护 | — |
| `.claude/.initialized` | JSON | ✅ 正常 | `01-session-start/base.sh` | `init.sh` | 首次初始化标记 |
| `.claude/.python-state` | JSON | ✅ 正常 | `scripts/ensure_python.sh` | `scripts/ensure_python.sh` | Python 环境状态缓存 |
| `.claude/settings.json` | JSON | ✅ Claude 官方 | Claude Code CLI | Claude Code CLI | Hooks 路由配置（29 个事件映射） |
| `.claude/settings.local.json` | JSON | ✅ Claude 官方 | Claude Code CLI | Claude Code CLI | 用户授权权限列表 |

### 1.2 Skills 专属配置 — `.claude/skills/`

| 文件路径 | 格式 | 当前状态 | 读者 | 写者 | 问题诊断 |
|----------|------|----------|------|------|----------|
| `.claude/skills/.active` | 每行 `sid\|skill` | ✅ 正常 | `active.sh`, `enforce_boundary.sh`, `16-stop/base.sh` | `active.sh` (Hook 03 激活时写，Hook 16 清理时删) | 带 `.active.lock` 文件锁保护 |
| `.claude/skills/registry.conf` | 每行一个 skill 名 | ✅ 正常 | `skill-inject.sh` | `skill-register.sh` (16-stop 时自动追加) | — |
| `.claude/skills/001-1-issue-init/labels.conf` | 每行 `名\|颜色\|描述` | ✅ 正常 | `001-1-issue-init/scripts/init.sh` | 手动维护 | — |

### 1.3 用户家目录配置 — `~/.claude-tap-plus/`

| 文件路径 | 格式 | 当前状态 | 读者 | 写者 | 问题诊断 |
|----------|------|----------|------|------|----------|
| `~/.claude-tap-plus/backend.json` | JSON | ✅ 正常 | `skills/backend.sh`, `01-session-start/base.sh` | Go 后端进程启动时写入 | **`.claude/backend.conf` 的替代者**，存储 `host`+`port` |
| `~/.claude-tap-plus/proxy.json` | JSON | ✅ 正常 | Go 后端 `IdleWatchdog` | Go 代理进程 (读写) | 活跃代理会话列表 |
| `~/.claude-tap-plus/profiles.json` | JSON | ✅ 正常 | Go 代理 `config.ResolveTargetConfig` | 手动维护 | API 多 Profile 配置 |
| `~/.claude-tap-plus/.traces/` | 目录树 | ✅ 正常 | 外部查看器 | Go 代理 `TraceWriter` | Trace JSONL 文件 |

### 1.4 项目根目录

| 文件路径 | 格式 | 当前状态 | 读者 | 写者 | 问题诊断 |
|----------|------|----------|------|------|----------|
| `backend.db` | SQLite | ✅ 正常 | Go 后端 `SQLiteStore` | Go 后端 | 会话/Issue 状态机持久化 |
| `backend.db-shm` / `backend.db-wal` | SQLite WAL | ✅ 正常 | SQLite | SQLite | WAL 模式产生的辅助文件 |

---

## 二、环境变量全清单

### 2.1 Claude Code CLI 注入的环境变量（官方）

| 变量名 | 示例值 | 使用位置 | 说明 |
|--------|--------|----------|------|
| `CLAUDE_PROJECT_DIR` | `D:\CodeDevelopment\CodeProject\claude-hk` | **全部脚本** | 项目根目录，所有路径的基准 |
| `CLAUDE_SESSION_ID` | `6b6f69d3-9e1d-...` | `skills/backend.sh` (`_get_session_id`) | 当前会话 UUID |

### 2.2 Hooks 层设置/使用的环境变量

| 变量名 | 设置方 | 使用位置 | 说明 |
|--------|--------|----------|------|
| `HOOK_INPUT` | `hooks/base.sh` (`cat`) | `hooks/base.sh` (`json_get`) | 从 stdin 读取的 Hook JSON 输入 |
| `HOOK_EVENT` | `hooks/base.sh` | `hooks/base.sh` (`log`) | 事件名，用于日志前缀 |
| `LOG_FILE` | `hooks/base.sh` | 所有 `*/base.sh` | 统一日志文件路径：`hooks/logs/YYYY-MM-DD.log` |
| `OS_TYPE` | `hooks/platform.sh` | `hooks/base.sh`, `01-session-start`, `init.sh` | `windows` / `linux` / `macos` |
| `PYTHON_CMD` | `hooks/platform.sh` | `json_get.py` fallback, `ensure_python.sh` | Python 可执行文件路径 |
| `LANG` / `LC_ALL` | `01-session-start/base.sh`, `init.sh` | 子进程继承 | UTF-8 编码设置 |

### 2.3 Skills 层设置/使用的环境变量

| 变量名 | 设置方 | 使用位置 | 说明 |
|--------|--------|----------|------|
| `BACKEND_URL` | `skills/backend.sh` (`_load_backend_url`) | `skills/backend.sh` (全部函数) | 后端 HTTP 地址，从 `~/.claude-tap-plus/backend.json` 解析拼接 |
| `SKILL_TAG` | 各 skill 脚本自身 | `skills/log.sh` | 日志来源标记，如 `002-2-doc-testcode-python` |
| `PROJECT_DIR` | 各 skill 脚本 | skill 内部 | `$CLAUDE_PROJECT_DIR` 的局部别名 |
| `CLAUSE_DIR` | `init.sh`, `ensure_dirs.sh` | 初始化脚本 | `$PROJECT_DIR/.claude` 的别名 |

### 2.4 Go 代理层设置的环境变量

| 变量名 | 设置方 | 使用位置 | 说明 |
|--------|--------|----------|------|
| `ANTHROPIC_BASE_URL` | Go 代理 `config.BuildChildEnv` | Claude Code 子进程 | 指向本地代理地址 (`http://127.0.0.1:PORT`) |
| `ANTHROPIC_API_KEY` / `ANTHROPIC_AUTH_TOKEN` | Go 代理 `runProxy` | Claude Code 子进程 | API 认证（二选一） |

---

## 三、核心问题诊断

### 问题 1：`.claude/backend.conf` 是孤儿配置 ⚠️

```
.claude/backend.conf  内容: BACKEND_URL=http://127.0.0.1:8080
         ↑
         └─ 没有任何脚本读取它！
```

真正的后端地址读取链：
```
Go 后端启动 ──► ~/.claude-tap-plus/backend.json ──► skills/backend.sh (_load_backend_url)
                        {"host":"127.0.0.1","port":8080}
```

**影响**：用户在 `.claude/backend.conf` 里改了地址，发现完全不生效，因为根本没人读。

### 问题 2：配置分散在两个"地盘"

| 配置类型 | 项目内 `.claude/` | 家目录 `~/.claude-tap-plus/` |
|----------|-------------------|------------------------------|
| 后端地址 | `backend.conf` (孤儿) ❌ | `backend.json` ✅ |
| 会话状态 | `.active`, `registry.conf` | `proxy.json` |
| Trace 存储 | — | `.traces/` |
| 数据库 | `backend.db` (项目根目录) | — |

**问题**：开发者需要同时关心两个位置，心智负担重。

### 问题 3：同功能有多份实现

- 机器 ID：`01-session-start` 自己拼 `whoami@hostname`，Go 后端也自己拼 —— 逻辑重复
- 项目 slug：`01-session-start` 从 `transcript_path` sed 提取，Go 代理也自己算 —— 逻辑重复
- backend URL：`.claude/backend.conf` 一份，`~/.claude-tap-plus/backend.json` 一份 —— 数据源二义性

### 问题 4：环境变量缺乏统一规范

- `CLAUDE_PROJECT_DIR` 是官方注入的，但脚本里到处复制到 `PROJECT_DIR`、`CLAUSE_DIR`
- `BACKEND_URL` 是全局变量但只在 `backend.sh` 里管理，没有统一的 "配置加载器"
- 缺少一个 `.claude/env` 或类似机制让 skill 声明自己需要的环境变量

---

## 四、整理方案（推荐）

### 方案 A：保守整理 — 只删孤儿，保留现有结构

**改动量：极小**

1. **删除 `.claude/backend.conf`** — 彻底移除孤儿配置，避免误导
2. **在 `.claude/init.sh` 中添加注释说明**：后端地址由 Go 进程自动管理，勿手动编辑
3. **保留其他所有文件不动**

**优点**：零风险，立刻解决最大困惑源
**缺点**：配置仍然分散在两个地盘

---

### 方案 B：中度整理 — 统一配置加载层

**改动量：中等**

1. **删除 `.claude/backend.conf`**
2. **新建 `.claude/lib/config.sh`** — 统一配置加载器：
   ```bash
   # config.sh — 所有脚本的统一配置入口
   config_get_backend_url()   { ...读 ~/.claude-tap-plus/backend.json... }
   config_get_proxy_url()     { ...读 ~/.claude-tap-plus/backend.json proxy_url... }
   config_get_project_dir()   { echo "$CLAUDE_PROJECT_DIR"; }
   config_get_session_id()    { ...优先 CLAUDE_SESSION_ID... }
   ```
3. **所有脚本统一 `source .claude/lib/config.sh`**，不再各自实现 `_load_backend_url`
4. **把 `backend.sh` 中 `_load_backend_url` 的逻辑迁移到 `config.sh`**，`backend.sh` 改为 `source config.sh`

**优点**：配置读取逻辑统一，一处修改全局生效
**缺点**：需要改多个脚本的 `source` 链

---

### 方案 C：彻底整理 — 所有配置归拢到 `.claude/`

**改动量：较大**

1. **删除 `.claude/backend.conf`**
2. **Go 后端启动时同时写入两份 `backend.json`**：
   - `~/.claude-tap-plus/backend.json`（保持 Go 进程自发现）
   - `.claude/runtime/backend.json`（供 shell 脚本读取，避免跨目录）
3. **新建 `.claude/runtime/` 目录**，存放所有运行时生成的配置/状态：
   ```
   .claude/runtime/
   ├── backend.json       # 后端地址（Go 写入，shell 读取）
   ├── proxy.json         # 代理会话状态（Go 写入，shell 读取）
   ├── .active            # skill 激活状态（从 skills/ 迁移过来）
   └── .initialized       # 从 .claude/ 根目录迁移过来
   ```
4. **`.claude/skills/` 只保留 skill 定义（SKILL.md + scripts/）**，运行时状态全部迁到 `.claude/runtime/`
5. **统一入口脚本 `.claude/lib/bootstrap.sh`**：
   ```bash
   # 所有脚本开头只 source 这一份
   source "$CLAUDE_PROJECT_DIR/.claude/lib/bootstrap.sh"
   # 自动设置：PROJECT_DIR, CLAUSE_DIR, RUNTIME_DIR, BACKEND_URL, LOG_FILE, ...
   ```

**优点**：所有配置和状态一目了然，项目自包含（除了 `~/.claude-tap-plus/.traces/` 和 `backend.db`）
**缺点**：需要改 Go 代码 + 大量 shell 脚本，改动面广

---

## 五、推荐执行路径

**建议按阶段执行，不要一次性做大改：**

| 阶段 | 动作 | 预计改动文件数 |
|------|------|--------------|
| **Phase 1** | 删除 `.claude/backend.conf`，更新 `AGENTS.md` 或相关文档说明配置位置 | 1 文件删除 + 文档 |
| **Phase 2** | 新建 `.claude/lib/config.sh`，把 `backend.sh`、 `01-session-start`、 `29-session-end` 中重复的 backend URL / proxy URL 读取逻辑收拢 | 4~5 个脚本 |
| **Phase 3** | 新建 `.claude/runtime/` 目录，逐步将 `.active`、`.initialized`、`.python-state` 等运行时状态迁移过去 | 10+ 个脚本 |
| **Phase 4** | Go 后端写入 `.claude/runtime/backend.json` 双写，过渡后去掉 `~/.claude-tap-plus/backend.json` 的 shell 依赖 | Go 代码 + shell 脚本 |

---

## 六、配置速查表（整理后理想状态）

| 你要找的配置 | 当前位置（混乱） | 整理后位置（方案 B） | 整理后位置（方案 C） |
|-------------|-----------------|---------------------|---------------------|
| 后端 HTTP 地址 | `~/.claude-tap-plus/backend.json` | 同左，但统一由 `config.sh` 读取 | `.claude/runtime/backend.json` |
| 代理地址 | `~/.claude-tap-plus/backend.json` (proxy_url) | 同左，但统一由 `config.sh` 读取 | `.claude/runtime/proxy.json` |
| Skill 激活状态 | `.claude/skills/.active` | 同左 | `.claude/runtime/.active` |
| Skill 注册表 | `.claude/skills/registry.conf` | 同左 | 同左 |
| 初始化标记 | `.claude/.initialized` | 同左 | `.claude/runtime/.initialized` |
| 预创建目录列表 | `.claude/dirs.conf` | 同左 | 同左 |
| Issue 标签定义 | `.claude/skills/001-1-issue-init/labels.conf` | 同左 | 同左 |
| 统一配置加载 | 不存在 | `.claude/lib/config.sh` | `.claude/lib/bootstrap.sh` |
| Trace 存储目录 | `~/.claude-tap-plus/.traces/` | 同左 | 同左 |
| SQLite 数据库 | 项目根目录 `backend.db` | 同左 | 同左 |



● 总结

  基础对象一览

  ┌──────────────────────┬────────────┬─────────────────────────┬──────────────────────────┬──────────────────────────────────┐
  │         对象         │    位置    │        共享/独立        │         并发保护         │            多会话风险            │
  ├──────────────────────┼────────────┼─────────────────────────┼──────────────────────────┼──────────────────────────────────┤
  │ ProxyHandler.proxies │ 后端进程内 │ 🔴 共享（全局）         │ sync.RWMutex ✅          │ 多代理并发注册/注销，锁保护正确  │
  ├──────────────────────┼────────────┼─────────────────────────┼──────────────────────────┼──────────────────────────────────┤
  │ ReverseProxy         │ 代理进程内 │ 🟢 独立（每会话一个）   │ atomic.Int64 + sync.Once │ 无风险，各实例隔离               │
  ├──────────────────────┼────────────┼─────────────────────────┼──────────────────────────┼──────────────────────────────────┤
  │ TraceWriter          │ 代理进程内 │ 🟢 独立（每会话一个）   │ sync.Mutex               │ 无风险                           │
  ├──────────────────────┼────────────┼─────────────────────────┼──────────────────────────┼──────────────────────────────────┤
  │ proxy.json           │ 磁盘文件   │ 🔴 共享                 │ sync.Mutex（进程内）     │ 多进程写同一文件 → ⚠️ 有竞争风险 │
  ├──────────────────────┼────────────┼─────────────────────────┼──────────────────────────┼──────────────────────────────────┤
  │ backend.json         │ 磁盘文件   │ 🔴 共享                 │ 无锁                     │ 多代理同时写 → ⚠️ 有竞争风险     │
  ├──────────────────────┼────────────┼─────────────────────────┼──────────────────────────┼──────────────────────────────────┤
  │ SQLiteStore          │ 后端进程内 │ 🔴 共享（单例）         │ WAL 模式 + 事务          │ 安全，SQLite 自己处理并发        │
  ├──────────────────────┼────────────┼─────────────────────────┼──────────────────────────┼──────────────────────────────────┤
  │ IdleWatchdog         │ 后端进程内 │ 🟢 独立（单 goroutine） │ 无需锁                   │ 安全，单线程读取                 │
  ├──────────────────────┼────────────┼─────────────────────────┼──────────────────────────┼──────────────────────────────────┤
  │ backend.db           │ 磁盘文件   │ 🔴 共享                 │ WAL 模式                 │ 安全                             │
  └──────────────────────┴────────────┴─────────────────────────┴──────────────────────────┴──────────────────────────────────┘

  潜在并发问题

  1. proxy.json 跨进程无锁：proxyMu 只保护同一进程内的读写。如果两个代理进程同时写 proxy.json，可能丢失数据。需要文件锁（如 flock/mkdir 原子操作）。
  2. backend.json 跨进程写：代理和后端都写这个文件，无跨进程保护。

  项目配置文件清单

  ┌──────────────────────────────────┬───────────┬──────────────────┬───────────────────────┬────────────────────┬────────────────────────────────┐
  │             配置文件             │   格式    │       用途       │         读者          │        写者        │              状态              │
  ├──────────────────────────────────┼───────────┼──────────────────┼───────────────────────┼────────────────────┼────────────────────────────────┤
  │ ~/.claude-tap-plus/backend.json  │ JSON      │ 后端 host+port   │ hooks, 代理, watchdog │ 后端启动时         │ ✅ 活跃                        │
  ├──────────────────────────────────┼───────────┼──────────────────┼───────────────────────┼────────────────────┼────────────────────────────────┤
  │ ~/.claude-tap-plus/proxy.json    │ JSON      │ 活跃会话注册表   │ watchdog, 代理        │ 代理 trace-init 时 │ ✅ 活跃                        │
  ├──────────────────────────────────┼───────────┼──────────────────┼───────────────────────┼────────────────────┼────────────────────────────────┤
  │ ~/.claude-tap-plus/.traces/      │ JSONL     │ API 调用 trace   │ 用户查阅              │ 代理写入           │ ✅ 活跃                        │
  ├──────────────────────────────────┼───────────┼──────────────────┼───────────────────────┼────────────────────┼────────────────────────────────┤
  │ backend.db                       │ SQLite    │ 会话/Issue 数据  │ 后端 API              │ 后端服务           │ ✅ 活跃                        │
  ├──────────────────────────────────┼───────────┼──────────────────┼───────────────────────┼────────────────────┼────────────────────────────────┤
  │ ~/.claude.json                   │ JSON      │ Claude Code 配置 │ config 包             │ Claude Code        │ ✅ 活跃                        │
  ├──────────────────────────────────┼───────────┼──────────────────┼───────────────────────┼────────────────────┼────────────────────────────────┤
  │ ~/.claude-tap-plus/profiles.json │ JSON      │ 命名配置 profile │ config 包             │ 用户手动           │ ✅ 活跃                        │
  ├──────────────────────────────────┼───────────┼──────────────────┼───────────────────────┼────────────────────┼────────────────────────────────┤
  │ .claude/backend.conf             │ KEY=VALUE │ 旧的后端 URL     │ ~~backend.sh~~        │ ~~旧脚本~~         │ ❌ 已废弃（改读 backend.json） │
  ├──────────────────────────────────┼───────────┼──────────────────┼───────────────────────┼────────────────────┼────────────────────────────────┤
  │ ~/.claude/settings.json          │ JSON      │ Claude Code 设置 │ config 包             │ Claude Code        │ ✅ 活跃                        │
  └──────────────────────────────────┴───────────┴──────────────────┴───────────────────────┴────────────────────┴────────────────────────────────┘

  .claude/backend.conf 已经被我们改掉的 backend.sh 不再使用了，可以删掉。