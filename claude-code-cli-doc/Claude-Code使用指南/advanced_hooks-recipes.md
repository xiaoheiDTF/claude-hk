# Claude-Code / Advanced / Hooks-Recipes

> 来源: claudecn.com

# Hooks 配方：把经验变成自动化护栏

Hooks 的价值在于：把你曾经踩过的坑（忘跑测试、dev server 没日志、误 push、console.log 遗留、文档碎片化）变成自动化护栏。下面挑选最通用的几类配方，并说明适用边界。

注意：Hooks 会直接影响你的工作流。建议先用“提醒型”，再逐步上“阻断型”。

Hook [ 脚本](#)通常会用到 Claude Code 注入的环境变量：

- $CLAUDE_TOOL_INPUT_FILE_PATH：本次操作涉及的文件路径
- $CLAUDE_TOOL_NAME：触发的工具名称
- $CLAUDE_PROJECT_DIR：项目根目录
此外，`PreToolUse` 的“阻断”通常依赖退出码：`exit 2` 表示阻断当前工具调用。

## 1) 长命令提醒：建议在 tmux 里跑（提醒型）

用途：避免长任务（install/test/build/docker/pytest 等）中断后丢日志/丢上下文。

示例（节选）：

```json
{
  "matcher": "tool == \"Bash\" && tool_input.command matches \"(npm (install|test)|pnpm (install|test)|yarn (install|test)|bun (install|test)|cargo build|make|docker|pytest|vitest|playwright)\"",
  "hooks": [
    {
      "type": "command",
      "command": "#!/bin/bash\ninput=$(cat)\nif [ -z \"$TMUX\" ]; then\n  echo '[Hook] Consider running in tmux for session persistence' >&2\n  echo '[Hook] tmux new -s dev  |  tmux attach -t dev' >&2\nfi\necho \"$input\""
    }
  ]
}
```

## 2) 强制 dev server 进 tmux（阻断型）
用途：把“开发服务器必须可追溯日志”变成硬规则，避免开在普通终端后丢失输出。

示例（原样节选）：

```json
{
  "matcher": "tool == \"Bash\" && tool_input.command matches \"(npm run dev|pnpm( run)? dev|yarn dev|bun run dev)\"",
  "hooks": [
    {
      "type": "command",
      "command": "#!/bin/bash\ninput=$(cat)\ncmd=$(echo \"$input\" | jq -r '.tool_input.command // \"\"')\n\n# Block dev servers that aren't run in tmux\necho '[Hook] BLOCKED: Dev server must run in tmux for log access' >&2\necho '[Hook] Use this command instead:' >&2\necho \"[Hook] tmux new-session -d -s dev 'npm run dev'\" >&2\necho '[Hook] Then: tmux attach -t dev' >&2\nexit 1"
    }
  ],
  "description": "Block dev servers outside tmux - ensures you can access logs"
}
```

落地建议：如果团队不习惯 tmux，可以先改成“提醒型”，等共识形成再升级为阻断。

## 3) git push 前强制停一下（提醒型/交互型）

用途：push 前强制 review 一次，降低“把不该推的推上去”的概率。

示例（原样节选）：

```json
{
  "matcher": "tool == \"Bash\" && tool_input.command matches \"git push\"",
  "hooks": [
    {
      "type": "command",
      "command": "#!/bin/bash\n# Open editor for review before pushing\necho '[Hook] Review changes before push...' >&2\n# Uncomment your preferred editor:\n# zed . 2>/dev/null\n# code . 2>/dev/null\n# cursor . 2>/dev/null\necho '[Hook] Press Enter to continue with push or Ctrl+C to abort...' >&2\nread -r"
    }
  ],
  "description": "Pause before git push to review changes"
}
```

## 4) 编辑后自动格式化/类型检查（自动化一致性）
用途：把“格式化 + 类型检查”从人肉习惯变成自动动作，减少 review 成本。

示例（原样节选）：

### 4.1 JS/TS 编辑后自动格式化（Prettier）

```json
{
  "matcher": "tool == \"Edit\" && tool_input.file_path matches \"\\\\.(ts|tsx|js|jsx)$\"",
  "hooks": [
    {
      "type": "command",
      "command": "#!/bin/bash\n# Auto-format with Prettier after editing JS/TS files\ninput=$(cat)\nfile_path=$(echo \"$input\" | jq -r '.tool_input.file_path // \"\"')\n\nif [ -n \"$file_path\" ] && [ -f \"$file_path\" ]; then\n  if command -v prettier >/dev/null 2>&1; then\n    prettier --write \"$file_path\" 2>&1 | head -5 >&2\n  fi\nfi\n\necho \"$input\""
    }
  ],
  "description": "Auto-format JS/TS files with Prettier after edits"
}
```

### 4.2 TS 编辑后增量类型检查（tsc）

```json
{
  "matcher": "tool == \"Edit\" && tool_input.file_path matches \"\\\\.(ts|tsx)$\"",
  "hooks": [
    {
      "type": "command",
      "command": "#!/bin/bash\n# Run TypeScript check after editing TS files\ninput=$(cat)\nfile_path=$(echo \"$input\" | jq -r '.tool_input.file_path // \"\"')\n\nif [ -n \"$file_path\" ] && [ -f \"$file_path\" ]; then\n  dir=$(dirname \"$file_path\")\n  project_root=\"$dir\"\n  while [ \"$project_root\" != \"/\" ] && [ ! -f \"$project_root/package.json\" ]; do\n    project_root=$(dirname \"$project_root\")\n  done\n  \n  if [ -f \"$project_root/tsconfig.json\" ]; then\n    cd \"$project_root\" && npx tsc --noEmit --pretty false 2>&1 | grep \"$file_path\" | head -10 >&2 || true\n  fi\nfi\n\necho \"$input\""
    }
  ],
  "description": "TypeScript check after editing .ts/.tsx files"
}
```

适用边界：这类 Hook 会引入额外耗时；建议只在“项目根目录可定位且工具存在”的前提下启用。

## 5) console.log 审计：编辑后提醒 + 会话结束扫描

用途：把 `console.log` 当作“必须清理”的临时调试痕迹，避免流入生产分支。

示例（节选）：

### 5.1 编辑后提醒（PostToolUse）

```json
{
  "matcher": "tool == \"Edit\" && tool_input.file_path matches \"\\\\.(ts|tsx|js|jsx)$\"",
  "hooks": [
    {
      "type": "command",
      "command": "#!/bin/bash\n# Warn about console.log in edited files\ninput=$(cat)\nfile_path=$(echo \"$input\" | jq -r '.tool_input.file_path // \"\"')\n\nif [ -n \"$file_path\" ] && [ -f \"$file_path\" ]; then\n  console_logs=$(grep -n \"console\\\\.log\" \"$file_path\" 2>/dev/null || true)\n  \n  if [ -n \"$console_logs\" ]; then\n    echo \"[Hook] WARNING: console.log found in $file_path\" >&2\n    echo \"$console_logs\" | head -5 >&2\n    echo \"[Hook] Remove console.log before committing\" >&2\n  fi\nfi\n\necho \"$input\""
    }
  ],
  "description": "Warn about console.log statements after edits"
}
```

### 5.2 会话结束扫描（Stop）

```json
{
  "matcher": "*",
  "hooks": [
    {
      "type": "command",
      "command": "#!/bin/bash\n# Final check for console.logs in modified files\ninput=$(cat)\n\nif git rev-parse --git-dir > /dev/null 2>&1; then\n  modified_files=$(git diff --name-only HEAD 2>/dev/null | grep -E '\\.(ts|tsx|js|jsx)

落地建议：如果你的项目允许日志，请改成检查 `logger.debug` 或限定目录（例如仅 `src/`）。

## 6) 防止“随机文档碎片化”（阻断型，谨慎）

用途：阻止随手创建很多 `.md/.txt`，把文档入口集中到 README/CLAUDE 等少数文件。

示例（节选）：

```json
{
  "matcher": "tool == \"Write\" && tool_input.file_path matches \"\\\\.(md|txt)$\" && !(tool_input.file_path matches \"README\\\\.md|CLAUDE\\\\.md|AGENTS\\\\.md|CONTRIBUTING\\\\.md\")",
  "hooks": [
    {
      "type": "command",
      "command": "#!/bin/bash\n# Block creation of unnecessary documentation files\ninput=$(cat)\nfile_path=$(echo \"$input\" | jq -r '.tool_input.file_path // \"\"')\n\nif [[ \"$file_path\" =~ \\.(md|txt)$ ]] && [[ ! \"$file_path\" =~ (README|CLAUDE|AGENTS|CONTRIBUTING)\\.md$ ]]; then\n  echo \"[Hook] BLOCKED: Unnecessary documentation file creation\" >&2\n  echo \"[Hook] File: $file_path\" >&2\n  echo \"[Hook] Use README.md for documentation instead\" >&2\n  exit 1\nfi\n\necho \"$input\""
    }
  ],
  "description": "Block creation of random .md files - keeps docs consolidated"
}
```

适用边界：对“文档驱动型仓库”（比如 Hugo 站点）不适用；更适合应用代码仓库。

## 7) 配合会话连续性与持续学习

Hooks 还常用于两个“长期收益很高”的地方：

- SessionStart/SessionEnd：写会话交接文件，方便第二天继续（见 会话连续性与战略压缩）
- Stop：在会话结束时触发一次复盘提示（见 持续学习）
## 8) 补充：团队协作护栏配方

这一组护栏更偏团队协作与可持续维护，建议先在小范围试点。

### 8.1 禁止在 main 分支直接编辑（PreToolUse，阻断型）

```json
{
  "matcher": "Edit|MultiEdit|Write",
  "hooks": [
    {
      "type": "command",
      "command": "[ \"$(git branch --show-current)\" != \"main\" ] || { echo '{\"block\": true, \"message\": \"Cannot edit files on main branch. Create a feature branch first.\"}' >&2; exit 2; }",
      "timeout": 5
    }
  ]
}
```

### 8.2 修改测试文件后自动跑相关测试（PostToolUse，自动化）

```json
{
  "matcher": "Edit|MultiEdit|Write",
  "hooks": [
    {
      "type": "command",
      "command": "if [[ \"$CLAUDE_TOOL_INPUT_FILE_PATH\" =~ \\\\.test\\\\.(js|jsx|ts|tsx)$ ]]; then\\n  npm test -- --findRelatedTests \"${CLAUDE_TOOL_INPUT_FILE_PATH}\" --passWithNoTests 2>&1 | tail -30\\nfi",
      "timeout": 90
    }
  ]
}
```

### 8.3 package.json 变更后自动安装依赖（PostToolUse，自动化）

```json
{
  "matcher": "Edit|MultiEdit|Write",
  "hooks": [
    {
      "type": "command",
      "command": "if [[ \"$CLAUDE_TOOL_INPUT_FILE_PATH\" =~ package\\\\.json$ ]]; then\\n  npm install >/dev/null 2>&1\\nfi",
      "timeout": 60
    }
  ]
}
```

这些自动化配方会引入额外环境依赖与耗时（例如 `npm install`、测试运行）。如果你的团队工具链不是 Node（或包管理器不同），请先改成“提醒型”，或仅在 CI 中启用。

## 参考
无

 || true)\n  \n  if [ -n \"$modified_files\" ]; then\n    has_console=false\n    while IFS= read -r file; do\n      if [ -f \"$file\" ]; then\n        if grep -q \"console\\.log\" \"$file\" 2>/dev/null; then\n          echo \"[Hook] WARNING: console.log found in $file\" >&2\n          has_console=true\n        fi\n      fi\n    done <<< \"$modified_files\"\n    \n    if [ \"$has_console\" = true ]; then\n      echo \"[Hook] Remove console.log statements before committing\" >&2\n    fi\n  fi\nfi\n\necho \"$input\""
    }
  ],
  "description": "Final audit for console.log in modified files before session ends"
}
```

落地建议：如果你的项目允许日志，请改成检查 `logger.debug` 或限定目录（例如仅 `src/`）。

## 6) 防止“随机文档碎片化”（阻断型，谨慎）

用途：阻止随手创建很多 `.md/.txt`，把文档入口集中到 README/CLAUDE 等少数文件。

示例（节选）：

%%CODE_BLOCK_7%%

适用边界：对“文档驱动型仓库”（比如 Hugo 站点）不适用；更适合应用代码仓库。

## 7) 配合会话连续性与持续学习

Hooks 还常用于两个“长期收益很高”的地方：

- SessionStart/SessionEnd：写会话交接文件，方便第二天继续（见 会话连续性与战略压缩）
- Stop：在会话结束时触发一次复盘提示（见 持续学习）
## 8) 补充：团队协作护栏配方

这一组护栏更偏团队协作与可持续维护，建议先在小范围试点。

### 8.1 禁止在 main 分支直接编辑（PreToolUse，阻断型）

%%CODE_BLOCK_8%%

### 8.2 修改测试文件后自动跑相关测试（PostToolUse，自动化）

%%CODE_BLOCK_9%%

### 8.3 package.json 变更后自动安装依赖（PostToolUse，自动化）

%%CODE_BLOCK_10%%

这些自动化配方会引入额外环境依赖与耗时（例如 `npm install`、测试运行）。如果你的团队工具链不是 Node（或包管理器不同），请先改成“提醒型”，或仅在 CI 中启用。

## 参考
无
