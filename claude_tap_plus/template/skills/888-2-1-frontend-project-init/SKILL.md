---
name: 888-2-1-frontend-project-init
description: 前端项目初始化：分析项目结构，生成 project.md 开发指南
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Glob
  - Grep
  - Write
---

# 前端项目初始化

> 分析项目前端代码结构，生成实用的 `project.md` 开发指南

---

## 一、产出目标

生成 `888-2-2-frontend-modify/project.md`，包含以下三个章节：

1. **目录索引** — 东西在哪里找
2. **开发原则** — 项目用了什么开发方法论
3. **开发工作流** — 常见任务的标准步骤

不需要太细，够用就行。重点是让后续 `888-2-2-frontend-modify` 知道**文件放哪、按什么顺序开发、遵循什么规则**。

---

## 二、生成流程

### Step 1：确认前端根目录

问用户前端代码的根目录位置。如果只有一个明确的目录，直接确认。

### Step 2：生成目录索引

扫描前端根目录，列出每个目录的职责。格式参考：

```
| 要找的东西 | 路径 | 说明 |
|-----------|------|------|
| 页面 | src/pages/ | 按业务模块分目录 |
| 组件 | src/components/ | 通用/业务组件 |
| 状态管理 | src/store/ | 全局状态 |
| API | src/api/ | 接口封装 |
| 类型 | src/types/ | TypeScript 类型定义 |
| 样式 | src/styles/ | 全局样式/Token |
| 测试 | tests/ 或 __tests__/ | 测试文件 |
```

只需要读目录结构和 1-2 个代表性文件确认用途即可，**不需要逐文件分析**。

### Step 3：识别开发原则

从代码结构中识别项目采用了哪些开发原则。常见原则：

| 原则 | 识别特征 |
|------|---------|
| **CDD 组件驱动** | 有 atoms/molecules/organisms 分层，组件有 stories 文件 |
| **BDD 行为驱动** | 有 .feature 文件，Gherkin 场景描述 |
| **Atomic Design 原子设计** | atoms → molecules → organisms 分层 |
| **Design Tokens 设计令牌** | 有 tokens/ 目录，CSS 变量统一管理颜色/间距/字号 |
| **Storybook-Driven** | 有 .storybook/ 目录，组件配套 .stories.ts |
| **Composition API** | Vue 3 setup 语法，composable 函数 |
| **响应式状态管理** | Pinia/Vuex/Zustand 等状态库 |

只列出项目中**实际使用的**，不要猜测。

### Step 4：生成开发工作流

根据识别到的目录结构和开发原则，为以下常见任务生成标准步骤：

- **新增组件**（在哪个层级创建、需要哪些配套文件）
- **新增页面**（页面文件 + 路由注册 + 样式）
- **新增 API 接口**（接口封装 + 类型定义）
- **新增状态管理**（store/composable）

每个步骤只需要说明：
1. 在哪个目录创建什么文件
2. 文件命名规则
3. 必须遵循的约定（如：样式用 Token 不要写 hex）

### Step 5：补充构建命令

列出常用的开发、构建、测试命令。

### Step 6：写入 project.md

将以上内容写入 `888-2-2-frontend-modify/project.md`。

---

## 三、前置与后续

| 关系 | Skill |
|------|-------|
| **前置** | 无 |
| **后续** | `/888-2-2-frontend-modify` 使用生成的 `project.md` |
