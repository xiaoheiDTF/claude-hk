# Claude-Code / Workflows / Build-Troubleshooting

> 来源: claudecn.com

# 构建排障：最小 diff 的增量修复

构建失败（TypeScript 报错、依赖缺失、模块解析失败）最怕两件事：

- 一次性改太多，修好一个又引入三个
- 趁机重构，最后“改对了但回不去了”
一个更稳的排障思路是：**只做最小修改把 build 拉回绿灯，不做架构调整**。

## 1) 排障节奏：收集 → 分类 → 一次修一个

推荐按这个节奏推进：

- 跑一次完整构建/类型检查，把错误收集全（不要只看第一个）。
- 按类型分类：类型推断失败 / import 错误 / null/undefined / 配置问题 / 依赖冲突。
- 按阻塞程度排序：会导致 build 直接失败的优先。
- 一次只修一个错误，每修一次就重跑检查确认没有连锁反应。
这类节奏可以固化为团队命令（类似 `/build-fix`）：修一个 → 复跑 → 记录进度。

## 2) “最小 diff”常见修法（示例节选）

### 隐式 any：补最小类型注解

```typescript
function add(x: number, y: number): number {
  return x + y
}
```

### null/undefined：补空值路径（优先可读性）

```typescript
const name = user?.name?.toUpperCase()
```

### 泛型约束：给 T 加最小约束

```typescript
function getLength<T extends { length: number }>(item: T): number {
  return item.length
}
```

原则：类型断言（`as ...`）是最后手段；能用更明确的类型/约束/分支就别硬断言。

## 3) 何时停止继续“硬修”

当你遇到这些情况，建议停下来回到“规划/设计”再继续：

- 同一错误修了 2–3 次仍反复出现
- 修复开始触及公共 API 行为（不是单纯类型层面）
- 错误根因是架构/约定缺失（例如路径别名、模块边界）
此时可以切到 Plan Mode，先明确“要改什么约定/配置”，再恢复修复节奏（见 [计划模式](https://claudecn.com/docs/claude-code/workflows/plan-mode/)）。

## 4) 团队落地建议

- 把“构建失败如何处理”写进 团队质量门禁。
- 把常用排障命令写进 CLAUDE.md，减少重复解释。
- 如果团队经常跑长命令，建议结合 tmux 与 Hooks 做“提醒/记录”（见 会话连续性与战略压缩）。
## 参考

无
