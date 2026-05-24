# Shell 脚本开发踩坑总结

> 本文档记录在开�?Claude Code hooks/skills 过程中遇到的所�?shell 脚本问题，包括根因分析和防范措施。供后续开发和修改脚本时参考�?

---

## 1. 路径格式不兼容：Git Bash vs Windows 原生程序

### 现象
嵌入�?Python �?`FileNotFoundError`，路径为 `/d/development/...`

### 根因
Git Bash 使用 Unix 风格路径（`/d/development/...`），�?Windows 原生程序（python.exe、node.exe 等）只能识别 Windows 路径（`D:\development\...`）。Bash 变量 `$VAR` 产生的是 Unix 路径，直接传给外部程序会失败�?

### 规则
- **凡是传给 Windows 原生程序的路径参数，必须经过 `cygpath -w` 转换**
- �?bash 操作（cat/mv/grep/mkdir）用 Unix 路径没问�?
- 转换函数示例�?
```bash
_to_native() {
  command -v cygpath &>/dev/null && cygpath -w "$1" || echo "$1"
}
# 使用
"$python" -c "open(r'$(_to_native "$FILE")')"  # 传给 Python
cat "$FILE"                                     # bash 自用，不�?
```

---

## 2. flock 不可�?+ 动态文件描述符不支�?

### 现象
`flock` 命令不存在，`exec {fd}>"$file"` 报语法错�?

### 根因
- `flock` �?Linux 特有工具，Git Bash for Windows 不包�?
- `exec {fd}>` �?Bash 4.1+ 的自�?fd 分配特性，MSYS2/Git Bash 支持不完�?

### 规则
- **文件锁使�?`mkdir` 原子操作**，跨平台通用�?
```bash
lock_acquire() {
  local lock_dir="$1" timeout="${2:-10}" elapsed=0
  while ! mkdir "$lock_dir" 2>/dev/null; do
    elapsed=$((elapsed + 1))
    [ "$elapsed" -ge "$timeout" ] && return 1
    sleep 1
  done
}
lock_release() {
  rm -rf "$_LOCK_DIR" 2>/dev/null
}
```
- **不要使用** `flock`、`exec {fd}>` �?Linux 特有特�?
- 需要外部命令时先检测：`command -v flock &>/dev/null`

---

## 3. 环境变量不传子进�?

### 现象
`CLAUDE_PROJECT_DIR="$(pwd)" bash script.sh` �?script.sh 中拿不到 `CLAUDE_PROJECT_DIR`

### 根因
`VAR=value` 是局部赋值，只在当前 shell 生效。`bash` 启动的子进程只继�?`export` 过的变量。`$(bash ...)` 中的 bash 是子进程的子进程�?

### 规则
- **需要子进程访问的变量必�?`export`**�?
```bash
# 错误：子进程拿不�?
CLAUDE_PROJECT_DIR="$(pwd)" bash script.sh

# 正确：子进程可以访问
export CLAUDE_PROJECT_DIR="$(pwd)"
bash script.sh
```
- 在测试脚本时�?`export`，实�?Claude Code 运行�?`CLAUDE_PROJECT_DIR` 已经是环境变�?

---

## 4. JSON 双重包装

### 现象
hook 输出�?JSON 嵌套了两�?`hookSpecificOutput`

### 根因
`03UserPromptSubmit.sh` 已经输出了完整的 `{"hookSpecificOutput":{...}}`，但调用方又把它转义后包进另一�?JSON。每层都以为自己是最终输出格式�?

### 规则
- **明确数据流的每一层职责，每个脚本只做自己的事**
- `03UserPromptSubmit.sh` �?输出完整 hook JSON
- `skill-inject.sh` �?透传，只附加元数�?
- `base.sh` �?直接 `hook_output` 传给 Claude Code
- **上游已完成的工作，下游不要重复做**

---

## 5. session_id 自�?vs 官方提供

### 现象
�?`sess_$$_$(date +%s)` 自�?session_id

### 根因
没有查看 hook 输入 JSON 的实际字段。所�?hook 事件都包�?`session_id` 字段，值是 Claude Code 生成�?UUID�?

### 规则
- **开�?hook 脚本前，先查日志了解输入 JSON 的完整字�?*
- 日志位置：`\.codex/hooks/logs/<日期>.log`，搜�?`Input:`
- 常用字段�?
  - 所有事件都有：`session_id`、`cwd`、`hook_event_name`、`transcript_path`
  - `UserPromptSubmit`：`prompt`、`permission_mode`
  - `PreToolUse`：`tool_name`、`tool_input`、`tool_use_id`
  - `Stop`：`last_assistant_message`、`stop_hook_active`

---

## 6. Python 硬依�?+ 一次性安装失败不重试

### 现象
init.sh 下载 Python 失败后不再重试，后续所有依�?Python 的功能静默失�?

### 根因
- `init.sh` 不管成功失败都写 `.initialized` 标记，然后永远不再执�?
- `session-start` 只检查不修复
- `json_get`、`active.sh` 都依�?Python/jq，没有降级方�?

### 规则
- **关键依赖的保障要�?一次性装�?改为"每次巡检+自动修复"**
- 使用独立模块 `ensure_python.sh`，每次会话调�?
- 多下载源 + 重试补偿 + 最大重试次�?
- **简单数据结构用�?bash 操作，不依赖 Python/jq**
  - `.active` �?`session_id|skill_name` 行格式，sed/grep 操作
  - `json_get` �?sed fallback 处理简单字段提�?

---

## 7. 脚本命名不统一导致调用困难

### 现象
skill �?`on_load.sh`、`on_stop.sh` 命名无法直接对应到哪�?hook 触发�?

### 规则
- **脚本命名 = `<hook编号><PascalCase事件�?.sh`**
- `03-user-prompt-submit` �?`03UserPromptSubmit.sh`
- `16-stop` �?`16Stop.sh`
- 好处：看到文件名就知道哪�?hook 触发，可以按编号动态拼接路�?

---

## 8. Python heredoc 引号嵌套问题

### 现象
Python heredoc 中的 `$VAR` �?bash 展开后，字符串内引号冲突导致语法错误

### 规则
- **Python 代码块使�?`-c "..."` + `r"..."` 原始字符串处理路�?*
```bash
# 正确
"$py" -c "
import json
with open(r\"$native_path\", \"r\") as f:
    d = json.load(f)
"
# 错误：heredoc �?$VAR 展开后引号冲�?
"$py" << PYEOF
d = json.load(open('$FILE'))  # $FILE 可能含反斜杠
PYEOF
```
- **更好的方案：避免�?bash 中写 Python JSON 操作**，用�?bash 处理简单数�?

---

## 9. `.bashrc` 重复写入

### 现象
UTF-8 配置被写�?`.bashrc` 两次（手动写一�?+ init.sh 又写一次）

### 规则
- **写入用户文件前必须检查是否已存在**
- 用标记块包裹，检查标记是否存在：
```bash
marker="# >>> claude-hk-utf8 >>>"
ender="# <<< claude-hk-utf8 <<<"
if grep -q "$marker" "$bashrc"; then
  # 有标�?�?跳过
elif grep -q "LANG=.*UTF-8" "$bashrc"; then
  # 有内容无标记 �?只补标记
else
  # 什么都没有 �?完整写入
fi
```

---

## 10. 跨平台适配检查清�?

每次写新脚本时逐项检查：

- [ ] 路径传给外部程序（Python/Node）→ `cygpath -w` 转换
- [ ] 文件�?�?�?`mkdir`，不�?`flock`
- [ ] 需要子进程访问的变�?�?`export`
- [ ] 使用 `exec {fd}>` �?改用固定方案
- [ ] 依赖 `jq`/`python` �?提供�?bash sed/awk fallback
- [ ] 写入用户配置文件 �?先检查再�?
- [ ] hook 输入字段 �?查日志确认，不猜
- [ ] `json_get` 解析字段 �?�?sed fallback
- [ ] 中文输出 �?确保 `chcp 65001` + `LANG=UTF-8`
- [ ] 测试时用 `export CLAUDE_PROJECT_DIR`，不是局部赋�?
