# 初始化流程

## 首次运行 (init.sh)

当 `.claude/.initialized` 文件不存在时，`01-session-start/base.sh` 触发 `init.sh`。

### 5 个步骤

```
1. 目录创建
   → ensure_dirs.sh 读取 dirs.conf
   → 创建所有声明的目录
   → 创建当天 doc/otherDoc/YYYY-MM-DD/

2. 平台检测
   → platform.sh 检测 OS (linux/macos/windows)
   → 解析 Python 路径

3. Python 配置
   → ensure_python.sh 多策略保障：
     a. 嵌入版 Python (.claude/localLanguage/python/)
     b. 系统 python3 / python
     c. 自动下载（Windows 嵌入版，5 次尝试上限）
   → 状态写入 .python-state

4. UTF-8 配置
   → 写入 ~/.bashrc 的标记块
   → 幂等：标记块不重复写入

5. 标记文件
   → 写入 .claude/.initialized
   → JSON: {os, python_path, utf8, timestamp}
   → 后续启动跳过 init.sh
```

## 每次会话 (01-session-start/base.sh)

每个会话开始时执行环境巡检：

```
1. UTF-8 检查
   → 确保 locale 设置正确

2. Python 检查
   → 确认 Python 可用
   → 不可用时尝试重新配置

3. 目录完整性
   → ensure_dirs.sh check_dirs()
   → 报告缺失的目录

4. Skill 初始化检查
   → 遍历每个 Skill 的 scripts/init_check.sh
   → 健康检查（如 git/gh 是否可用）
```

## 幂等性设计

### 标记文件 (.initialized)

```json
{"os":"windows","python":"C:/path/to/python.exe","utf8":true,"time":"2026-05-22T10:00:00"}
```

存在则跳过 init.sh，不存在才执行。

### .bashrc 标记块

```bash
# >>> claude-hk-utf8 >>>
export LANG=en_US.UTF-8
# <<< claude-hk-utf8 <<<
```

通过标记块注释检测是否已写入，防止重复。

### 目录创建

`ensure_dirs.sh` 只创建不存在的目录，已存在则跳过。

## 平台适配

### Windows

- 检测：`uname -s` 包含 `MINGW`/`MSYS`/`CYGWIN`
- Python：优先嵌入版 `.claude/localLanguage/python/python.exe`
- 自动下载：从 python.org 下载 embeddable package 解压到嵌入版路径
- 路径：使用 `C:/` 风格路径

### macOS

- 检测：`uname -s` = `Darwin`
- Python：系统 `python3`
- UTF-8：`export LANG=en_US.UTF-8`

### Linux

- 检测：`uname -s` = `Linux`
- Python：系统 `python3` / `python`
- UTF-8：`export LANG=en_US.UTF-8`
