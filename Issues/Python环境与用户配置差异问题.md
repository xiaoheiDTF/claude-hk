# Python 环境与用户配置差异问题

## 问题描述

项目 hooks 依赖 Python 执行 JSON 解析（`json_get.py`），但不同用户的电脑环境差异大，导致解析全部失败。

## 已踩过的坑

1. **Windows Store 占位符**：`python3` 命令指向 `WindowsApps/python3`，不是真正的 Python，执行后只输出 "Python was not found" 提示安装
2. **PATH 顺序问题**：`command -v python3` 找到占位符后不再查找 `python`，而真正的 Python 是 `python`
3. **版本不一致**：用户可能有 Python 3.8 ~ 3.13 各种版本，`json_get.py` 需兼容
4. **未安装 Python**：部分用户电脑完全没有 Python

## 当前方案

- 下载 Python 3.13.9 嵌入版到 `.claude/python/python.exe`
- `_find_python` 优先使用嵌入版，回退到系统 Python 并验证可用性
- 仅使用标准库 `json` 和 `sys`，无第三方依赖

## 待解决

1. 嵌入版体积 ~20MB，需要初始化脚本自动下载
2. Linux/macOS 没有 embeddable package，需要另外的处理方式（系统 python3 / conda / pyenv）
3. 网络不通时下载失败，需要友好的错误提示和回退方案
