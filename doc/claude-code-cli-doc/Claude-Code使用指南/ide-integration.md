# Claude-Code / Ide-Integration

> 来源: claudecn.com

# IDE 集成

Claude Code 可以与主流 IDE 深度集成，提供图形化界面和更便捷的操作体验。

## 支持的 IDE
[VS Code](vscode/)
[JetBrains](jetbrains/)
[Chrome DevTools](chrome/)

## 选择指南

| IDE | 特点 | 适用场景 |
| --- | --- | --- |
| **VS Code** | 官方扩展，功能最完整 | 日常开发首选 |
| **JetBrains** | IntelliJ IDEA、WebStorm 等 | Java/Kotlin/前端开发 |
| **Chrome** | DevTools 集成 | 前端调试 |

## 与 CLI 的关系

- IDE 扩展提供图形界面，CLI 提供命令行界面
- 两者共享对话历史和配置
- 部分高级功能（如 MCP 配置）需要通过 CLI 完成
- 可以在 IDE 内置终端中运行 claude 使用 CLI
