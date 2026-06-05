---
name: 003-6-2-frontend-cdd
description: 前端 CDD（Vue/Storybook）：根据前端 BDD 写 Storybook Story，通过 Atomic Design 驱动组件实现
user-invocable: true
allowed-tools:
  - Bash
  - Read
  - Write
  - Edit
  - Glob
  - Grep
---

# 单项目开发流程 — 前端 CDD（Component Driven Development）

> 文档日期：2026-06-03
> 所属系列：单项目开发完整流程
> 阶段定位：根据前端 BDD 写 Vue 组件级测试，通过 Storybook 场景驱动组件实现

---

## 一、核心目标

- 从前端 BDD 场景生成 Storybook Story 和组件测试
- 遵循 Atomic Design 原子设计规范组织组件
- 每个 Vue 组件文件对应一个 Storybook Story 文件（1:1 强制映射）
- 保证所有 UI/交互行为符合前端 BDD

---

## 二、阶段职责

### 本阶段负责（约束）

- 从前端 BDD 场景生成 Storybook Story 和组件测试
- 编写组件实现让 Story 和测试通过
- 按 Atomic Design 层级组织组件目录结构
- 每个 .vue 文件必须有对应的 .stories.ts 文件

### 本阶段不负责（约束）

- 前端 BDD 场景编写 → 前端 BDD 阶段
- UI 设计与视觉规范 → UI 设计规范 + 状态定义阶段（06）
- 接口契约定义 → API 契约阶段
- 后端服务开发 → 后端 TDD 阶段
- 端到端测试 → 联调阶段

---

## 三、测试发现规则

### 规则 1：前端 BDD 场景映射

每条前端 BDD 场景描述了用户在页面上的交互行为，CDD 需要将其拆解为组件级别的 Story。

映射方式：
- BDD 的 UI 状态前提 → Story 的 args（组件 props 初始状态）
- BDD 的用户操作 → Story 的 play 函数（交互模拟）
- BDD 的预期 UI 变化 → Story 的断言（assertion）

### 规则 2：Atomic Design 分层驱动（强约束）

组件必须按 Atomic Design 层级组织，每层有明确的职责边界：

| 层级 | 目录 | 职责 | 示例 |
|------|------|------|------|
| Atoms | `components/atoms/` | 最小可复用单元，无业务逻辑 | Button、Input、Badge、Icon、Label |
| Molecules | `components/molecules/` | Atom 组合，有局部交互逻辑 | SearchBar、FormField、Card |
| Organisms | `components/organisms/` | 业务组件，承载业务逻辑 | UserCard、ArticleList、FavoriteButton |
| Templates | `components/templates/` | 页面骨架，定义布局和插槽 | MainLayout、SidebarLayout |
| Pages | `pages/` 或 `views/` | 完整页面，组合 Organisms + Templates | ArticlePage、UserProfilePage |

**分层判断标准：**

- 组件内出现业务逻辑（调用接口、状态管理）→ Organisms
- 组件只组合 Atoms，有交互但无业务逻辑 → Molecules
- 组件是单个 UI 元素，通过 props 控制 → Atoms
- 组件定义布局结构，用 slot 承载内容 → Templates
- 组件组装 Templates 和 Organisms，绑定路由数据 → Pages

### 规则 3：1:1 Story 映射（强约束）

**每一个 .vue 文件必须有一个对应的 .stories.ts 文件。** 不存在没有 Story 的组件，也不存在没有组件的 Story。

文件对应关系：

```
components/atoms/BaseButton.vue        → components/atoms/BaseButton.stories.ts
components/molecules/SearchBar.vue     → components/molecules/SearchBar.stories.ts
components/organisms/ArticleList.vue   → components/organisms/ArticleList.stories.ts
```

Story 必须覆盖该组件的所有 UI 状态。一个 Story 文件至少包含：
- **Default Story** — 组件的默认渲染状态
- 按前端 BDD 场景拆分的具体交互 Story

### 规则 4：可追溯

每个 Story 必须能追溯到前端 BDD 的某条场景。追溯方式：

- Story 文件注释标注对应的前端 BDD 场景编号
- Story name 使用前端 BDD 场景名称
- 无法追溯到前端 BDD 的 Story，需要说明来源（规则 7 的扩展 Story）

### 规则 5：Vue + Storybook 规范约束（强约束）

所有组件和测试必须遵循 Vue + Storybook 规范：

**框架要求：**
- 组件框架：Vue 3（Composition API / SFC）
- 组件测试：Storybook + play 函数
- 断言方式：Storybook 内置的 interaction testing（expect）
- Mock 方案：MSW（Mock Service Worker）拦截 API 请求

**组件命名规范：**
- Atoms: `Base{Name}.vue`（如 BaseButton.vue）
- Molecules: `{Name}.vue`（如 SearchBar.vue）
- Organisms: `{Name}.vue`（如 ArticleList.vue）
- Pages: `{Name}Page.vue`（如 ArticlePage.vue）

**Story 命名规范：**
```typescript
// 每个组件的默认导出定义 meta
export default {
  title: '{Atomic层级}/{组件名}',
  component: {ComponentName},
} as Meta;

// 每个 Story 对应一个前端 BDD 场景
export const {场景名称}: StoryObj = { };
```

**目录结构规范：**
```
src/
├── components/
│   ├── atoms/          # 基础组件
│   │   ├── BaseButton.vue
│   │   └── BaseButton.stories.ts
│   ├── molecules/      # 组合组件
│   │   ├── SearchBar.vue
│   │   └── SearchBar.stories.ts
│   ├── organisms/      # 业务组件
│   │   ├── ArticleList.vue
│   │   └── ArticleList.stories.ts
│   └── templates/      # 页面骨架
│       ├── MainLayout.vue
│       └── MainLayout.stories.ts
├── pages/              # 完整页面
│   ├── ArticlePage.vue
│   └── ArticlePage.stories.ts
└── composables/        # 可复用逻辑
```

### 规则 6：文档驱动确认（强约束）

同后端 TDD 阶段，采用**文档驱动确认**模式。

**默认流程：**

1. AI 根据前端 BDD，一次性生成 Story 和组件代码
2. 代码注释中附待确认问题和总结
3. 用户在代码中批注、修改
4. AI 读取修改后继续

**主动提问的触发条件：**

- 待确认问题超过一轮未被回答
- 用户修改后组件层级不符合 Atomic Design 规范
- 发现前端 BDD 场景存在歧义，无法确定组件行为

### 规则 7：Story 扩展（规则）

在编写组件过程中，如果发现前端 BDD 未覆盖的 UI 状态或交互，**直接在 Story 中补充扩展 Story**。

**扩展 Story 的来源：**

- 组件状态机推导出了新的状态转换路径
- Atomic Design 组合过程中暴露了边界 UI 情况
- 响应式适配中发现了新的显示差异

**扩展 Story 的写法：**

- Story name 标注为"扩展: {原因}"
- 注释说明扩展来源
- 遵循与映射 Story 相同的质量标准

扩展 Story 在总结中单独列出数量。

---

## 四、Storybook-Driven 开发流程

**前置条件：UI 设计规范 + 状态定义阶段（06）已完成，HTML 原型已获用户确认。**

```
1. STORY   → 根据前端 BDD + 已确认的 HTML 原型生成 Story
2. 组件    → 编写 Vue 组件，还原 HTML 原型中的视觉效果和交互行为
3. 测试    → 在 Story 的 play 函数中编写交互测试，验证 UI 行为
4. 整理    → 按项目规范整理代码（目录、可复用逻辑、样式），保证 Story 仍通过
```

**每条前端 BDD 场景对应一个完整的 Story-组件-测试循环。** 不允许跳过 Story 直接写组件。

### STORY 阶段

从前端 BDD 场景和已确认的 HTML 原型生成 Story。**Story 的视觉效果以 HTML 原型为准**——颜色、间距、动画时长、交互行为都必须还原原型中用户确认的效果。

### 组件阶段

编写最小 Vue 组件让 Story 渲染通过。**组件只需要满足对应的 Story 渲染和交互测试**，不多做。

组件阶段允许：
- 临时硬编码数据
- 简单的 inline 样式
- 不抽取任何可复用逻辑

### 测试阶段

在 Story 的 play 函数中编写交互测试，验证用户操作后的 UI 变化。

### 整理阶段

Story 通过后，按项目规范整理代码：

- **目录结构** — 组件放到正确的 Atomic Design 层级目录
- **可复用逻辑** — 多个组件共用的状态逻辑提取到 `composables/`
- **样式整理** — 公共样式提取，组件样式 scoped

整理的边界（同后端 TDD 原则）：
- 不抽离无意义的方法——一两行没有复用场景的逻辑不需要抽成 composable
- 不过度抽象——不为将来提前设计
- 目标不变——整理完之后，所有 Story 仍然通过

---

## 五、测试分层

| 层级 | 测试内容 | Story 关注点 | 对应前端 BDD |
|------|---------|---|---|
| Atoms | props 渲染、事件触发 | 不同 props 下的渲染结果 | UI 展示场景 |
| Molecules | Atom 组合交互 | 组合后的交互行为 | 交互正向/异常场景 |
| Organisms | 业务逻辑 + UI | API 调用 Mock + 状态切换 | 完整交互场景 |
| Templates | 布局和插槽 | 不同内容填充的布局 | 页面结构场景 |
| Pages | 路由数据绑定 | 完整页面渲染 | 用户旅程场景 |

---

## 六、测试写法（规则）

### 6.1 Atom Story

```typescript
// BaseButton.stories.ts
export default {
  title: 'Atoms/BaseButton',
  component: BaseButton,
  argTypes: {
    variant: { control: 'select', options: ['primary', 'secondary', 'danger'] },
    disabled: { control: 'boolean' },
  },
} as Meta;

// 默认渲染
export const Default: StoryObj = {
  args: {
    label: '点击按钮',
    variant: 'primary',
  },
};

// 禁用状态 — 对应前端 BDD 的边界场景
export const Disabled: StoryObj = {
  args: {
    label: '点击按钮',
    disabled: true,
  },
};
```

### 6.2 Molecule Story

```typescript
// SearchBar.stories.ts
export default {
  title: 'Molecules/SearchBar',
  component: SearchBar,
} as Meta;

// 交互测试 — 对应前端 BDD 的交互正向场景
export const SearchAndSubmit: StoryObj = {
  play: async ({ canvasElement, step }) => {
    const canvas = within(canvasElement);

    await step('输入搜索关键词', async () => {
      const input = canvas.getByPlaceholderText('搜索...');
      await userEvent.type(input, '测试文章');
    });

    await step('点击搜索按钮', async () => {
      const button = canvas.getByRole('button', { name: '搜索' });
      await userEvent.click(button);
    });

    await step('应触发搜索事件', async () => {
      // 验证搜索行为
    });
  },
};
```

### 6.3 Organism Story（含 API Mock）

```typescript
// ArticleList.stories.ts
export default {
  title: 'Organisms/ArticleList',
  component: ArticleList,
} as Meta;

// 加载成功 — 对应前端 BDD 的正向场景
export const LoadedWithArticles: StoryObj = {
  parameters: {
    msw: {
      handlers: [
        http.get('/api/v1/articles', () => {
          return HttpResponse.json({
            items: [mockArticle({ title: '测试文章 1' }), mockArticle({ title: '测试文章 2' })],
          });
        }),
      ],
    },
  },
  play: async ({ canvasElement, step }) => {
    const canvas = within(canvasElement);
    await step('应显示文章列表', async () => {
      expect(canvas.getByText('测试文章 1')).toBeVisible();
      expect(canvas.getByText('测试文章 2')).toBeVisible();
    });
  },
};

// 加载中 — 对应前端 BDD 的 Loading 状态场景
export const Loading: StoryObj = {
  parameters: {
    msw: {
      handlers: [
        http.get('/api/v1/articles', async () => {
          await delay('infinite');
        }),
      ],
    },
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    expect(canvas.getByTestId('loading-skeleton')).toBeVisible();
  },
};
```

### 6.4 Mock 工厂

```typescript
// test-utils/mocks.ts
function mockArticle(overrides?: Partial<Article>): Article {
  return {
    id: '1',
    title: '默认文章标题',
    content: '默认内容',
    author: mockUser(),
    createdAt: '2026-06-03',
    ...overrides,
  };
}

function mockUser(overrides?: Partial<User>): User {
  return {
    id: '1',
    name: '测试用户',
    avatar: '/default-avatar.png',
    ...overrides,
  };
}
```

---

## 七、测试数据管理（规则）

| 方案 | 说明 |
|------|------|
| Mock 工厂 | `mockEntity()` 函数生成符合 schema 的数据，override 差异字段 |
| MSW Handler | 按前端 BDD 场景定义不同的 Mock 响应（成功、失败、超时、空数据） |
| Storybook Args | 每个 Story 对应一组测试数据，通过 args 传入 |
| Storybook Parameters | MSW handlers 通过 parameters.msw 按场景配置 |

---

## 八、测试质量（约束）

每个 Story 必须满足：

- **独立** — 不依赖其他 Story 的执行结果
- **可重复** — 多次运行结果相同，不依赖时间、随机数、外部服务
- **可追溯** — Story name 对应前端 BDD 场景名称
- **状态完整** — 组件的所有 UI 状态都有对应的 Story（Idle / Loading / Success / Error / Empty）
- **1:1 映射** — 有 .vue 文件就有 .stories.ts 文件，不存在例外

做不到的，说明组件或 Story 写得不完整，需要补充。

---

## 九、测试完整性（约束）

每个组件的 Story 必须覆盖三种类型：

- **正向** — 正常渲染和交互成功
- **异常** — API 失败、网络错误、数据格式异常
- **边界** — 空数据、超长文本、Loading 状态、禁用状态

三缺一，Story 不完整。

---

## 十、输出（约束）

前端 CDD 阶段完成后必须产出以下内容：

### Story 文件

每个 Vue 组件对应的 .stories.ts 文件，覆盖所有 UI 状态和交互场景。

### 组件文件

通过 Story-组件-测试循环产生的 Vue 组件代码，所有 Story 通过。

### 覆盖映射

每个 Story 对应前端 BDD 的哪条场景，无法追溯的标注来源。

### 总结

每次输出必须包含一份**总结**，供用户快速审核：

- 生成了多少个 Story 文件、多少个 Story
- 每个组件覆盖了多少场景（正向 / 异常 / 边界各多少）
- Atomic Design 分层情况（各层多少个组件）
- 与前端 BDD 场景的覆盖情况（是否每条都有对应）
- 一份快速确认检查清单

### 待确认问题

代码注释或文档中附**待确认问题清单**，每条问题包含：

- 问题描述
- 为什么需要确认（BDD 场景歧义、组件层级不确定、Mock 行为不确定）
- 建议的默认选项

---

## 十一、完成定义（约束）

满足以下所有条件后，前端 CDD 阶段结束：

- UI 设计规范 + 状态定义阶段（06）的 HTML 原型已获用户确认
- 每个 Vue 组件都有对应的 Story 文件（1:1 映射，无例外）
- 每条前端 BDD 场景都有对应的 Story
- 每个 Story 的交互测试都通过
- 组件视觉效果还原了 HTML 原型中用户确认的效果
- 组件按 Atomic Design 层级正确组织
- 组件在所有 UI 状态下的渲染正确（Idle / Loading / Success / Error / Empty）
- 待确认问题已全部被用户回答（或用户明确跳过）
- 总结中的检查清单已获得用户确认

---

## 十二、禁止事项（约束）

前端 CDD 阶段禁止：

- 跳过 Story 直接写组件
- 在 Story 中定义接口契约（Mock 响应结构由 API 契约阶段决定）
- Story 之间产生状态依赖
- 修改 Story 来适配组件（Story 来源是前端 BDD，组件应适配 Story）
- .vue 文件没有对应的 .stories.ts 文件
- 不按 Atomic Design 层级组织组件

---

## 十三、测试日志输出规范（强约束）

每次运行 Storybook 测试都必须留存完整日志，用于排查问题和追溯历史。

### 输出路径确认（强约束）

**AI 首次为当前项目生成 Story 时，必须先向用户确认：**

- 日志输出到哪个目录？默认 `.test-output/`（项目根目录下）
- 用户可指定任意路径

确认后路径写入 `.test-output/_config.json`，后续运行脚本和框架配置均读取此文件。

```json
// .test-output/_config.json
{
  "cddOutputDir": ".test-output/cdd",
  "confirmedAt": "2026-06-03T150000",
  "confirmedBy": "user"
}
```

### 目录结构

```
{outputDir}/                               ← 用户确认的路径，gitignored + AI-excluded
└── cdd/
    ├── run.sh                             ← 一键运行脚本（见下方）
    ├── BaseButton/                        ← Story 名称 = 目录名
    │   ├── 2026-06-03T150000.log          ← 每次运行一份，时间戳命名，不覆盖
    │   ├── 2026-06-03T150000-summary.json
    │   ├── 2026-06-04T100000.log
    │   ├── 2026-06-04T100000-summary.json
    │   └── screenshots/                   ← 失败截图
    │       └── 2026-06-04T100000-failure.png
    └── _index.json                        ← 所有 Story 的运行索引
```

### 日志文件说明

| 文件 | 内容 | 用途 |
|------|------|------|
| `{timestamp}.log` | Storybook test-runner 完整输出 | 人读：排查渲染/交互失败 |
| `{timestamp}-summary.json` | Story 名称、通过/失败数、失败 Story 列表、截图路径 | 机器读：快速定位 |
| `screenshots/` | 失败 Story 的截图 | 视觉对比 |
| `_index.json` | 所有 Story 的最近运行记录汇总 | 全局概览 |

`summary.json` 结构：

```json
{
  "story": "BaseButton",
  "timestamp": "2026-06-03T150000",
  "total": 5,
  "passed": 4,
  "failed": 1,
  "durationMs": 3200,
  "failedStories": ["Disabled"],
  "screenshots": ["screenshots/2026-06-03T150000-Disabled-failure.png"]
}
```

### 运行脚本（强约束）

**每个测试类型的输出目录下必须有一个 `run.sh`**，用户可直接执行，不依赖 IDE。

`{outputDir}/cdd/run.sh`：

```bash
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H%M%S")

# 读取配置
CONFIG_FILE="$SCRIPT_DIR/../_config.json"
if [ -f "$CONFIG_FILE" ]; then
  OUTPUT_DIR=$(python3 -c "import json; print(json.load(open('$CONFIG_FILE'))['cddOutputDir'])" 2>/dev/null || echo ".test-output/cdd")
else
  OUTPUT_DIR=".test-output/cdd"
fi

echo "=== CDD 测试运行 ${TIMESTAMP} ==="
echo "输出目录: ${OUTPUT_DIR}"
echo ""

# 确保 Storybook 已启动（后台）
if ! curl -s http://localhost:6006 > /dev/null 2>&1; then
  echo "启动 Storybook..."
  npm run storybook &
  STORYBOOK_PID=$!
  sleep 5
fi

# 运行测试
npx test-storybook 2>&1 | tee "${OUTPUT_DIR}/_raw-${TIMESTAMP}.log"

EXIT_CODE=${PIPESTATUS[0]}

# 整理日志（按 Story 拆分）
node scripts/cdd-log-organizer.mjs "${OUTPUT_DIR}/_raw-${TIMESTAMP}.log" "${OUTPUT_DIR}"

# 清理原始文件
rm -f "${OUTPUT_DIR}/_raw-${TIMESTAMP}.log"

# 关闭 Storybook（如果是脚本启动的）
if [ -n "${STORYBOOK_PID:-}" ]; then
  kill $STORYBOOK_PID 2>/dev/null || true
fi

echo ""
echo "=== 测试完成，退出码: ${EXIT_CODE} ==="
echo "日志已写入: ${OUTPUT_DIR}/{StoryName}/${TIMESTAMP}.log"
echo "最新日志:"
ls -lt "${OUTPUT_DIR}"/*/"${TIMESTAMP}"*.log 2>/dev/null | head -5

exit ${EXIT_CODE}
```

**单独运行某个 Story：**

```bash
bash .test-output/cdd/run.sh BaseButton
```

### 隔离规则（强约束）

- 输出目录必须加入 `.gitignore`
- 输出目录必须加入 **当前使用的 AI 工具的忽略文件**（`.cursorignore` / `.claudeignore` / `.aiderignore` 等，按实际使用的工具配置）
- AI 默认不读输出目录
- 用户明确要求"查看 CDD 日志"时，AI 只读 `summary.json`，不读完整 `.log` 文件（除非用户指定）

---

## 十四、产物保护（约束）

Story 和组件测试一旦生成，进入锁定状态：

- **只增不改** — 允许追加新 Story 和测试用例，禁止修改或删除已有 Story 和测试用例
- **修改需特定技能** — 确需修改已有 Story 或测试时，必须使用专门的修改技能（而非在开发过程中随意改动），并记录修改原因
- **锁定范围** — Story 的 args、play 函数中的断言、Mock 工厂的默认值均受保护；Storybook 配置和装饰器的调整不在此限制内

---

## 十五、完整性检查

### 15.1 覆盖检查（约束）

每条前端 BDD 场景，必须至少对应一个 Story。有 BDD 场景但没有对应 Story 的，属于遗漏。

### 15.2 1:1 检查（强约束）

每个 .vue 文件必须有对应的 .stories.ts 文件。有组件但没有 Story 的，属于遗漏。

### 15.3 层级检查（约束）

每个组件必须放在正确的 Atomic Design 层级目录。放错层级的，需要调整：
- 包含业务逻辑的组件不应放在 atoms/
- 只做 UI 渲染的组件不应放在 organisms/
- 定义布局的组件应放在 templates/

### 15.4 状态覆盖检查（约束）

每个组件的 Story 必须覆盖其所有 UI 状态。常见必覆盖状态：
- 有数据的正常渲染
- 无数据的空状态
- 加载中的 Loading 状态
- 出错的 Error 状态

---

## 附录：转 Skill 约束分级（注入指引）

> 本附录写给"把本文档转成 skill"的人和注入脚本看，不是流程本身的一环。
> 目的是防止转 skill 时把"方向性引导"拍平成"命令式硬约束"，导致 AI 被限制太死。

本文档的内容分两类，转 skill 注入时必须保留这个区分：

### HARD（硬约束 — 违反即流程失效，skill 必须强制卡死）

- 前置：06 的 HTML 原型已获用户确认，不确认不进 CDD
- 每个 .vue 文件必须有对应的 .stories.ts 文件（1:1 映射，无例外）
- 每条前端 BDD 场景对应一个完整的 Story-组件-测试循环，禁止跳过 Story 直接写组件
- 组件必须按 Atomic Design 层级正确组织，包含业务逻辑的不放 atoms、纯 UI 不放 organisms
- 遵循 Vue3 + Storybook 规范（play 函数 + interaction testing + MSW 拦截 + 命名规范）
- 组件视觉效果必须还原已确认 HTML 原型，覆盖所有 UI 状态
- 产物保护：Story 和测试只增不改，args / play 断言 / Mock 工厂默认值受保护，确需改用专门修改技能
- 禁止在 Story 中定义接口契约（Mock 响应结构由契约阶段决定）、禁止改 Story 适配组件
- 待确认问题须全部被回答，总结检查清单须获用户确认

### GUIDE（引导规则 — 给方向，AI 按语境自由组织，禁止固化为唯一模板）

- 六节的各层 Story 写法、Mock 工厂是示例，怎么写按组件调整
- 整理阶段的提取标准同后端 TDD（两处以上复用或可读性明显提升），不抽离无意义 composable、不过度抽象
- 正向 / 异常 / 边界是最小覆盖，交互越复杂 Story 越多
- 扩展 Story 的发现来源（规则 7）是方向提示
- 分层判断标准是原则，具体组件归哪层按内容判断

### 注入原则（写 skill 注入文本时遵守）

- 注入时保留 HARD / GUIDE 标记，不要把 GUIDE 类内容改写成"必须""禁止"的命令句式
- "Story 写法示例""整理边界"这类注入时强调"按判断原则走"，示例标注"是示例不是模板"
- 1:1 映射、Atomic Design 分层、产物保护、原型确认前置属于 HARD，注入时保留完整；宁可注入少而精的 HARD 列表 + 一句"其余按文档精神发挥"，也不要把全文压缩成几十条命令式条款

---
