# 设计亮点

## 两层调度

`base.sh` 只负责调度，业务脚本独立。

```
XX-event-name/
├── base.sh            # 调度器：source 公共基础 → 调用业务脚本
└── other-script.sh    # 业务逻辑
```

**好处**：
- 扩展时只需新建脚本 + 在 base.sh 中加一行调用
- 不需要修改 `settings.json`
- 业务脚本可独立测试

## 自动注册

`skill-register.sh` 在每次 Stop 事件时扫描 `skills/` 目录：

```
扫描 .claude/skills/*/SKILL.md
  → 提取 name 字段
  → 对比 registry.conf
  → 新发现的自动追加
```

**好处**：新增 Skill 放入目录即生效，无需手动注册。

## 渐进降级

### JSON 解析

```
jq（最快）→ Python + json_get.py（中等）→ sed（最基础）
```

优先使用最高效的工具，不可用时自动降级到下一级。

### Python 环境

```
嵌入版 Python → 系统 python3 → 系统 python → 自动下载
```

Windows 环境下自动下载 embeddable Python，无需用户手动安装。

## 幂等初始化

- **标记文件** `.initialized`：存在则跳过初始化
- **bashrc 标记块**：`>>> claude-hk-utf8 >>>` 检测避免重复写入
- **目录创建**：只创建不存在的目录
- **Python 状态**：`.python-state` 持久化配置结果

所有操作可安全重复执行，不会产生副作用。

## 并发安全

### 文件锁 (lock.sh)

```bash
lock_acquire "$LOCK_FILE" 10   # mkdir 原子操作
# 临界区
lock_release                   # rm -rf
```

- `mkdir` 在所有平台都是原子操作
- 超时机制：10 秒内获取不到锁则放弃
- 僵尸锁：超过 60 秒自动清理

### 活跃状态管理 (active.sh)

`.active` 文件记录 `session_id|skill_name`，所有读写操作都在锁保护下。

## 双路日志

```
skill_log "INFO" "message"
  → hooks/logs/YYYY-MM-DD.log      # 全局面板
  → skills/log/<tag>/YYYY-MM-DD.log # 模块隔离
```

**好处**：
- 全局日志可以看所有 hook 和 skill 的活动时间线
- 模块日志可以精确排查单个 skill 的问题
