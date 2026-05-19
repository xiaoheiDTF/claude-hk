# Claude-Code / Advanced / Config-Snippets

> 来源: claudecn.com

# 配置片段库：从成熟实践提炼可复制模板

把一套成熟的 Claude Code 配置里最有用、最常迁移的部分**提炼成可复制片段**，方便你快速落地到：

- ~/.claude/（个人全局）
- .claude/（项目/团队共享）
所有示例片段都来自公开配置文件（见文末“参考”），你需要按自己的团队约束与系统环境做裁剪。

## 1) 最小目录骨架（建议从这里开始）

把“底线 + 流程 + 角色 + 自动化”拆成可审查的文件：

```text
your-repo/
├─ CLAUDE.md
└─ .claude/
   ├─ rules/
   ├─ commands/
   ├─ agents/
   ├─ hooks/
   └─ settings.json
```

更完整的落地路径见：

- 配置工程化
- 团队 Starter Kit
## 2) CLAUDE.md（项目级）：把“事实与底线”写成入口

下面是项目级 `CLAUDE.md` 的典型结构（节选）：

```markdown
## Critical Rules

### 1. Code Organization

- Many small files over few large files
- High cohesion, low coupling
- 200-400 lines typical, 800 max per file

### 3. Testing

- TDD: Write tests first
- 80% minimum coverage

### 4. Security

- No hardcoded secrets
- Validate all user inputs
```

你可以把它翻译/转成团队常用表述，但建议保留“可执行”约束（例如测试命令、提交规范、禁区目录）。

## 3) ~/.claude/CLAUDE.md（用户级）：全局原则 + 模块化规则入口

用户级配置更适合放“跨项目通用”的理念与入口（节选）：

```markdown
## Core Philosophy

**Key Principles:**
1. **Agent-First**: Delegate to specialized agents for complex work
2. **Parallel Execution**: Use Task tool with multiple agents when possible
3. **Plan Before Execute**: Use Plan Mode for structured approach
4. **Test-Driven**: Write tests before implementation
5. **Security-First**: Never compromise on security
```

如果你希望把“全局偏好”与“项目约束”分开管理，可以配合 [上下文管理](https://claudecn.com/docs/claude-code/workflows/context-management/) 一起使用。

## 4) /commands：把高频流程变成强约束入口

与其每次口头提醒“先规划再改”，不如把流程固化成命令模板（节选）：

```markdown
---
description: Restate requirements, assess risks, and create step-by-step implementation plan. WAIT for user CONFIRM before touching any code.
---
```

相关落地建议见：

- 自定义命令
- 团队质量门禁
## 5) hooks/hooks.json：把护栏自动化（提醒/校验/阻断）

下面这些 Hook 片段可以直接迁移（原样节选）：

不同仓库对 Hook 的“文件组织方式”不同：有的把 Hook 独立成 `hooks/hooks.json`，有的直接放在 `.claude/settings.json` 的 `hooks` 字段下。**片段内容是等价的**，你只需要按自己的工程组织方式改一下外壳结构。

### 5.1 长命令提醒：建议在 tmux 里跑（提醒型）

```json
{
  "matcher": "tool == \"Bash\" && tool_input.command matches \"(npm (install|test)|pnpm (install|test)|yarn (install|test)|bun (install|test)|cargo build|make|docker|pytest|vitest|playwright)\"",
  "hooks": [
    {
      "type": "command",
      "command": "#!/bin/bash\ninput=$(cat)\nif [ -z \"$TMUX\" ]; then\n  echo '[Hook] Consider running in tmux for session persistence' >&2\n  echo '[Hook] tmux new -s dev  |  tmux attach -t dev' >&2\nfi\necho \"$input\""
    }
  ],
  "description": "Reminder to use tmux for long-running commands"
}
```

### 5.2 git push 前停一下（交互型）

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

### 5.3 console.log 审计：编辑后提醒 + 会话结束扫描

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

### 5.4 禁止在 main 分支直接编辑（阻断型）
用途：把“先建分支再改动”变成硬规则，降低误操作概率（示例节选）。

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

### 5.5 修改测试文件后自动跑相关测试（自动化，但谨慎）
用途：把“写完测试就要跑”变成默认行为（示例节选）。

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

落地建议：先在单仓库试点；如果你的测试体系较慢/不稳定，可以改为“提醒型”，或只在 CI 中跑。

### 5.6 package.json 变更后自动安装依赖（自动化，但谨慎）

用途：降低“改了依赖但忘了装”的概率（示例节选）。

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

落地建议：团队如果统一用 `pnpm`/`yarn`，请替换命令；也可以只在“锁文件变更”时触发。

### 5.7 UserPromptSubmit：自动提示应该启用哪些 Skills（提醒型）

用途：把“应该用哪些 Skills”从口头约定变成自动提示（示例节选）。

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/skill-eval.sh",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
```

更多“可迁移配方”见：[Hooks 配方](https://claudecn.com/docs/claude-code/advanced/hooks-recipes/)。

## 6) mcp-servers.json：从占位符清单里挑你需要的那几个

示例配置强调两点：**用占位符**、**控制启用数量**（节选）：

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "YOUR_GITHUB_PAT_HERE"
      },
      "description": "GitHub operations - PRs, issues, repos"
    }
  }
}
```

配置与作用域细节见：[MCP 服务器](https://claudecn.com/docs/claude-code/advanced/mcp-servers/)。

另一个“团队共享 `.mcp.json` 模板”的例子（示例节选）：

```json
{
  "mcpServers": {
    "jira": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-jira"],
      "env": {
        "JIRA_HOST": "${JIRA_HOST}",
        "JIRA_EMAIL": "${JIRA_EMAIL}",
        "JIRA_API_TOKEN": "${JIRA_API_TOKEN}"
      }
    },
    "github": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-github"],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    }
  }
}
```

## 参考
无
