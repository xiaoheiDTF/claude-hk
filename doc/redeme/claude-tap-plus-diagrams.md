# claude_tap_plus 流程图集

本文档包含 claude_tap_plus 项目的所有 Mermaid 图表。配合 [claude-tap-plus-architecture.md](claude-tap-plus-architecture.md) 阅读效果更佳。

---

## 1. 系统总体架构图

展示三大子系统：代理模式、后端服务、会话管理的关系。

```mermaid
graph TB
    subgraph CLI 入口
        MAIN["cmd/claude-tap/main.go<br/>子命令分发"]
    end

    subgraph 代理模式
        CONFIG["config/<br/>配置解析"]
        PROXY["proxy/<br/>HTTP 反向代理"]
        SSE["sse/<br/>SSE 流重组"]
        TRACE["trace/<br/>JSONL 追踪"]
        USAGE["usage/<br/>Token 归一化"]
    end

    subgraph 后端服务
        API["backend/api/<br/>HTTP 处理器"]
        SVC["backend/service/<br/>业务逻辑"]
        STORE["backend/store/<br/>SQLite 持久化"]
        DOMAIN["backend/domain/<br/>实体定义"]
    end

    subgraph 会话管理
        SESSION["session/<br/>Push/Pull/Status"]
    end

    subgraph 基础设施
        LOGGER["logger/<br/>统一日志"]
    end

    MAIN -->|"默认子命令"| PROXY
    MAIN -->|"backend 子命令"| API
    MAIN -->|"session-push/pull/status"| SESSION

    PROXY --> CONFIG & SSE & TRACE & LOGGER
    SSE --> USAGE & LOGGER
    TRACE --> USAGE & LOGGER
    CONFIG --> LOGGER

    API --> SVC & LOGGER
    SVC --> STORE & LOGGER
    STORE --> DOMAIN & LOGGER

    SESSION --> TRACE & LOGGER

    style MAIN fill:#4a90d9,color:#fff
    style LOGGER fill:#888,color:#fff
```

**说明**：三大子系统共用 `logger/` 基础设施。代理模式包含完整的 API 流量拦截管线（config → proxy → sse → trace → usage），后端服务采用三层架构（api → service → store，11 个 Service + 9 个 Handler + 15 个端点），会话管理独立运行。

---

## 2. 代理模式数据流图

从 CLI 启动到 Trace 文件写入的完整数据流。

```mermaid
flowchart TD
    A["CLI 启动<br/>main.go"] --> B["配置解析<br/>ResolveTargetConfig()"]
    B --> B1["model 优先级链<br/>profile.model > ~/.claude.json model > 空"]
    B --> C["创建代理<br/>NewReverseProxy(target, traceDir)"]
    C --> C1["SetModel(resolved.Model)<br/>设置 model 改写"]
    C1 --> C2["loadFallbackConfig()<br/>读取 ~/.claude/settings.json"]
    C2 --> C3["SetFallbackConfig(fallback)<br/>设置兜底配置"]
    C3 --> D["启动本地代理<br/>proxy.Start(127.0.0.1, port)"]
    D --> E["启动 Claude Code 子进程<br/>设置 ANTHROPIC_BASE_URL=proxy"]

    E --> F["Claude Code 发送 API 请求"]
    F --> G["代理拦截请求<br/>serveHTTP()"]

    G --> G1["读取请求体 → 解析 JSON"]
    G1 --> G2["rewriteModel()<br/>强制改写 model 字段"]
    G2 --> G3{"upstreamAvailable?"}
    G3 -->|"true"| G4["用 profile target 转发"]
    G3 -->|"false"| G5["切换 fallbackConfig<br/>base_url + model + auth 全量替换"]

    G4 --> H{请求类型?}
    G5 --> H

    H -->|"/_internal/trace-init"| I["handleInternal()<br/>创建 TraceWriter"]
    I --> J["SessionID + ProjectSlug<br/>写入 proxy.json"]

    H -->|"Streaming 响应"| K["handleStreaming()"]
    K --> L["转发请求到上游 API"]
    L --> M["接收 SSE 字节流"]
    M --> N["SSEReassembler.FeedBytes()"]
    N --> O["accumulate() 累积片段"]
    O --> P["Reconstruct() 重组响应"]
    P --> Q["NormalizeUsage() 归一化 Token"]
    Q --> R["TraceWriter.Write()<br/>写入 JSONL"]

    H -->|"非 Streaming"| S["handleNonStreaming()"]
    S --> T["转发请求到上游 API"]
    T --> U["接收完整响应"]
    U --> Q

    L -->|"连接失败 / 4xx/5xx"| FB["markUnavailable()<br/>标记上游不可用"]
    FB --> G3

    R --> V["~/.claude-tap-plus/.traces/<br/>{machineID}/{projectSlug}/{sessionID}.jsonl"]
    Q --> V

    style V fill:#6f6,color:#fff
    style I fill:#ff9
    style FB fill:#f66,color:#fff
```

**说明**：代理启动后以本地 HTTP 服务器形式运行，Claude Code 的 API 请求被重定向到代理。代理在转发前会改写请求体中的 `model` 字段。当上游不可用时（连接失败或响应 4xx/5xx），自动切换到 `~/.claude/settings.json` 的兜底配置，全量替换 base_url、model 和认证信息。

---

## 3. 配置解析优先级图

多级优先级的配置解析决策树。

### 3.1 Base URL / Auth 优先级

```mermaid
flowchart TD
    A["ResolveTargetConfig()"] --> B{CLI 参数<br/>--tap-base-url?}
    B -- 有 --> RESOLVED["使用 CLI 配置"]
    B -- 无 --> C{指定了 Profile?<br/>--tap-profile?}
    C -- 有 --> D["ResolveProfileConfig(name)"]
    D --> RESOLVED
    C -- 无 --> E{环境变量<br/>ANTHROPIC_BASE_URL?}
    E -- 有 --> RESOLVED2["使用环境变量"]
    E -- 无 --> F{"~/.claude.json<br/>BaseURL?}
    F -- 有 --> RESOLVED3["使用 Claude 配置"]
    F -- 无 --> G["使用 DefaultTarget<br/>https://api.anthropic.com"]

    style RESOLVED fill:#6f6
    style RESOLVED2 fill:#6f6
    style RESOLVED3 fill:#6f6
    style G fill:#ff9
```

### 3.2 Model 优先级

```mermaid
flowchart TD
    A["Model 解析"] --> B{"profile 有 model?"}
    B -- 有 --> M1["使用 profile.model<br/>（强制覆盖）"]
    B -- 无 --> C{"~/.claude.json 有 model?"}
    C -- 有 --> M2["使用 ~/.claude.json model<br/>（默认兜底）"]
    C -- 无 --> M3["Model = 空<br/>（不做替换，原样透传）"]

    style M1 fill:#6f6
    style M2 fill:#ff9
    style M3 fill:#ddd
```

**说明**：配置按 5 级优先级解析：CLI 参数 > Profile > 环境变量 > ~/.claude.json > 默认值。Model 单独解析：profile.model > ~/.claude.json model > 空（不改写）。

---

## 4. HTTP 请求处理时序图

从代理接收到 HTTP 请求到 Trace 写入的完整时序。

```mermaid
sequenceDiagram
    participant CC as Claude Code
    participant P as ReverseProxy
    participant Up as 上游 API
    participant FB as 兜底上游
    participant SSE as SSEReassembler
    participant TW as TraceWriter
    participant FS as 文件系统

    CC->>P: HTTP 请求

    alt 路径 == /_internal/trace-init
        P->>P: handleInternal()
        P->>FS: NewTraceWriter(path)
        P->>FS: Write proxy.json
        P-->>CC: 200 OK {trace_path}
    else 正常 API 请求
        P->>P: 读取请求体 → 解析 JSON
        P->>P: rewriteModel() 改写 model
        P->>P: turn++（请求计数）

        alt upstreamAvailable == true
            P->>P: FilterHeaders（过滤 Header）
            P->>Up: 转发请求（profile target）
        else upstreamAvailable == false
            P->>P: rewriteModel() 改写为 fallback model
            P->>P: 替换认证头（token/api_key）
            P->>FB: 转发请求（fallback target）
        end

        alt 上游连接失败
            Up--xP: 连接错误
            P->>P: markUnavailable()
            P-->>CC: 502 Bad Gateway
        else 上游返回 4xx/5xx
            Up-->>P: 500 错误
            P->>P: markUnavailable()
            P-->>CC: 透传错误响应
        else Streaming 响应
            Up-->>P: 200 OK (SSE)
            loop 每个 SSE 数据块
                Up-->>P: data: {...}
                P->>SSE: FeedBytes(chunk)
            end
            P->>SSE: Reconstruct()
            SSE-->>P: 完整响应 map
            P->>P: NormalizeUsage()
            P->>TW: Write(record)
            TW->>FS: 追加 JSONL 行
            P-->>CC: SSE 流（透传）
        else 非 Streaming 响应
            Up-->>P: 200 OK {json}
            P->>P: NormalizeUsage()
            P->>TW: Write(record)
            TW->>FS: 追加 JSONL 行
            P-->>CC: JSON 响应（透传）
        end
    end
```

**说明**：代理在转发前会改写请求体中的 `model` 字段。当上游可用时使用 profile 的 target 转发；当上游不可用时自动切换到 fallback 配置（来自 `~/.claude/settings.json`），全量替换 base_url、model 和认证信息。上游首次失败（连接失败或 4xx/5xx）会触发 `markUnavailable()`，后续请求全部走 fallback。

---

## 5. SSE 重组流程图

SSE 字节流重组为完整 API 响应的过程。

```mermaid
flowchart LR
    subgraph 输入
        A["原始 SSE 字节流<br/>data: {content_block...}<br/>data: {content_block_delta...}<br/>data: {message_delta...}<br/>data: [DONE]"]
    end

    subgraph FeedBytes 处理
        B["逐行解析"]
        C{"行类型?"}
        D["event: xxx<br/>记录事件类型"]
        E["data: xxx<br/>累积到 curData"]
        F["空行<br/>完成一个事件"]
    end

    subgraph accumulate 累积
        G["按事件类型累积<br/>content_block → 初始化快照<br/>content_block_delta → 增量更新<br/>message_delta → 统计更新"]
    end

    subgraph Reconstruct 重建
        H["合并所有累积数据<br/>返回完整响应 map"]
    end

    subgraph 输出
        I["完整 API 响应<br/>包含 content + usage"]
    end

    A --> B --> C
    C -->|"event:"| D --> G
    C -->|"data:"| E --> G
    C -->|"空行"| F --> G
    G --> H --> I

    style A fill:#ff9
    style I fill:#6f6,color:#fff
```

**说明**：SSE 流是分块发送的，代理需要将所有片段重组为完整的 API 响应才能正确统计 Token 用量。`content_block` 事件初始化响应快照，后续的 `delta` 事件增量更新内容。

---

## 6. 后端服务分层架构图

api → service → store 的调用关系。

```mermaid
graph TD
    subgraph HTTP 入口
        REQ["HTTP 请求"]
    end

    subgraph API 路由层
        R["router.go（15 个端点）"]
        IH["IssueHandler"]
        SH["SessionHandler"]
        PH["ProxyHandler"]
        MH["MachineHandler"]
        PrH["ProjectHandler"]
        LH["LogHandler"]
        CH["ConfigHandler"]
        STH["StatusHandler"]
        HH["Health"]
    end

    subgraph Service 业务层
        IS["IssueService"]
        SS["SessionService"]
        MS["MachineService"]
        PS["ProjectService"]
        TS["TokenService"]
        TrS["TraceService"]
        LS["LogService"]
        CS2["ConfigService"]
        STS["StatusService"]
        CS["CleanupService"]
        WD["IdleWatchdog"]
    end

    subgraph Store 持久化层
        IStore["IssueStore"]
        SStore["SessionStore"]
        MStore["MachineStore"]
        PrStore["ProjectStore"]
        CStore["ConfigStore"]
        LStore["LogStore"]
        SQLite["SQLiteStore"]
        DB[("SQLite<br/>backend.db")]
    end

    REQ --> R
    R --> IH & SH & PH & MH & PrH & LH & CH & STH & HH

    IH --> IS
    SH --> SS
    MH --> MS
    PrH --> PS
    LH --> LS
    CH --> CS2
    STH --> STS
    SS --> TS & TrS

    IS --> IStore
    SS --> SStore
    MS --> MStore
    PS --> PrStore
    LS --> LStore
    CS2 --> CStore
    STS --> SStore & IStore
    CS --> SStore
    WD --> IStore

    IStore & SStore & MStore & PrStore & CStore & LStore --> SQLite
    SQLite --> DB

    style REQ fill:#4a90d9,color:#fff
    style DB fill:#f9f, color:#000
```

**说明**：严格的单向依赖——API 层调用 Service 层，Service 层调用 Store 接口，Store 接口由 SQLiteStore 统一实现。路由从原来的 7 个端点扩展到 15 个，新增 Machine/Project/Log/Config/Status 五组处理器。CleanupService 和 IdleWatchdog 是后台任务，直接操作 Store 层。

---

## 7. 数据库 ER 图

6 张表的关系图。

```mermaid
erDiagram
    machines {
        INTEGER id PK
        TEXT machine_id UK
        TEXT os
        TEXT hostname
        TEXT username
        DATETIME first_seen_at
        DATETIME last_seen_at
    }

    projects {
        INTEGER id PK
        TEXT project_slug UK
        TEXT project_cwd
        DATETIME first_seen_at
        DATETIME last_seen_at
    }

    sessions {
        INTEGER id PK
        TEXT session_id UK
        TEXT machine_id FK
        TEXT os
        TEXT project_slug FK
        TEXT project_cwd
        TEXT transcript_path
        TEXT local_trace_path
        TEXT model
        TEXT source
        TEXT status
        DATETIME registered_at
        DATETIME closed_at
        TEXT close_reason
    }

    issue_claims {
        INTEGER id PK
        TEXT repo_full_name
        INTEGER issue_number
        TEXT issue_title
        TEXT status
        TEXT session_id
        DATETIME claimed_at
        DATETIME updated_at
    }

    config {
        TEXT key PK
        TEXT value
        DATETIME updated_at
    }

    proxies {
        TEXT proxy_id PK
        TEXT project_slug
        TEXT status
        DATETIME registered_at
        DATETIME last_ping_at
    }

    machines ||--o{ sessions : "machine_id"
    projects ||--o{ sessions : "project_slug"
    projects ||--o{ proxies : "project_slug"
    sessions ||--o{ issue_claims : "session_id"
```

**说明**：`machines` 和 `projects` 通过 `machine_id` 和 `project_slug` 与 `sessions` 关联。`issue_claims` 通过 `session_id` 与 `sessions` 关联，记录哪个会话领取了哪个 Issue。`config` 表是独立的键值存储，用于系统运行时配置。`proxies` 表记录已注册的代理实例，通过 `project_slug` 关联项目。`issue_claims` 有 UNIQUE(repo_full_name, issue_number) 约束确保同一 Issue 不会被重复创建记录。

---

## 8. 会话管理流程图

Push/Pull/Status 三种操作的流程。

```mermaid
flowchart TD
    subgraph session-push
        A1["扫描 ~/.claude/projects/"] --> A2["查找 JSONL 会话文件"]
        A2 --> A3["ParseSessionJSONL()<br/>解析元信息"]
        A3 --> A4["copyFile()<br/>复制到本地存储"]
        A4 --> A5["SaveMeta()<br/>更新 meta.json"]
    end

    subgraph session-pull
        B1["读取本地存储 meta.json"] --> B2["遍历 SessionEntry"]
        B2 --> B3["复制 JSONL 到<br/>~/.claude/projects/"]
        B3 --> B4["更新 meta.json"]
    end

    subgraph session-status
        C1["LoadMeta()"] --> C2["汇总统计"]
        C2 --> C3["输出会话数量、大小等"]
    end
```

**说明**：session-push 从 Claude Code 的项目目录收集会话数据到本地存储。session-pull 从本地存储恢复会话到 Claude Code 目录。session-status 查看本地存储的会话统计信息。

---

## 9. 包间依赖关系图

internal/ 下所有包的依赖拓扑。

```mermaid
graph TD
    subgraph cmd 层
        CMD["cmd/claude-tap"]
    end

    subgraph internal 包
        CONFIG["config"]
        PROXY["proxy"]
        SSE["sse"]
        TRACE["trace"]
        USAGE["usage"]
        SESSION["session"]
        LOGGER["logger"]
        API["backend/api"]
        SVC["backend/service"]
        STORE["backend/store"]
        DOMAIN["backend/domain"]
        SERVER["backend/server"]
    end

    CMD --> CONFIG & PROXY & SESSION & LOGGER
    CMD --> SERVER

    CONFIG --> LOGGER
    PROXY --> SSE & TRACE & LOGGER
    SSE --> USAGE & LOGGER
    TRACE --> USAGE & LOGGER
    SESSION --> TRACE & LOGGER

    SERVER --> API & SVC & STORE & LOGGER
    API --> SVC & LOGGER
    SVC --> STORE & LOGGER
    STORE --> DOMAIN & LOGGER

    style LOGGER fill:#888,color:#fff
    style USAGE fill:#e8f4fd
    style DOMAIN fill:#f9f, color:#000
```

**说明**：所有包都依赖 `logger/`。代理管线中 `proxy → sse → usage` 和 `proxy → trace → usage` 形成两条数据流路径。后端部分严格分层：server → api → service → store → domain。

---

## 10. Token 用量归一化流程图

多 Provider 的 Token 字段映射到统一格式。

```mermaid
flowchart TD
    A["原始 usage 对象<br/>map[string]any"] --> B["NormalizeUsage()"]

    B --> C["映射 input_tokens"]
    C --> C1{"查找优先级"}
    C1 -->|"1"| C2["input_tokens<br/>(Anthropic)"]
    C1 -->|"2"| C3["prompt_tokens<br/>(OpenAI)"]
    C1 -->|"3"| C4["promptTokenCount<br/>(Gemini)"]

    B --> D["映射 output_tokens"]
    D --> D1{"查找优先级"}
    D1 -->|"1"| D2["output_tokens<br/>(Anthropic)"]
    D1 -->|"2"| D3["completion_tokens<br/>(OpenAI)"]
    D1 -->|"3"| D4["candidatesTokenCount<br/>(Gemini)"]

    B --> E["映射 cache_read_input_tokens"]
    E --> E1{"查找优先级"}
    E1 -->|"1"| E2["cache_read_input_tokens<br/>(Anthropic)"]
    E1 -->|"2"| E3["cached_tokens<br/>(OpenAI)"]
    E1 -->|"3"| E4["cachedContentTokenCount<br/>(Gemini)"]

    B --> F["处理嵌套结构"]
    F --> F1["input_tokens_details.cached_tokens"]
    F --> F2["prompt_tokens_details.cached_tokens"]
    F1 & F2 -->|"累加"| E2

    C2 & D2 & E2 & F --> G["标准化结果<br/>map[string]int64<br/>input_tokens<br/>output_tokens<br/>cache_read_input_tokens<br/>cache_creation_input_tokens"]

    style G fill:#6f6,color:#fff
```

**说明**：不同 AI Provider 的 API 响应使用不同的字段名表示 Token 用量。`NormalizeUsage()` 使用 `firstInt64()` 按优先级查找第一个非零值，将所有格式统一为 Anthropic 规范。嵌套结构中的缓存 Token 会被额外累加。

---

> 相关文档：[claude_tap_plus 架构文档](claude-tap-plus-architecture.md)
