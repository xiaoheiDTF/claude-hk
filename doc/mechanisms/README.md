# 功能机制

本目录包含所有功能的实现原理文档，回答"怎么实现的？设计原理是什么？"。

## 建议阅读顺序

1. [系统架构](architecture.md) — 理解整体三层设计
2. [目录结构](project-structure.md) — 了解每个文件/目录的职责
3. [Hooks 管道](hooks-pipeline.md) — 事件注册、调度、JSON 解析
4. [Skill 系统](skill-system.md) — 上下文注入、生命周期、并发安全
5. [初始化流程](initialization.md) — init.sh 幂等初始化
6. [设计亮点](design-highlights.md) — 技术决策与设计原则

## 机制关系总览

```
settings.json（声明入口）
       │
       ▼
Hooks 管道（29 个事件调度）
       │
       ├── 01-session-start → 初始化流程
       ├── 03-user-prompt-submit → Skill 激活
       ├── 05-pre-tool-use → 工具边界检查
       └── 16-stop → Skill 清理 + 自动注册
               │
               ▼
       Skills 系统（13 个内置技能）
```
