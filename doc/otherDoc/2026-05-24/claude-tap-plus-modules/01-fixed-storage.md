# 模块 1：固定本地存储

> 阶段：M1 | 依赖：无

## 目标

`claude-tap-plus` 的 trace、session、issue、sandbox 等所有数据必须写入固定位置，而不是写入当前工作目录。

## 功能

### 功能 1：统一存储根目录

为所有模块提供统一存储根目录。

1. 所有模块读取同一个存储根目录。
2. 不同机器、项目、系统、目录的数据必须有稳定隔离。
3. 存储目录不存在时自动创建。
4. 创建失败时输出明确错误，不回退到随机目录。

### 功能 2：项目身份识别

识别当前项目身份，用于关联 trace、session、issue、sandbox、usage。

识别优先级：

1. 用户显式指定的 `--project`。
2. Git remote repo name。
3. Git root 目录名。
4. 当前 cwd basename。

1. 同一个项目在不同模块中必须得到相同 project name。
2. 需要保存原始 cwd、git root、remote url、branch。
3. 无 Git 仓库时仍可运行 proxy trace，但项目名需要标记来源。

### 功能 3：数据目录分层

固定存储需要承载以下数据类型：

1. **traces** — 保存 API 交互记录。
2. **sessions** — 保存 session 索引或快照。
3. **db** — 保存本地状态库。
4. **sandboxes** — 保存 sandbox 元数据，不保存 worktree 源代码。
5. **logs** — 保存命令审计日志。

### 功能 4：存储健康检查

提供健康检查能力：

1. 存储根目录是否可读写。
2. 数据库目录是否可创建。
3. trace/session/log 子目录是否存在。
4. 当前项目身份是否可解析。
5. 是否存在历史目录结构需要迁移。
