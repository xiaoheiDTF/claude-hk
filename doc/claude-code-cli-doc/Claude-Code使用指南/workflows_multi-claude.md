# Claude-Code / Workflows / Multi-Claude

> 来源: claudecn.com

# 多 Claude 协作

通过运行多个 Claude 实例，实现更高效的并行开发和任务分工。

## 多 Claude 工作流模式

### 1. 一个编码，一个验证

启动两个终端窗口：

**终端 1 - 编码 Claude：**

```bash
claude "实现用户认证模块"
```

**终端 2 - 验证 Claude：**

```bash
claude "监控 src/auth/ 目录的变化，发现问题立即指出"
```

这种模式让一个 Claude 专注于实现，另一个专注于验证代码质量。

### 2. 多代码库 Checkout

为每个功能分支创建独立的工作目录：

```bash
# 创建多个 checkout
git clone myrepo feature-a-dir
git clone myrepo feature-b-dir
git clone myrepo feature-c-dir

# 在不同目录启动 Claude
cd feature-a-dir && claude "实现功能 A"
cd feature-b-dir && claude "实现功能 B"
cd feature-c-dir && claude "实现功能 C"
```

### 3. Git Worktrees（推荐）
使用 Git Worktrees 更高效地管理多分支开发：

```bash
# 添加 worktree
git worktree add ../project-feature-a feature-a
git worktree add ../project-feature-b feature-b
git worktree add ../project-feature-c feature-c

# 列出所有 worktree
git worktree list

# 在各 worktree 中启动 Claude
cd ../project-feature-a && claude
cd ../project-feature-b && claude

# 完成后移除 worktree
git worktree remove ../project-feature-a
```

### 4. Headless 模式扇出
使用 `-p` 标志实现并行任务处理：

```bash
# 并行处理多个文件
for file in src/*.ts; do
  claude -p "审查 $file 的代码质量" --output-format json &
done
wait

# 管道化处理
find . -name "*.test.ts" | xargs -P 4 -I {} \
  claude -p "分析测试覆盖率: {}"
```

## 详细工作流模式

### 探索 → 规划 → 编码 → 提交
分阶段处理复杂任务：

```bash
# 阶段 1：探索
claude "分析这个项目的架构，列出主要模块和依赖关系"

# 阶段 2：规划
claude -c "基于分析结果，制定重构计划"

# 阶段 3：编码
claude -c "按计划执行重构，从模块 A 开始"

# 阶段 4：提交
claude commit
```

### 测试驱动开发（TDD）

```bash
# Claude 1：编写测试
claude "为 UserService 编写单元测试，覆盖所有公共方法"
claude commit  # 提交测试

# Claude 2：实现代码
claude "实现 UserService，确保所有测试通过"

# 迭代直到测试通过
claude -c "修复失败的测试"
claude commit  # 提交实现
```

### 代码 → 截图 → 迭代
前端开发工作流，结合 Playwright MCP：

```bash
# 1. 实现 UI
claude "创建登录页面组件"

# 2. 截图验证
claude "使用 Playwright 打开 localhost:3000/login 并截图"

# 3. 根据截图迭代
claude -c "调整布局，按钮应该居中显示"
```

### 安全 YOLO 模式
在隔离环境中完全自动化执行：

仅在 Docker 容器或一次性环境中使用，避免在生产系统运行。

```bash
# Docker 容器中的自动化
docker run --rm -it \
  -v $(pwd):/workspace \
  -w /workspace \
  node:20 \
  npx claude \
    --dangerously-skip-permissions \
    -p "运行所有测试，修复失败的用例，然后提交"
```

### 代码库问答
快速了解陌生代码库：

```bash
# 启动专门用于问答的会话
claude -r "codebase-qa"

# 提问
> 这个项目使用什么框架？
> 数据库模型定义在哪里？
> 认证流程是怎样的？
> 如何添加新的 API 端点？
```

## 协作模式最佳实践

### 任务分工

| Claude 实例 | 职责 |
| --- | --- |
| Claude A | 核心功能实现 |
| Claude B | 测试编写 |
| Claude C | 文档更新 |
| Claude D | 代码审查 |

### 会话命名
使用有意义的会话名称便于管理：

```bash
# 创建命名会话
claude --session-id "feature-auth-impl"
claude --session-id "feature-auth-tests"
claude --session-id "feature-auth-docs"

# 恢复特定会话
claude -r "feature-auth-impl"
```

### 结果汇总
使用 JSON 输出合并多个 Claude 的结果：

```bash
# 收集多个 Claude 的分析结果
claude -p "分析 src/api/" --output-format json > api-analysis.json
claude -p "分析 src/models/" --output-format json > models-analysis.json
claude -p "分析 src/utils/" --output-format json > utils-analysis.json

# 汇总
claude -p "根据这些 JSON 文件生成完整的代码质量报告"
```

## 并行开发注意事项

- 避免文件冲突：确保不同 Claude 实例操作不同的文件
- 同步点：在关键节点进行 Git 合并和冲突解决
- 资源管理：注意 API 调用配额和系统资源
- 日志追踪：使用 --verbose 记录详细操作日志
