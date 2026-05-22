# Skill 系统

## 完整调用链

```
用户输入 /skill-name args
  → settings.json 触发 UserPromptSubmit 事件
    → 03-user-prompt-submit/base.sh（调度器）
      → skill-inject.sh
        → 1. 检查 prompt 以 / 开头
        → 2. 提取技能名（/ 和空格之间的文本）
        → 3. 提取参数（空格之后的文本）
        → 4. 在 registry.conf 中精确匹配
        → 5. 运行 skill 的 scripts/03UserPromptSubmit.sh
        → 6. 输出作为 additionalContext 注入会话
        → 7. active_add(session_id, skill_name)
          → SKILL.md 加载，Claude 执行任务
          → Claude 完成 → 16Stop.sh 清理
            → active_remove(session_id)
```

## 生命周期管理 (active.sh)

`.active` 文件记录当前活跃的 Skill 会话，格式：`session_id|skill_name`

| 操作 | 函数 | 说明 |
|------|------|------|
| 注册 | `active_add(sid, name)` | 幂等添加/更新 |
| 注销 | `active_remove(sid)` | 按 session_id 删除 |
| 查询 | `active_get(sid)` | 返回 skill_name |
| 批量删 | `active_remove_by_skill(name)` | 按 skill 名删除所有 |
| 列表 | `active_skills()` | 去重列出所有活跃 skill |

## 工具边界检查 (enforce_boundary.sh)

在 `05-pre-tool-use` 事件中运行：

1. 从 `.active` 获取当前活跃 Skill
2. 读取该 Skill 的 `SKILL.md` YAML frontmatter
3. 提取 `allowed-tools` 白名单
4. 检查即将调用的工具是否在白名单中
5. 不在白名单 → 返回 `deny`，阻止调用

如果 `allowed-tools` 未定义，允许所有工具。

## 并发安全 (lock.sh)

使用 `mkdir` 作为跨平台原子锁：

```bash
lock_acquire "/path/to/lock" 10   # 获取锁，超时 10 秒
# ... 临界区操作 ...
lock_release                       # 释放锁
```

特性：
- **原子性**：`mkdir` 在所有平台都是原子操作
- **超时**：可配置等待超时，1 秒轮询间隔
- **僵尸锁清理**：锁超过 60 秒自动清理（防止进程崩溃导致死锁）
- **可重入检测**：通过全局 `_LOCK_DIR` 变量避免重复获取

## 双路日志 (log.sh)

每个 Skill 的日志同时写入两个位置：

| 目标 | 路径 | 用途 |
|------|------|------|
| 统一日志 | `.claude/hooks/logs/YYYY-MM-DD.log` | 与 hook 日志合并，全局面板 |
| 模块日志 | `.claude/skills/log/<tag>/YYYY-MM-DD.log` | 按 Skill 隔离，精确排查 |

使用方式：

```bash
SKILL_TAG="003-5-issue-fix"
source "$PROJECT_DIR/.claude/skills/log.sh"
skill_log "INFO" "context injected for #17"
```

## 自动注册 (skill-register.sh)

在每次 `16-stop` 事件时运行：

1. 扫描 `.claude/skills/*/SKILL.md` 发现所有 Skill 目录
2. 对比 `registry.conf` 中已有条目
3. 新发现的 Skill 自动追加到 `registry.conf`

## SKILL.md Frontmatter

```yaml
---
name: skill-name           # 必需，与目录名一致
description: 一句话描述      # 必需
user-invocable: true        # 可选，是否可通过 / 调用
allowed-tools:              # 可选，工具白名单
  - Bash
  - Read
---
```
