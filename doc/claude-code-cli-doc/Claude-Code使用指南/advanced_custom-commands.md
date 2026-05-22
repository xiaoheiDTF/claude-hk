# Claude-Code / Advanced / Custom-Commands

> 来源: claudecn.com

# 自定义命令

自定义命令（`/command`）可以把高频提示模板、流程清单固化成可复用入口。

**自定义命令已与 Skills 打通**：`.claude/commands/review.md` 与 `.claude/skills/review/SKILL.md` 都会提供 `/review`，效果等价；已有 `.claude/commands/` 文件仍可继续使用。
推荐优先用 Skills：它支持“支持文件目录”“调用控制（是否允许模型自动触发）”“在子代理中隔离执行”等能力。详见：[Skills（技能）](https://claudecn.com/docs/claude-code/advanced/skills/)。

## 命令文件位置
Claude Code 支持两类自定义命令：

| 级别 | 位置 | 调用方式 |
| --- | --- | --- |
| 项目级 | `.claude/commands/` | `/` 或 `/:` |
| 个人级 | `~/.claude/commands/` | `/` 或 `/:` |

如果命令文件位于子目录，会自动形成命名空间：
例如 `.claude/commands/frontend/component.md` 对应 `/frontend:component`。

## 命令语法

命令文件是普通的 Markdown 文件，使用 `$ARGUMENTS` 占位符传递用户输入的参数。

### Front matter（强烈建议）

如果你希望命令更“工程化”，建议在命令文件顶部加 YAML frontmatter（命令与 Skill 共用同一套字段）：

```yaml
---
description: 在命令列表中显示的说明
argument-hint: "[issue-number]"
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Bash(git:*), Bash(gh:*)
---
```

- description：用于命令列表展示，帮助团队知道“这个命令是干嘛的”
- argument-hint：自动补全时提示预期参数
- disable-model-invocation：设为 true 后，Claude 不会自动触发（只允许你手动执行 /command）
- allowed-tools：把命令的能力收敛到最小必要集合（尤其是包含写入/执行命令时）
### 变量

- $ARGUMENTS：整段参数（最常用）
- $1、$2、$3：按空格拆分的参数（适合固定结构参数）
## 把命令当“工作流入口”
很多团队会把 `/command` 只当成“提示词模板”。更高效的做法是把它当成**工作流入口**：在命令里约束执行顺序（先澄清 → 先规划 → 再验证 → 最后改代码），必要时让 Claude 使用子代理做隔离探索或审查。

例如，定义一个 `/plan` 类命令时，可以明确要求：

- 先复述需求并列出风险
- 给出分阶段计划
- 等用户确认后再开始改代码
如果你希望团队把这套方法论工程化落地，可以参考：[团队落地：把 Claude Code 配置当代码管理](https://claudecn.com/docs/claude-code/advanced/config-as-code/)。

### 基本结构

```markdown
请分析以下问题并提供解决方案：

$ARGUMENTS

要求：
1. 分析问题根因
2. 提供修复建议
3. 考虑边缘情况
```

## 实用示例
下面这些示例不是“提示词大全”，而是把一条高频流程固化成入口。一组可复用的团队命令通常会覆盖：

| 命令 | 用途（概括） |
| --- | --- |
| `/ticket` | 工单驱动：读需求→建分支→实现→更新工单→建 PR |
| `/pr-review` | 审查指定 PR，并用 `gh pr comment` 留评 |
| `/pr-summary` | 为当前分支生成 PR 描述模板 |
| `/docs-sync` | 检查“文档是否真的与代码不一致”（只报告实际问题） |
| `/code-quality` | 对目录跑 lint/typecheck + 人工清单复核 |

你可以先从 1~2 个命令开始（例如 `/plan` + `/code-review`），形成可重复闭环，再逐步扩展。

### fix-github-issue.md

```markdown
请修复以下 GitHub Issue：

$ARGUMENTS

步骤：
1. 分析 Issue 描述和相关代码
2. 确定问题根因
3. 实现修复方案
4. 添加必要的测试
5. 确保现有测试通过
```

### debug.md

```markdown
调试以下问题：

$ARGUMENTS

调试流程：
1. 重现问题
2. 检查相关日志和错误信息
3. 定位问题代码
4. 分析可能的原因
5. 提出修复方案
```

### review.md

```markdown
对以下代码进行 Code Review：

$ARGUMENTS

Review 关注点：
- 代码逻辑正确性
- 错误处理完整性
- 性能问题
- 安全漏洞
- 代码可读性和可维护性
```

### test.md

```markdown
为以下功能编写测试：

$ARGUMENTS

测试要求：
1. 覆盖正常流程
2. 覆盖边缘情况
3. 覆盖错误处理
4. 遵循项目现有测试风格
```

## 团队分享
将 `.claude/commands/` 目录提交到 Git 仓库，团队成员可共享相同的命令集：

```bash
git add .claude/commands/
git commit -m "Add Claude Code custom commands"
```

## 最佳实践

- 保持专注 - 每个命令只做一件事
- 清晰指令 - 提供明确的步骤和要求
- 善用参数 - 通过 $ARGUMENTS 保持命令灵活性
- 合理命名 - 使用描述性的文件名
- 文档说明 - 在命令开头注释说明用途
- 最小权限 - 通过 allowed-tools 限制命令能做什么，避免“命令变成万能脚本”
## 使用方式

```bash
# 调用项目级命令
/project:fix-github-issue #123

# 调用全局命令
/debug 应用启动时报 ECONNREFUSED 错误
```
