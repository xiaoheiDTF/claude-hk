# Hooks + Skills 系统流程图集

本文档包含 Hooks + Skills 系统的所有 Mermaid 图表。配合 [hooks-skills-architecture.md](hooks-skills-architecture.md) 阅读效果更佳。

---

## 1. Hooks 系统总览图

展示 Claude Code 的 29 个 Hook 事件与共享基础设施的关系。

```mermaid
graph TB
    CC["Claude Code"]

    subgraph 共享基础设施
        BASE["hooks/base.sh<br/>JSON 解析 · 日志 · 分发"]
        PLAT["hooks/platform.sh<br/>平台检测 · Python 路径"]
        JSON["hooks/json_get.py<br/>Python JSON 解析"]
    end

    subgraph 有业务逻辑的 Hook
        H01["01-session-start<br/>初始化 · 巡检 · 注册"]
        H03["03-user-prompt-submit<br/>Skill 匹配注入"]
        H05["05-pre-tool-use<br/>双层权限拦截"]
        H06["06-permission-request<br/>前台前置"]
        H16["16-stop<br/>清理 · 注册"]
        H29["29-session-end<br/>释放 · 注销"]
    end

    subgraph 纯日志转发的 Hook
        H02["02-setup"]
        H04["04-user-prompt-expansion"]
        H07["07-permission-denied"]
        H08["08-post-tool-use"]
        H09["09-post-tool-use-failure"]
        H10["10-post-tool-batch"]
        H11["11-notification"]
        H12["12-subagent-start"]
        H13["13-subagent-stop"]
        H14["14-task-created"]
        H15["15-task-completed"]
        H17["17-stop-failure"]
        H18["18-teammate-idle"]
        H19["19-instructions-loaded"]
        H20["20-config-change"]
        H21["21-cwd-changed"]
        H22["22-file-changed"]
        H23["23-worktree-create"]
        H24["24-worktree-remove"]
        H25["25-pre-compact"]
        H26["26-post-compact"]
        H27["27-elicitation"]
        H28["28-elicitation-result"]
    end

    subgraph Skill 共享模块
        ACTIVE["active.sh<br/>.active CRUD"]
        LOCK["lock.sh<br/>文件锁"]
        LOG["log.sh<br/>双写日志"]
        BACKEND["backend.sh<br/>后端 API"]
        ENFORCE["enforce_boundary.sh<br/>白名单拦截"]
    end

    CC --> H01 & H02 & H03 & H04 & H05 & H06
    CC --> H07 & H08 & H09 & H10 & H11 & H16
    CC --> H17 & H18 & H19 & H20 & H29

    H01 & H02 & H03 & H04 & H05 & H06 & H07 & H08
    H09 & H10 & H11 & H12 & H13 & H14 & H15 & H16
    H17 & H18 & H19 & H20 & H21 & H22 & H23 & H24
    H25 & H26 & H27 & H28 & H29 --> BASE
    BASE --> PLAT & JSON

    H05 --> ENFORCE
    H03 & H16 & H29 --> ACTIVE & BACKEND
    ACTIVE --> LOCK
```

**说明**：所有 Hook 都依赖 `base.sh` 共享基础设施。其中 6 个有独立业务逻辑，23 个仅做日志转发。Skill 共享模块被多个 Hook 调用。

---

## 2. Hook 事件处理通用流程

每个 Hook 事件的通用处理流程。

```mermaid
flowchart TD
    A["Claude Code 触发事件"] --> B["Hook 脚本启动<br/>stdin → HOOK_INPUT"]
    B --> C["source hooks/base.sh"]
    C --> D["json_get('.hook_event_name')<br/>解析事件名"]
    D --> E["log INFO 记录事件"]
    E --> F{有业务逻辑?}

    F -- 有业务逻辑 --> G["执行兄弟脚本<br/>或内联逻辑"]
    F -- 纯转发 --> H["dispatch_to_skill(event_num)"]

    G --> I["dispatch_to_skill(event_num)<br/>（可选）"]
    I --> J["hook_output(0/2, json)"]
    H --> J

    J --> K{"exit_code"}
    K -- "0" --> L["放行 / 继续处理"]
    K -- "2" --> M["阻止 / 拒绝操作"]
```

**说明**：这是所有 29 个 Hook 的通用执行路径。纯转发的 Hook 直接跳到 `dispatch_to_skill`，有业务逻辑的 Hook 先执行自定义逻辑再分发。

---

## 3. 01-session-start 初始化时序图

会话启动时 7 个步骤的执行时序。

```mermaid
sequenceDiagram
    participant CC as Claude Code
    participant H01 as 01-session-start
    participant Init as .claude/init.sh
    participant Backend as Go 后端

    CC->>H01: SessionStart 事件

    Note over H01: 步骤 1: 首次运行检查
    alt 首次运行（无 .initialized）
        H01->>Init: bash init.sh
        Init-->>H01: 初始化完成
    end

    Note over H01: 步骤 2: UTF-8 编码
    H01->>H01: ensure_utf8()

    Note over H01: 步骤 3: Python 检查
    H01->>H01: ensure_python_check()

    Note over H01: 步骤 4: 目录完整性
    H01->>H01: ensure_dirs()

    Note over H01: 步骤 5: Skill 巡检
    loop 每个 Skill
        H01->>H01: bash {skill}/scripts/init_check.sh
    end

    Note over H01: 步骤 6: Trace 初始化
    H01->>Backend: POST /api/proxy/trace-init
    Backend-->>H01: trace_path（可选）

    Note over H01: 步骤 7: 会话注册
    H01->>Backend: POST /api/session/register
    Backend-->>H01: 注册确认（可选）

    H01-->>CC: hook_output(0, '{}')
```

**说明**：步骤 1 是一次性的，步骤 2-7 每次会话都执行。步骤 6、7 在后端不可达时静默跳过。

---

## 4. 03-skill-inject 匹配时序图

用户输入 `/skill-name` 时的完整匹配流程。

```mermaid
sequenceDiagram
    participant User as 用户
    participant CC as Claude Code
    participant H03 as 03-user-prompt-submit
    participant SI as skill-inject.sh
    participant Reg as registry.conf
    participant Active as .active
    participant Skill as {skill}/scripts/<br/>03UserPromptSubmit.sh

    User->>CC: 输入 /003-4-issue-claim #42
    CC->>H03: UserPromptSubmit 事件
    H03->>SI: bash skill-inject.sh<br/>prompt + session_id

    Note over SI: 解析 prompt
    SI->>SI: 首字符是 / → 提取 skill 名
    SI->>SI: skill = "003-4-issue-claim"<br/>args = "#42"

    SI->>Reg: 查找 skill 名
    Reg-->>SI: 匹配成功

    SI->>Skill: bash 03UserPromptSubmit.sh
    Skill-->>SI: CONTEXT（注入的上下文）

    SI->>Active: active_add(session_id, skill_name)
    Active-->>SI: 写入成功

    SI-->>H03: skill_name|args + CONTEXT
    H03-->>CC: hook_output(0, CONTEXT)
    Note over CC: 上下文注入到 Claude 的输入
```

**说明**：首字符不是 `/` 则直接跳过（非 Skill 调用）。`03UserPromptSubmit.sh` 返回的 CONTEXT 会作为 hook 输出注入到 Claude 的上下文中。

---

## 5. 05-双层拦截决策流程图

PreToolUse 事件的双层拦截决策树。

```mermaid
flowchart TD
    A["05-pre-tool-use 触发"] --> B["解析 tool_name"]

    B --> C["A 层: enforce_boundary.sh"]
    C --> C1{读取 .active<br/>获取当前 skill}
    C1 -- 无活跃 skill --> PASS1["放行"]
    C1 -- 有活跃 skill --> C2{解析 SKILL.md<br/>allowed-tools}
    C2 -- 无白名单 --> PASS1
    C2 -- 有白名单 --> C3{tool_name<br/>在白名单中?}
    C3 -- 是 --> PASS1
    C3 -- 否 --> DENY1["hook_output(2, deny)<br/>工具不在白名单"]

    PASS1 --> D["B 层: dispatch_to_skill 05"]
    D --> D1{存在 05PreToolUse.sh?}
    D1 -- 不存在 --> PASS2["放行"]
    D1 -- 存在 --> D2["执行 05PreToolUse.sh"]
    D2 --> D3{脚本退出码}
    D3 -- "0" --> PASS2
    D3 -- "2" --> DENY2["hook_output(2, deny)<br/>Skill 级拒绝"]

    PASS2 --> E["hook_output(0, '{}')<br/>最终放行"]

    style DENY1 fill:#f66,color:#fff
    style DENY2 fill:#f66,color:#fff
    style E fill:#6f6,color:#fff
```

**说明**：A 层是粗粒度的工具名白名单检查，B 层是 Skill 自定义的细粒度检查。两层都放行才允许工具调用。

---

## 6. Skill 生命周期时序图

从 `/skill-name` 触发到 `16Stop` 清理的完整生命周期。

```mermaid
sequenceDiagram
    participant User as 用户
    participant CC as Claude Code
    participant H03 as 03-user-prompt-submit
    participant H05 as 05-pre-tool-use
    participant H08 as 08-post-tool-use
    participant H16 as 16-stop
    participant Active as .active

    rect rgb(230, 245, 255)
        Note over User,Active: 阶段 1: Skill 激活
        User->>CC: /skill-name [args]
        CC->>H03: UserPromptSubmit
        H03->>Active: active_add(session_id, skill_name)
        Note over Active: session_id|skill_name
        H03-->>CC: 注入上下文 CONTEXT
    end

    rect rgb(230, 255, 230)
        Note over User,Active: 阶段 2: Skill 活跃期间（多轮对话）
        loop Claude 每轮处理
            CC->>H05: PreToolUse（工具调用前）
            H05->>H05: A层: enforce_boundary<br/>B层: 05PreToolUse.sh
            H05-->>CC: allow / deny

            CC->>CC: 执行工具

            CC->>H08: PostToolUse（工具调用后）
            H08->>H08: 08PostToolUse.sh
            H08-->>CC: 记录 / 上下文
        end
    end

    rect rgb(255, 240, 230)
        Note over User,Active: 阶段 3: Skill 清理
        CC->>H16: Stop（响应完成）
        H16->>Active: active_get(session_id)
        Active-->>H16: skill_name
        H16->>H16: bash {skill}/scripts/16Stop.sh
        Note over Active: .active 条目被移除
        H16-->>CC: hook_output(0, '{}')
    end
```

**说明**：Skill 的生命周期分为三个阶段——激活（03 事件注入）、活跃（05-15 事件转发）、清理（16 事件移除）。

---

## 7. dispatch_to_skill 事件映射图

事件编号到 Skill 脚本名的完整映射关系。

```mermaid
graph LR
    subgraph Hook 事件
        E02["02 Setup"]
        E04["04 UserPromptExpansion"]
        E05["05 PreToolUse"]
        E06["06 PermissionRequest"]
        E07["07 PermissionDenied"]
        E08["08 PostToolUse"]
        E09["09 PostToolUseFailure"]
        E10["10 PostToolBatch"]
        E11["11 Notification"]
        E12["12 SubagentStart"]
        E13["13 SubagentStop"]
        E14["14 TaskCreated"]
        E15["15 TaskCompleted"]
        E17["17 StopFailure"]
        E18["18 TeammateIdle"]
        E19["19 InstructionsLoaded"]
        E20["20 ConfigChange"]
        E21["21 CwdChanged"]
        E22["22 FileChanged"]
        E23["23 WorktreeCreate"]
        E24["24 WorktreeRemove"]
        E25["25 PreCompact"]
        E26["26 PostCompact"]
        E27["27 Elicitation"]
        E28["28 ElicitationResult"]
        E29["29 SessionEnd"]
    end

    subgraph dispatch_to_skill 映射
        M["事件编号 → 脚本名"]
    end

    subgraph Skill 脚本
        S["skills/{name}/scripts/{ScriptName}.sh"]
    end

    E02 & E04 & E05 & E06 & E07 & E08 --> M
    E09 & E10 & E11 & E12 & E13 & E14 --> M
    E15 & E17 & E18 & E19 & E20 & E21 --> M
    E22 & E23 & E24 & E25 & E26 & E27 --> M
    E28 & E29 --> M
    M --> S
```

**说明**：`dispatch_to_skill()` 接收事件编号，映射到对应的脚本名（如 `05` → `05PreToolUse.sh`），然后在当前激活的 Skill 目录下查找并执行。事件 03 和 16 不走此映射（在各自的 `base.sh` 中直接处理）。

---

## 8. 003 Issue 工作流状态图

9 个 Issue 工作流 Skill 的状态转换。

```mermaid
stateDiagram-v2
    [*] --> Uninitialized

    Uninitialized --> LabelsCreated: /003-1-issue-init

    LabelsCreated --> IssueCreated: /003-2-issue
    IssueCreated --> IssueCreated: /003-3-issue-discuss

    IssueCreated --> Claimed: /003-4-issue-claim<br/>POST /api/issue/claim

    Claimed --> Fixing: /003-5-issue-fix<br/>创建分支

    Fixing --> ReadyForPR: /003-6-issue-done<br/>开发完成

    ReadyForPR --> PRCreated: /003-7-issue-pr<br/>创建 PR

    PRCreated --> Testing: /003-8-issue-test<br/>执行 Test Plan

    Testing --> Reviewing: /003-9-issue-review<br/>提交审核

    Reviewing --> Merged: merge<br/>gh pr merge
    Reviewing --> Rejected: reject<br/>添加 rejected Label

    Merged --> [*]
    Rejected --> Fixing: 重新修复
    Rejected --> [*]: 关闭

    note right of Claimed
        原子性领取
        后端 API 防冲突
    end note

    note right of Testing
        Test Plan 必须全部通过
        否则阻止合并
    end note
```

**说明**：每个状态转换对应一个 Skill 触发。Claimed 状态的领取是原子性的（通过后端 API 保证）。Testing → Reviewing 的转换要求 Test Plan 全部通过。

---

## 9. active.sh 并发安全模型图

文件锁 + .active 操作的并发安全机制。

```mermaid
sequenceDiagram
    participant A as Agent A
    participant B as Agent B
    participant Lock as .active.lock<br/>(mkdir 原子操作)
    participant File as .active 文件

    par Agent A 写入
        A->>Lock: mkdir .active.lock
        Note over Lock: 创建成功（获取锁）
        A->>File: 读取 → 修改 → 写入
        A->>Lock: rm -rf .active.lock（释放锁）
    and Agent B 写入（等待）
        B->>Lock: mkdir .active.lock
        Note over Lock: 创建失败（锁被占用）
        Loop 每 1 秒重试
            B->>Lock: mkdir .active.lock
            Note over Lock: 重试中...
        end
        Note over Lock: Agent A 释放后创建成功
        B->>File: 读取 → 修改 → 写入
        B->>Lock: rm -rf .active.lock（释放锁）
    end

    Note over Lock,File: 僵尸锁检测：锁超过 60 秒 → 自动清理
```

**说明**：使用 `mkdir` 的原子性实现跨平台文件锁。写操作通过 `lock_acquire` 获取锁后才执行，防止多 Agent 并发写入导致数据损坏。超时 60 秒的僵尸锁会被自动清理。

---

## 10. 日志双写架构图

统一日志 + 模块日志的双写路径。

```mermaid
flowchart TD
    subgraph 调用方
        H["hooks/base.sh<br/>log() 函数"]
        S["skills/log.sh<br/>skill_log() 函数"]
    end

    subgraph 日志文件
        UL["统一日志<br/>.claude/hooks/logs/YYYY-MM-DD.log"]
        ML["模块日志<br/>.claude/skills/log/{tag}/YYYY-MM-DD.log"]
    end

    subgraph 消费方
        R1["全局排查：查看所有事件"]
        R2["模块排查：查看特定 Skill"]
    end

    H -->|"仅写统一日志"| UL
    S -->|"写统一日志"| UL
    S -->|"写模块日志"| ML

    UL --> R1
    ML --> R2

    style UL fill:#e8f4fd
    style ML fill:#e8f8e8
```

**说明**：`hooks/base.sh` 的 `log()` 函数仅写入统一日志。`skills/log.sh` 的 `skill_log()` 函数同时写入统一日志和模块日志，实现双写。统一日志用于全局排查，模块日志用于特定 Skill 的问题定位。

---

> 相关文档：[Hooks + Skills 架构文档](hooks-skills-architecture.md)
