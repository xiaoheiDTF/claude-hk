# PRD：claude-tap-plus 别名路由与多环境隔离

| 项 | 值 |
|---|---|
| 版本 | v1.0（草案，待评审） |
| 日期 | 2026-06-15 |
| 状态 | 待评审 |
| 关联项目 | claude_tap_plus |
| 作者 | xiaoheiDTF |

---

## 1. 背景与目标

### 1.1 背景
claude_tap_plus 是 Claude Code 的本地反向代理（Go），通过拦截 API 请求转发到第三方 LLM 后端（glm 系列等）。当前后端隔离依赖 `~/.claude-tap-plus/profiles.json` 的扁平 profile 结构，存在以下局限：

- `ProfileConfig` 仅 5 个字段（`base_url`/`api_key`/`auth_token`/`provider`/`model`），**不支持任意 env 变量**；
- `profile.model` 是单一值，转发时 `rewriteModel` 会**强制覆盖所有请求**的 model 字段（一刀切），破坏 Claude Code 的"主模型 + 后台小模型 + 子代理"分工；
- 路由只按**启动时选定的 profile**，**不按请求中的 model 分流**到不同后端/key；
- subagent 走 kimi、同模型用不同 key 等"按用途/按模型路由"的需求无法实现。

### 1.2 目标
1. **别名路由**：以"别名（alias）"为核心实体，每个别名独立绑定一组 `(真实 model, api_key, base_url)`；proxy 转发时按请求 model（=别名）改写成真实 model 并选用对应凭证。
2. **多环境隔离**：profile 只承载 env 变量（客户端行为配置），`--tap-profile=<name>` 切换启动配置。
3. **key 池与多样化**：base_url + api_key 集中在别名表全局共享；同 model 可挂不同 key、同 key 可配不同 model，自由组合。
4. **容错 fallback**：某别名后端失败时，按其真实 model 名查找其他"同 model"的别名重试。

### 1.3 非目标（本期不做）
- 按 token 用量/成本自动选择 key（负载均衡策略，仅做失败容错）。
- 别名级速率限制、配额管理。
- profiles.json 的远程下发/加密存储。

---

## 2. 术语

| 术语 | 含义 |
|---|---|
| **别名 alias** | Claude Code 侧使用的 model 名（可自定义，可带 `[1m]` 等后缀），绑定一组后端凭证。是路由的查询单元。 |
| **真实 model** | 转发到后端时实际写入请求体 `model` 字段的模型标识（如 `glm-5.2[1m]`）。 |
| **profile** | 启动配置，只含 `env`。`--tap-profile` 选定其一，注入给 Claude Code 子进程。 |
| **key 池** | `aliases` 表中所有 `(api_key/auth_token, base_url)` 凭证的集合，全局共享，不按 profile 重复定义。 |

---

## 3. 现状与约束（代码证据）

| 现状 | 证据 | 对本 PRD 的影响 |
|---|---|---|
| ProfileConfig 仅 5 字段，无 env | `profiles.go:13-19` | 需替换为新结构 |
| profile.model 单值强制覆盖 | `reverse.go:959 rewriteModel`、`reverse.go:172` | 改为按别名查表改写 |
| 路由只看启动 profile，不看请求 model | `reverse.go:204`（Phase1 全部打 `p.target`） | 需在转发前按请求 model 选 target |
| fallback 仅容错，400 不触发 | `reverse.go:743 shouldFallback`（仅 401/403/429/5xx） | fallback 触发条件保持，数据源改为按真实 model 查别名 |
| 已有"同 model 优先"fallback 逻辑 | `profiles.go:99 ResolveFallbackProfiles` | 思路可复用，数据源从 profile 改为 alias |
| has1mContext 纯字符串 `[1m]` 检测 | `cc-source/src/utils/context.ts:39`，且 `:69-72` 为最高优先级显式 opt-in | 别名带 `[1m]` 即可被 Claude Code 识别为 1M context |
| env 经 proxy 注入子进程（非 settings.json） | `main.go:305 BuildChildEnv` + `:312-318` | 不受 Claude Code `SAFE_ENV_VARS` 白名单限制，可注入任意 env |
| kimi/reasoning 注入按 base_url 判断 | `reverse.go:865 IsKimiURL`、`:268` fallback 分支 | 别名级按其 base_url 自动判断，或显式 `kimi_mode` |
| `ANTHROPIC_BASE_URL` 由代理强制设为本地 | `main.go:305`、`client.go:120 BuildSettingsArgs` | profile.env 中**禁止**再设 `ANTHROPIC_BASE_URL` |

---

## 4. 配置规范（profiles.json）

### 4.1 Schema

```jsonc
{
  // 顶层
  "default_profile": "<profile 名>",     // 可选，未指定 --tap-profile 时使用
  "default_alias":   "<别名 name>",      // 可选，请求 model 未命中任何别名时的兜底

  // 别名表（全局共享的 key 池 + 路由目标）
  "aliases": [
    {
      "name":      "<别名>",              // 必填，Claude Code 用作 model 名（如 "opus[1m]"）
      "model":     "<真实 model>",        // 必填，转发时写入请求体 model 字段（如 "glm-5.2[1m]"）
      "base_url":  "<后端地址>",          // 必填
      "api_key":   "<API key>",           // 与 auth_token 二选一
      "auth_token":"<OAuth token>",       // 可选，与 api_key 互斥
      "kimi_mode": true,                  // 可选，显式开启 reasoning_content 注入（默认按 base_url 自动判断）
      "priority":  100                    // 可选，同真实 model 多别名时的选用优先级，大者优先；默认 0，相同则按数组顺序
    }
  ],

  // 启动配置（只含 env）
  "profiles": {
    "<profile 名>": {
      "env": {
        "<ENV_VAR>": "<value>"            // Claude Code 客户端行为变量（见 4.3）
      }
    }
  }
}
```

### 4.2 字段说明

| 路径 | 必填 | 说明 |
|---|---|---|
| `default_profile` | 否 | 未传 `--tap-profile` 时使用的 profile 名 |
| `default_alias` | 否 | 请求 model 未命中任何别名时的兜底别名 |
| `aliases[].name` | 是 | 别名，Claude Code 在 env 中用作 model 名。可带 `[1m]` 等后缀 |
| `aliases[].model` | 是 | 真实模型名，转发时改写进请求体 |
| `aliases[].base_url` | 是 | 后端 API 地址 |
| `aliases[].api_key` | 否 | API key，与 `auth_token` 互斥 |
| `aliases[].auth_token` | 否 | OAuth token，与 `api_key` 互斥 |
| `aliases[].kimi_mode` | 否 | 显式指定是否注入 reasoning_content；缺省时按 `base_url` 是否匹配 kimi/moonshot/deepseek 自动判断 |
| `aliases[].priority` | 否 | 同真实 model 存在多个别名时的选用优先级（含 fallback 排序） |
| `profiles[].env` | 是 | 启动时注入 Claude Code 子进程的环境变量 |

### 4.3 完整示例

```json
{
  "default_profile": "work",
  "default_alias": "sonnet",

  "aliases": [
    { "name": "opus[1m]",  "model": "glm-5.2[1m]", "base_url": "https://glm.example.com", "api_key": "sk-aaa" },
    { "name": "opus2[1m]", "model": "glm-5.2[1m]", "base_url": "https://glm.example.com", "api_key": "sk-bbb" },
    { "name": "sonnet",    "model": "glm-5.1",      "base_url": "https://glm.example.com", "api_key": "sk-aaa" },
    { "name": "haiku",     "model": "glm-4.7",      "base_url": "https://glm.example.com", "api_key": "sk-bbb" },
    { "name": "kimi",      "model": "kimi-k2",      "base_url": "https://api.kimi.com",    "api_key": "sk-kkk" }
  ],

  "profiles": {
    "work": {
      "env": {
        "ANTHROPIC_MODEL": "opus[1m]",
        "CLAUDE_CODE_SUBAGENT_MODEL": "kimi",
        "ANTHROPIC_SMALL_FAST_MODEL": "haiku",
        "ANTHROPIC_DEFAULT_OPUS_MODEL": "opus[1m]",
        "ANTHROPIC_DEFAULT_SONNET_MODEL": "sonnet",
        "ANTHROPIC_DEFAULT_HAIKU_MODEL": "haiku",
        "CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "70",
        "CLAUDE_CODE_AUTO_COMPACT_WINDOW": "600000",
        "API_TIMEOUT_MS": "3000000"
      }
    },
    "personal": {
      "env": { "ANTHROPIC_MODEL": "sonnet" }
    }
  }
}
```

> 示例覆盖：同 model 不同 key（`opus[1m]`/`opus2[1m]` 都是 `glm-5.2[1m]`）；多 model 共用 key（`opus[1m]`/`sonnet` 都用 `sk-aaa`）；不同后端（kimi）；1M 别名（`opus[1m]`）；env 随 profile 切换。

---

## 5. 功能需求

### F1 别名解析与路由
- F1.1 proxy 收到请求后，解析请求体 `model` 字段（即别名 name）。
- F1.2 在 `aliases` 表中按 `name` 精确匹配。
- F1.3 命中后：将请求体 `model` 改写为该别名的真实 `model`；使用该别名的 `base_url` + `api_key`/`auth_token` 转发。
- F1.4 别名的 `kimi_mode`：显式值优先；缺省时按其 `base_url` 匹配 kimi/moonshot/deepseek 前缀自动判断（复用 `IsKimiURL`）。
- F1.5 未命中任何别名：使用 `default_alias`（若配置）；否则返回错误。

### F2 key 池与 fallback
- F2.1 主别名转发失败（连接错误，或响应码命中 `shouldFallback`：401/403/429/5xx）时进入 fallback。
- F2.2 fallback 候选 = `aliases` 中**真实 model 与主别名相同**、且排除主别名自身的其他别名，按 `priority` 降序、同 priority 按数组顺序排列。
- F2.3 逐个尝试候选（复用现有 fallback 转发逻辑：按候选的 base_url/key 改写认证、按其 base_url 判断 kimi 注入）。
- F2.4 主别名 + 所有候选均失败 → 返回友好错误（复用 `logProxyError`）。
- F2.5 fallback 选用策略本期仅"失败容错"，不做轮询/负载均衡。

### F3 env 启动配置（profile）
- F3.1 `--tap-profile=<name>` 指定启动配置；未指定时用 `default_profile`。
- F3.2 启动 Claude Code 子进程前，将该 profile 的 `env` 合并进子进程环境（`BuildChildEnv` 之后），覆盖同名继承变量。
- F3.3 `env` 中**禁止**包含 `ANTHROPIC_BASE_URL`（代理已强制设为本地地址）；解析时检测到应告警并忽略。
- F3.4 切换 profile 即切换整套 env；后端凭证与 env 解耦（凭证在 aliases，env 在 profiles）。

### F4 兼容与迁移
- F4.1 本期为 **breaking change**：`profiles.json` 格式与旧扁平 `ProfileConfig` 不兼容。
- F4.2 启动时检测旧格式（顶层存在 `profiles[].base_url` 等旧字段）→ 打印迁移提示并退出，避免静默错误。
- F4.3 `--tap-target`/`--tap-api-key`/`--tap-base-url`/`--tap-auth-token` 等 CLI 覆盖参数保留（优先级最高），作为别名路由之外的临时覆盖。

### F5 安全
- F5.1 `profiles.json` 含明文密钥，创建/读取时应校验文件权限，权限过宽（如其他用户可读）时告警。
- F5.2 trace 记录中 api_key/auth_token 沿用现有脱敏逻辑（`FilterHeaders`）。

---

## 6. 工作流程

```
1) claude-tap-plus --tap-profile=work claude
2) 读取 profiles.work.env → 合并进子进程 env（ANTHROPIC_MODEL=opus[1m], ...）
3) 加载 aliases 表到 proxy
4) Claude Code 发请求：
     主对话   → model="opus[1m]"
     subagent → model="kimi"
     后台任务 → model="haiku"
5) proxy 每个请求：
     解析 model(别名) → 查 aliases(name 匹配)
       命中 → 改写为真实 model + 用对应 base_url/key 转发
              （base_url 为 kimi → 注入 reasoning_content）
       失败(401/403/429/5xx) → 按真实 model 找同 model 的其他别名(priority 排序)重试
       全失败 → 友好错误
```

---

## 7. 实现范围（文件级）

| 文件 | 改动 |
|---|---|
| `internal/config/profiles.go` | 新增 `Alias`、`Profile`（含 env）、`ProfilesFile`（default_profile/default_alias/aliases/profiles）结构；`ReadProfiles`、`ResolveAlias(byName)`、`ResolveFallbackAliases(byRealModel, excludeName)`、`ReadProfileEnv(name)` |
| `internal/proxy/reverse.go` | `ReverseProxy` 增加 `aliases` 字段；`serveHTTP` 转发前按请求 model 查别名 → 选定 target/auth/真实 model/kimiMode；主别名失败按真实 model 走 fallback 别名链 |
| `cmd/claude-tap/main.go` | 解析 `--tap-profile`；读 `profiles[name].env` 注入 `childEnv`（F3）；将 aliases 传给 proxy（`SetAliases`）；保留 CLI 覆盖参数 |

---

## 8. 验收标准

| 编号 | 场景 | 预期 |
|---|---|---|
| AC1 | 发请求 `model=opus[1m]` | 后端收到 `model=glm-5.2[1m]`，请求来自 `sk-aaa` + glm base_url |
| AC2 | `opus[1m]` 后端返回 401 | 自动切到同真实 model 的 `opus2[1m]`（`sk-bbb`）重试并成功 |
| AC3 | `--tap-profile=work` 启动 | Claude Code 子进程 env 含 `work.env` 全部变量 |
| AC4 | `ANTHROPIC_MODEL=opus[1m]` | Claude Code `getContextWindowForModel` 返回 1,000,000（has1mContext 命中） |
| AC5 | 请求 `model=kimi` | 后端收到 `model=kimi-k2`，且 assistant tool_call 消息被注入 reasoning_content |
| AC6 | 请求 `model=unknown-xxx`（无 default_alias） | 返回明确错误，不崩溃 |
| AC7 | 请求 `model=unknown-xxx`（有 default_alias=sonnet） | 兜底走 sonnet 别名 |
| AC8 | 旧格式 profiles.json | 启动时打印迁移提示并退出，不静默运行 |
| AC9 | profile.env 含 `ANTHROPIC_BASE_URL` | 告警并忽略该项 |
| AC10 | profiles.json 权限过宽 | 启动时告警 |

---

## 9. 风险与注意事项

| 风险 | 说明 | 缓解 |
|---|---|---|
| Breaking change | profiles.json 格式变更，旧配置失效 | F4.2 检测旧格式并提示迁移 |
| 别名与 Claude Code 内部 model 检测 | 除 `[1m]` 外，Claude Code 部分逻辑按 `claude-xxx` 前缀判断模型族（如 `getModelFamilyInfo`），别名不匹配时某些 UI hint/计费预估不准 | 功能不受影响；需在文档说明别名主要用于路由，Claude Code 内部部分展示可能不准 |
| 明文密钥 | profiles.json 含 api_key/auth_token | F5.1 权限校验 + 告警；建议 0600 |
| 同名别名 | aliases 数组出现重复 name | 解析时报错或后者覆盖（实现时确定，倾向报错） |
| env 与代理 base_url 冲突 | profile.env 误设 ANTHROPIC_BASE_URL | F3.3 检测并忽略 |

---

## 10. 未决项（需评审确认）

1. **同别名名重复**：aliases 中出现相同 `name` 时，报错 还是 后者覆盖？（倾向报错）
2. **default_alias 必填还是可选**：当前设计可选。是否强制要求配置以避免未命中？
3. **priority 字段**：是否需要，还是直接按数组顺序？（当前提供 priority，默认 0）
4. **旧格式兼容层**：是否需要在过渡期同时支持旧扁平格式自动转换？（当前设计为检测+提示，不做自动转换）
5. **auth_token 与 api_key 互斥校验**：同一别名同时配置两者时，报错还是优先级取一？

---

## 11. 后续（本期之后）

- 别名级用量统计/成本展示（已有 trace + token 归一化基础）。
- key 池主动轮询/负载均衡（当前仅失败容错）。
- profiles.json 热重载（当前需重启）。
