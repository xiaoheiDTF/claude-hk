# Claude Tap Plus 后端服务设计

> 创建时间：2026-05-26
> 模块：claude-tap-plus
> 简述：为 claude-tap-plus 设计后端服务，支持会话注册、消息存储、Issue 全局领取去重

---

## 一、原始需求记录

设计一个工具：

**功能1：聊天记录存储**
- 支持 Claude 消息原始请求及响应的存储
- 加载方式：`claude-tap-plus claude` 启动后，向 claude-tap-plus 后端服务器发送注册信息
- 注册内容：机器 ID、OS、文件夹路径、session 信息
- 存储方式参考 Claude Code 的本地存储（文件夹 → session）

**功能2：适配 001-4-issue-claim 技能，提供 Issue 领取的全局信息**
- 不可以通过 git 获取对应的 issue 来完成，因为只有一个 GitHub 账号，容易导致 issue 被同一个 agent 获取到
- 实现思路：
  1. `claude-tap-plus claude` 启动时通过 session_id 向后端服务器注册唯一 ID
  2. 调用需要修改 issue 状态的技能时，都去后端服务器查询
  3. `001-4-issue-claim` 技能获取 issue 列表后，需要去后台查询这些 issue 的实际状态
  4. 不存在或已被其他会话领取的 issue 会被去重过滤
  5. 服务器内部只支持通过 issue 技能提交上来的数据进行去重
  6. issue-claim 脚本中需要写入访问后端服务的脚本，将 issue 唯一编号、项目 GitHub 地址、项目名字传入后端
  7. 后端判断是否有重复，无重复则直接返回对应 issue
  8. 用户确认后，AI 调用脚本发送请求到后端，后端记录被领取的 issue 唯一 ID
  9. 后续技能根据技能不同，标记对应 issue 进行对应更改

**功能2 补充方案：升级 001-5-issue-fix 技能为 worktree 模式**（此补充方案暂不纳入当前实现，留作未来迭代）
- 将 001-5-issue-fix 升级为创建 worktree
- 创建 worktree 后，将当前领取的 issue 信息注入到该 worktree 的 CLAUDE.md 中
- 用户主动 cd 到 worktree 中启动 Claude 去完成任务

---

## 二、理解与梳理

整体是为 `claude-tap-plus` 增加一个**后端服务层**，解决两个核心问题：

1. **会话追踪与消息归档** — 多机器、多项目环境下，统一收集 Claude 对话原始数据
2. **多 Agent Issue 协作** — 单 GitHub 账号下多 Agent 并行工作时的 Issue 领取冲突问题，通过后端中心化服务解决

核心设计思路：`claude-tap-plus` 作为本地代理，在启动时向后端注册会话，后续 Issue 相关技能通过 HTTP 调用后端 API 来完成去重、状态查询、状态变更，不再依赖 GitHub 账号区分 Agent。

### 后端服务启动方式

后端服务作为 `claude-tap-plus` 的子命令运行，与代理进程独立：

```bash
# 启动后端（默认端口 8080）
claude-tap-plus backend

# 指定端口和数据库路径
claude-tap-plus backend --port 8080 --db ./backend.db

# 指定配置文件
claude-tap-plus backend --config ./backend.conf
```

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--port` | `8080` | 监听端口 |
| `--db` | `./backend.db` | SQLite 数据库路径 |
| `--config` | 无 | 配置文件路径（可选） |

### 后端服务代码结构

```
claude_tap_plus/
├── cmd/
│   ├── claude-tap/          # 现有代理 CLI
│   └── claude-tap-server/   # 新增：后端服务 CLI
├── internal/
│   ├── session/             # 现有：session push/pull
│   ├── backend/             # 新增：后端服务
│   │   ├── server.go        # HTTP server 生命周期
│   │   ├── handler.go       # API handler
│   │   ├── db.go            # SQLite 初始化 + 迁移
│   │   └── model.go         # 数据模型
│   └── ...
```

---

## 三、功能需求清单


### 模块 B：Issue 全局管理

| 编号 | 功能 | 简述 |
|------|------|------|
| B1 | Issue 状态查询 | 接收 issue 编号 + 项目信息，返回当前领取状态（空闲/已被领取/领取者信息） |
| B2 | Issue 批量去重 | 接收 issue 列表，过滤已领取的，返回可用 issue 列表 |
| B3 | Issue 领取 | 标记 issue 为已领取，绑定 session_id，记录领取时间 |
| B4 | Issue 状态流转 | 支持不同技能触发不同状态变更（fixing → PR → testing → review） |
| B5 | Issue 释放/过期 | 会话失效或任务完成时释放 issue，允许重新领取 |
| B8 | 降级策略 | 后端不可用时各技能的降级行为定义 |

### 模块 C：本地代理改造（claude-tap-plus）后端服务基础

| 编号 | 功能 | 简述 |
|------|------|------|
| C1 | 启动时注册 | `claude-tap-plus claude` 启动时自动向后端注册会话 |
| C2 | 消息拦截转发 | 将请求/响应原始数据同步发送到后端存储 |
 A1 | 会话注册 API | 接收 session_id、机器 ID、OS、项目路径，注册唯一会话 |
| A4 | 消息查询 API | 按会话/项目/时间范围查询历史消息 |


### 模块 D：Issue 技能改造

| 编号 | 功能 | 简述 |
|------|------|------|
| D0 | 公共后端调用基础设施 | backend.sh 共享模块、后端调用函数、label 同步函数 |
| D1 | issue-claim 脚本改造 | 获取 issue 列表后调用后端 B2 去重，确认后调用 B3 领取 |
| D2 | issue-fix 标记状态 | 创建分支后标记 fixing 状态（worktree 方案留作未来迭代） |
| D3 | 后续技能适配 | issue-done / issue-pr / issue-test / issue-review 调用后端 B4 更新状态 |
