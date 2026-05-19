# Hooks 跨平台适配问题

## 问题描述

当前 hooks 脚本仅针对 Windows (Git Bash) 环境开发和测试，在 Linux 和 macOS 上可能存在兼容性问题。

## 已知问题

- `cat` 读取 stdin 在不同 shell (bash/zsh/sh) 下行为差异
- `$CLAUDE_PROJECT_DIR` 路径分隔符：Windows `\` vs Unix `/`
- Python 嵌入版仅下载了 Windows amd64 版本，Linux/macOS 需另行处理
- `date +%Y-%m-%d` 在 macOS BSD 工具和 Linux GNU 工具间格式参数可能不同
- `mkdir -p` 权限问题在不同系统上的表现

## 需要解决

1. 检测操作系统，自动下载对应平台的 Python 嵌入版
2. 路径处理统一使用 `/` 分隔符
3. 验证所有 base.sh 命令在 bash/zsh 下均可执行
4. Linux/macOS 下 Python 回退策略（系统 python3 通常可用）

## 影响范围

- `.claude/hooks/base.sh`
- `.claude/hooks/json_get.py`
- `.claude/settings.json` 中的 `command` 字段（shell 类型）
