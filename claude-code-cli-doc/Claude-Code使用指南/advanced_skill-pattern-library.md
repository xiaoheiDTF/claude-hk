# Claude-Code / Advanced / Skill-Pattern-Library

> 来源: claudecn.com

# 模式库：把工程经验沉淀为 Skills

很多团队用 Claude Code 的瓶颈不是“不会写代码”，而是**重复解释同一套工程约定**：API 怎么设计、数据怎么访问、前端组件怎么组织、怎么避免常见坑。更可持续的做法是把这些经验沉淀成 Skills，让 Claude 在每次任务里自动遵循。
下文把常见的后端/前端工程模式整理成更适合团队落地的写法与取舍，重点不是“堆知识”，而是把可执行的约束与验收写清楚。

## 1) Skill 不要写成“百科”，要写成“约束 + 模板 + 检查清单”

一个可用的 Skill 最少包含三块：

- When to Use：什么场景必须启用（新 API、重构、性能优化等）
- Patterns：团队认同的模式（最好配小段代码模板）
- Checklist：完成前必须过的检查点（输入校验、错误处理、测试等）
## 2) 后端模式（摘录）：API / Repository / N+1 / Cache

### RESTful API 结构（示例节选）

```typescript
GET    /api/markets                 # List resources
GET    /api/markets/:id             # Get single resource
POST   /api/markets                 # Create resource
PUT    /api/markets/:id             # Replace resource
PATCH  /api/markets/:id             # Update resource
DELETE /api/markets/:id             # Delete resource
```

### Repository Pattern（示例节选）

```typescript
interface MarketRepository {
  findAll(filters?: MarketFilters): Promise<Market[]>
  findById(id: string): Promise<Market | null>
  create(data: CreateMarketDto): Promise<Market>
  update(id: string, data: UpdateMarketDto): Promise<Market>
  delete(id: string): Promise<void>
}
```

### 避免 N+1（示例节选）

```typescript
const markets = await getMarkets()
const creatorIds = markets.map(m => m.creator_id)
const creators = await getUsers(creatorIds)  // 1 query
const creatorMap = new Map(creators.map(c => [c.id, c]))

markets.forEach(market => {
  market.creator = creatorMap.get(market.creator_id)
})
```

落地建议：把“避免 N+1”写进后端 Skill 的 Checklist，让 Claude 在写查询代码时默认提醒你做批量查询/预加载。

## 3) 前端模式（摘录）：组件组合 / 自定义 Hook

### Composition Over Inheritance（示例节选）

```typescript
export function Card({ children, variant = 'default' }: CardProps) {
  return <div className={`card card-${variant}`}>{children}</div>
}
```

### Debounce Hook（示例节选）

```typescript
export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value)

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value)
    }, delay)

    return () => clearTimeout(handler)
  }, [value, delay])

  return debouncedValue
}
```

落地建议：前端 Skill 里不要堆太多“UI 细节”，重点固化：组件边界、状态管理、性能基本盘（避免不必要渲染、避免深层 props drilling 等）。

## 4) 如何让 Skill 真正生效（而不是躺在目录里）

把 Skill 变成团队工作流的一部分：

- 在 CLAUDE.md 写清楚：哪些任务必须使用哪些 Skills（例如“新 API 必须应用 backend-patterns”）。
- 把关键流程做成 /command（例如 /plan、/code-review），让每次任务自然经过 Skill 的检查点。
- 用 Hooks 把高频“遗漏项”自动化提醒（console.log、格式化、类型检查等）。
相关页面：

- Agent Skills
- 团队 Starter Kit（最小可用配置）
- 团队质量门禁：Plan → TDD → Build Fix → Review
## 5) 进阶：用 UserPromptSubmit 做 Skill 自动推荐（可选）

现实里“Skill 写好了但没被用起来”的常见原因不是目录结构，而是：用户提示词不稳定、文件路径没说清、模型没把你的 Skill 作为首选。

一种很实用的工程化做法是：在 `UserPromptSubmit` 时运行一个轻量脚本，**只做匹配与提醒**，把“应该启用哪些 Skills”变成可执行步骤（节选）：

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/skill-eval.sh",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
```

其匹配规则可以集中在一个 JSON 配置文件中（节选）：

```json
{
  "config": { "minConfidenceScore": 3, "maxSkillsToShow": 5 },
  "scoring": { "keyword": 2, "pathPattern": 4, "intentPattern": 4 }
}
```

落地建议：

- 把它当作“提醒型护栏”，而不是“自动启用器”：让人或命令先判断 YES/NO，再决定是否启用 Skill。
- 先从 5~10 个最核心的 Skills 写规则（关键词/路径/意图），不要一口气把全仓库都纳入匹配。
## 6) 给人看的“Skills 索引”和“技能组合”（强烈推荐）

很多团队只写 `SKILL.md`，但缺少一个“给人看的入口”，导致：

- 新成员不知道有哪些 Skills
- 不知道一个任务应该启用哪几个 Skills
一种常见做法是维护一个 `.claude/skills/README.md`，把 Skills 按类别列出，并给出“常见任务的技能组合”（节选）：

```text
### Building a New Feature
1. react-ui-patterns
2. graphql-schema
3. core-components
4. testing-patterns
```

落地建议：你可以先用 10 行以内写一个最小索引（只列名称 + 1 句用途 + 推荐组合），后续再慢慢补全。

## 参考

无
