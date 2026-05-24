# 模块 2：Session 索引与消息记录

> 阶段：M2 | 依赖：无（可独立运行）

## 目标

为 Claude Code session 建立本地只读索引，支持按机器、项目、分支快速查找。代理模式下同时记录一份完整的 API 消息。支持从其他机器导入 traces。

**核心原则：不动 Claude Code 的原始存储。**

## 功能

### 功能 1：Session 只读索引

只读扫描 Claude Code 原始 session，不主动修改 `~/.claude/`。

1. 扫描本机 Claude session 文件。
2. 解析 session_id、时间、cwd、slug、project、branch。
3. 写入本地索引库。
4. 重复扫描必须幂等。
5. 无法解析的文件进入 skipped 列表并记录原因。

### 功能 2：Trace 与 Session 关联

代理产生的 trace 需要能和 session 建立关联。

1. trace 中存在 session_id 时直接关联。
2. trace 中没有 session_id 时，使用 cwd、时间窗口、project 做弱关联。
3. 查询时要区分强关联和弱关联。
4. 关联失败不影响 trace 保存。

### 功能 3：外部导入

支持从其他机器导入 trace/session 数据。

1. 支持完整目录导入。
2. 支持单个 JSONL 文件导入。
3. 支持指定来源的机器、项目、系统、slug。
4. 同 session_id 默认跳过，强制模式才覆盖。
5. 导入数据必须标记来源为 import。

### 功能 4：Session 查询

提供多维查询：

1. 支持按 machine、project、os、slug、branch、session、source 查询。
2. 支持表格输出和 JSON 输出。
3. 查询结果必须显示原始路径、trace 路径、来源机器。
4. 不存在结果时给出可用筛选建议。
