---
title: 集成 Windows 前台唤醒和任务完成通知到 hooks 体系
labels: enhancement
assignee: xiaoheiDTF
priority: P2
status: published
created: 2026-05-23
---

## 描述

将 Windows 前台唤醒（Win32 API）和任务完成通知（Toast）功能集成到现有 hooks 体系中。

### 背景

Windows 上使用 Claude Code 时，当 Claude 需要用户确认权限或完成任务后，终端窗口不会自动弹出。用户经常错过权限确认提示或不知道任务已完成。需要一套机制在关键时刻将终端窗口拉到前台并给出通知。

### 待集成的三个脚本

1. **`win32-foreground.ps1`** — PowerShell Win32 API 封装
   - 通过 P/Invoke 调用 user32.dll：SetForegroundWindow、BringWindowToTop、FlashWindow 等
   - 自动查找终端窗口（WindowsTerminal/ConEmu/cmder）
   - 临时禁用 Windows 前台锁定超时（SPI_SETFOREGROUNDLOCKTIMEOUT → 0）
   - 执行 Alt 键技巧 + COM AppActivate + SetForegroundWindow 组合拳
   - 短暂置顶（TOPMOST 1秒）+ 任务栏闪烁 + 声音提醒
   - 完成后恢复原始前台锁定超时

2. **`pre-tool-confirm.sh`** — 权限确认时唤醒终端
   - 触发时机：`PermissionRequest` 事件（hook 06）
   - 解析 stdin JSON 获取 tool_name 和 tool_input
   - 调用 `win32-foreground.ps1` 将终端拉到前台

3. **`task-complete-notify.sh`** — 任务完成时通知用户
   - 触发时机：`Stop` 事件（hook 16）
   - 调用 `win32-foreground.ps1` 将终端拉到前台
   - 发送 Windows Toast 通知（ToastNotificationManager → 回退 NotifyIcon BalloonTip）
   - 非阻塞，exit 0，不影响 Claude

### 集成方案

遵循现有 hooks 两层架构（`base.sh` 调度 + 业务脚本）：

```
.claude/hooks/
├── lib/
│   └── win32-foreground.ps1        # 新增：Win32 前台唤醒（PowerShell）
├── 06-permission-request/
│   ├── base.sh                     # 已有：source base.sh + 日志
│   └── win32-foreground.sh         # 新增：权限确认时调用前台唤醒
└── 16-stop/
    ├── base.sh                     # 已有：skill 清理 + 注册
    ├── skill-register.sh           # 已有
    └── task-complete-notify.sh     # 新增：任务完成通知
```

**需要修改的文件：**
- `hooks/06-permission-request/base.sh` — 在日志记录后增加调用 `win32-foreground.sh`
- `hooks/16-stop/base.sh` — 在 skill 清理后增加调用 `task-complete-notify.sh`
- `hooks/base.sh` — 无需修改（公共基础设施不变）

**脚本适配要求：**
- 业务脚本需 source `hooks/base.sh` 使用 `json_get()` 而非自行解析 JSON
- 日志使用 `log()` / `skill_log()` 而非自行写日志
- 平台检测：仅 Windows 执行，Linux/macOS 静默跳过（检查 `$OS_TYPE`）
- PowerShell 路径使用 `cygpath -w` 转换

### 约束

- 仅影响 Windows 平台（`$OS_TYPE = "windows"`），其他平台无副作用
- 不修改 `settings.json`（复用已有的 hook 06 和 hook 16 入口）
- `task-complete-notify.sh` 必须 exit 0，不阻塞 Claude 停止
- `pre-tool-confirm.sh` 必须 exit 0，不影响权限决策（仅做通知）

## 发布记录

- Issue #27: https://github.com/xiaoheiDTF/claude-hk/issues/27 (发布于 2026-05-23)
