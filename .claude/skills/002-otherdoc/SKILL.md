---
name: 002-otherdoc
description: 将用户需要记录的内容以 Markdown 文件存储到 doc/otherDoc 目录，按日期归档
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Write
  - Edit
  - Glob
  - Grep
---

# OtherDoc Skill

将用户需要记录的内容写入 Markdown 文件，按日期归档，支持单文件扁平存储或多层级工程化目录结构。

## 目录

- 存储根目录：`$CLAUDE_PROJECT_DIR/doc/otherDoc/`
- 按日期建子目录：`YYYY-MM-DD/`

---

## 规则（三条，不要搞复杂）

### 情况1：本次会话只生成 1 个 md 文件

直接放日期目录下：

```
doc/otherDoc/2026-05-24/xxx.md
```

### 情况2：本次会话生成多个 md 文件，但**无法按模块分类**

在日期目录下创建一个**英文描述文件夹**，md 文件直接放在里面：

```
doc/otherDoc/2026-05-24/英文描述/xxx.md
doc/otherDoc/2026-05-24/英文描述/yyy.md
```

**英文描述文件夹名**根据内容主题来定，例如：
- 开发相关 → `code-dev`
- 架构设计 → `architecture`
- 会议记录 → `meeting-notes`
- API 设计 → `api-design`

### 情况3：本次会话生成多个 md 文件，**可以按模块分类**

在日期目录下创建**英文描述文件夹**，其中：
- **只有一个 md 的模块** → 直接放在英文描述文件夹里
- **有多个 md 的模块** → 创建模块子目录，放进去

```
doc/otherDoc/2026-05-24/英文描述/
├── xxx.md              ← 单篇模块，直接放
├── yyy.md              ← 单篇模块，直接放
└── 模块A/              ← 多篇模块，建子目录
    ├── aaa.md
    └── bbb.md
```

---

## 命名规范

| 层级 | 规则 | 示例 |
|------|------|------|
| 日期目录 | `YYYY-MM-DD` | `2026-05-24` |
| 英文描述文件夹 | 小写、短横线连接、概括主题 | `dev-workflow` |
| 模块子目录 | 小写、短横线连接 | `testing` |
| md 文件名 | 小写、短横线连接 | `unit-test-guide.md` |

---

## 示例

### 示例1：1个文件

```
doc/otherDoc/2026-05-24/会议记录.md
```

### 示例2：3个文件，无法分类

```
doc/otherDoc/2026-05-24/meeting-notes/
├── 0524-morning-sync.md
├── 0524-tech-discussion.md
└── action-items.md
```

### 示例3：5个文件，可以分类（开发工作流主题）

```
doc/otherDoc/2026-05-24/dev-workflow/
├── 00-overview.md
├── linter-check.md           ← 单篇，直接放
├── git-commits.md            ← 单篇，直接放
└── testing/                  ← 多篇，建子目录
    ├── unit-test.md
    ├── integration-test.md
    └── coverage.md
```

---

## 执行规则

1. 每次写入前确认目录存在，不存在则创建
2. 按上述三条规则判断该放哪里
3. 文件开头写明创建时间、所属模块和简述
4. 如果要追加到已有文件，先读取再追加
