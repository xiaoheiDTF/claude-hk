# 系统架构

## 三层设计

```
settings.json（声明入口）
       │
       ▼
Hooks 管道（29 个事件调度）
       │
       ▼
Skills 技能系统（13 个可扩展技能）
```

**第一层：settings.json** — 声明入口。Claude Code 读取此文件，注册所有 hook 事件。

**第二层：Hooks 管道** — 事件调度。每个生命周期事件触发对应的 `base.sh`，由调度器分发给业务脚本。

**第三层：Skills 系统** — 功能单元。每个 Skill 是独立的功能模块，有自己的定义（`SKILL.md`）、上下文注入（`03UserPromptSubmit.sh`）和清理逻辑（`16Stop.sh`）。

## 数据/控制流

```
Claude Code 启动
  → 读取 settings.json
  → 注册 29 个生命周期事件 hook

每次会话开始
  → 01-session-start/base.sh
  → 首次: init.sh（目录、Python、UTF-8）
  → 每次: 环境巡检

用户输入 prompt
  → 03-user-prompt-submit/base.sh
  → skill-inject.sh 提取 /技能名
  → 匹配 registry.conf
  → 运行 skill 的 03UserPromptSubmit.sh
  → 上下文注入到会话

Claude 执行工具调用
  → 05-pre-tool-use/base.sh
  → enforce_boundary.sh 检查工具白名单
  → 不在白名单则拒绝

Claude 完成响应
  → 16-stop/base.sh
  → skill-register.sh 自动注册新 Skill
  → skill 的 16Stop.sh 清理会话状态
```

## 模块间依赖

```
settings.json
  └── hooks/XX-event/base.sh
        ├── hooks/base.sh（公共基础设施）
        │     ├── hooks/platform.sh（平台检测）
        │     └── hooks/json_get.py（JSON 解析回退）
        ├── hooks/03-user-prompt-submit/skill-inject.sh
        │     └── skills/registry.conf
        └── skills/XXX/scripts/*.sh
              ├── skills/log.sh（日志模块）
              ├── skills/active.sh（状态管理）
              └── skills/lock.sh（文件锁）
```
