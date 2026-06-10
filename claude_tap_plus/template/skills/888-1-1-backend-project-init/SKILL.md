---
name: 888-1-1-backend-project-init
description: 后端项目初始化：分析项目结构，生成 project.md 开发指南
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Glob
  - Grep
  - Write
---

# 后端项目初始化

> 分析项目后端代码结构，生成实用的 `project.md` 开发指南

---

## 一、产出目标

生成 `888-1-2-backend-modify/project.md`，包含以下三个章节：

1. **目录索引** — 东西在哪里找
2. **开发原则** — 项目用了什么开发方法论
3. **开发工作流** — 常见任务的标准步骤

不需要太细，够用就行。重点是让后续 `888-1-2-backend-modify` 知道**文件放哪、按什么顺序开发、遵循什么规则**。

---

## 二、生成流程

### Step 1：确认后端根目录

问用户后端代码的根目录位置。如果只有一个明确的目录，直接确认。

### Step 2：生成目录索引

扫描后端根目录，列出每个目录的职责。格式参考：

```
| 要找的东西 | 路径 | 说明 |
|-----------|------|------|
| 入口文件 | cmd/xxx/ | CLI 入口和子命令 |
| API 路由 | internal/backend/api/ | HTTP handler + router |
| 业务逻辑 | internal/backend/service/ | Service 层 |
| 数据存储 | internal/backend/store/ | 持久化层 |
| 实体定义 | internal/backend/domain/ | 领域模型 |
| 测试 | tests/ | 各层级测试 |
```

只需要读目录结构和 1-2 个代表性文件确认用途即可，**不需要逐文件分析**。

### Step 3：识别开发原则

从代码结构中识别项目采用了哪些开发原则。常见原则：

| 原则 | 识别特征 |
|------|---------|
| **TDD 测试驱动** | 有 tests/ 目录，测试文件命名有规律，先写测试再写实现 |
| **BDD 行为驱动** | 有 .feature 文件或 Given-When-Then 风格的测试 |
| **DDD 领域驱动** | 有 domain/ 层，实体和值对象分离，聚合根 |
| **分层架构** | 有 handler → service → store 分层，各层职责清晰 |
| **接口隔离** | store 层定义 interface，实现分离 |
| **依赖注入** | 通过构造函数注入依赖 |

只列出项目中**实际使用的**，不要猜测。

### Step 4：生成开发工作流

根据识别到的目录结构和开发原则，为以下常见任务生成标准步骤：

- **新增 API 接口**（从 domain → store → service → handler → router）
- **修改现有接口**
- **新增数据表/实体**
- **编写测试**

每个步骤只需要说明：
1. 在哪个目录创建/修改什么文件
2. 文件命名规则
3. 必须遵循的约定（如：handler 不直接调 store）

### Step 5：补充构建命令

列出常用的构建、运行、测试命令。

### Step 6：写入 project.md

将以上内容写入 `888-1-2-backend-modify/project.md`。

---

## 三、前置与后续

| 关系 | Skill |
|------|-------|
| **前置** | 无 |
| **后续** | `/888-1-2-backend-modify` 使用生成的 `project.md` |
