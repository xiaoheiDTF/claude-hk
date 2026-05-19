# Claude-Code / Workflows / E2e-Testing

> 来源: claudecn.com

# E2E 测试工作流：Playwright 关键旅程与产物管理

端到端（E2E）测试的价值不在“覆盖率数字”，而在于它能把关键用户旅程跑通，并在失败时产出可追溯证据（截图/录像/trace）。一个可复用的落地框架可以概括为三件事：**旅程清单、稳定性、产物管理**。

## 什么时候必须上 E2E

推荐至少覆盖以下“高风险旅程”：

- 登录/鉴权/退出
- 下单/支付/交易等资金相关流程
- 创建/编辑/删除等关键数据写入流程
- 搜索/筛选等用户高频入口（尤其涉及异步与缓存）
## 一条推荐的 E2E 流程（可直接照做）

- 先写“用户旅程”与验收点（不要直接写代码）
- 生成或维护 Playwright 测试
- 本地跑 3–5 次确认稳定性
- CI 里跑（失败保留 artifacts）
- 发现 flaky：先隔离（quarantine），再修（不要拖垮主线 CI）
## Page Object Model（POM）建议

`e2e-runner` 提供了 POM 的示例结构（节选），关键点是把“定位器与操作”收敛到页面对象里，避免测试文件里到处写 locator：

```typescript
export class MarketsPage {
  async goto() {
    await this.page.goto('/markets')
    await this.page.waitForLoadState('networkidle')
  }
}
```

## 稳定性：用“等待条件”替代“硬睡眠”
`e2e-runner` 对 flaky 的典型原因与修法是：

- 竞态：优先用 Playwright 自带 auto-wait 的 locator().click()
- 网络时序：用 waitForResponse 等明确条件
- 动画：避免依赖动画时序（必要时禁用/降低动画影响）
示例（节选）：

```typescript
await page.waitForResponse(resp => resp.url().includes('/api/markets'))
```

## 产物（Artifacts）：失败时要能“复现当时发生了什么”
`e2e-runner` 建议在失败时保留：

- screenshot
- video
- trace（用于 step-by-step 回放）
- HTML report / JUnit XML（用于 CI 展示与聚合）
这能显著降低“CI 红了但不知道为什么”的排障成本。

## Flaky 隔离（Quarantine）建议

`e2e-runner` 给出了两种隔离方式（节选）：

```typescript
test.fixme(true, 'Test is flaky - Issue #123')
```

或在 CI 环境跳过：

```typescript
test.skip(process.env.CI, 'Test is flaky in CI - Issue #123')
```

## 下一步

- 把 E2E 纳入团队闭环：见 团队质量门禁：Plan → TDD → Build Fix → Review
- 构建失败优先增量修复：见 构建排障：最小 diff 的增量修复
## 参考
无
