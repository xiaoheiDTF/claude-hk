# Claude-Code / Advanced

> 来源: claudecn.com

# 进阶指南

本章介绍 Claude Code 的高级功能，适合已掌握基础操作的用户进一步提升开发效率。

如果你还在“先把 Claude Code 用起来”的阶段，建议先回到 [学习路径](https://claudecn.com/docs/learning-paths/)，确认自己是不是已经适合进入这一章。

## 内容概览
[Agent Loop（工作原理）从零理解 tool loop / Todo / Subagents / Skills
](agent-loop/)[Agent Skills](skills/)
[Subagents 子代理](subagents/)
[Hooks 系统](hooks/)
[Hooks 配方把经验变成自动化护栏
](hooks-recipes/)[MCP 服务器](mcp-servers/)
[自定义命令](custom-commands/)
[模式库（Skills）把后端/前端经验沉淀
](skill-pattern-library/)[团队规则库用 Rules 固化底线
](rules-playbook/)[配置工程化](config-as-code/)
[团队 Starter Kit最小可用的共享配置
](starter-kit/)[配置片段库从成熟实践提炼可复制模板
](config-snippets/)[Claude Code SDK](sdk/)

## 适合人群

- 熟悉 Claude Code 基础操作
- 希望提升工作效率和自动化程度
- 需要定制化开发流程
- 想要与现有工具链集成
- 需要团队协作和共享配置
## 核心能力

| 功能 | 用途 | 调用方式 |
| --- | --- | --- |
| **Skills** | 教 Claude 专业知识 | Claude 自动匹配使用 |
| **Subagents** | 委派任务到独立上下文 | Claude 自动委托或显式调用 |
| **Hooks** | 事件触发自动化 | 特定事件自动触发 |
| **MCP** | 连接外部工具和数据 | Claude 按需调用 |
| **自定义命令** | 可复用的提示模板 | 输入 `/命令` 运行 |

## 学习路径

- Agent Loop（工作原理） - 用最小实现看清 Claude Code 这类 Agent 系统的核心控制流
- Agent Skills - 创建可复用的知识模块，教 Claude 团队规范和专业技能
- Subagents - 使用专用子代理处理特定任务，保持主对话整洁
- Hooks 系统 - 在工具执行前后自动运行脚本，实现自动化工作流
- MCP 服务器 - 连接数据库、API、外部工具，扩展 Claude 能力
- 自定义命令 - 创建团队共享的提示模板
- SDK 集成 - 以 编程方式使用 Claude Code
## 这章更适合什么时候读

- 你已经稳定使用 Claude Code 处理日常任务
- 你开始关心复用、自动化、团队协作和外部系统接入
- 你不只是想“用好 Claude”，而是想把一套能力沉淀为工程体系
