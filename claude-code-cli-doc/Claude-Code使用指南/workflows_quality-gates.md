# Claude-Code / Workflows / Quality-Gates

> 来源: claudecn.com

# 团队质量门禁：Plan → TDD → Build Fix → Review

当团队把 Claude Code 用进日常开发，真正决定“能不能长期用”的不是单次产出，而是**可控性**：是否能稳定地把改动做对、做安全、做可回滚。

把方法论梳理成一条团队可执行的闭环流程（不依赖特定技术栈）：**先规划，再用测试驱动实现，构建失败就增量修复，最后做安全与质量审查**。

## 0) 先把“验收方式”写出来

无论你用不用 Claude Code，先明确一个可执行的验收方式，后续所有步骤才有落点：

- 单元测试：npm test / pnpm test / go test ./...
- 构建：npm run build / cargo build / make build
- 端到端：npx playwright test
- 关键页面：某个 URL + 关键交互点
## 1) Plan：先把需求与风险钉住

推荐两种做法：

- 轻量：在对话里要求“先列计划，不要改代码”
- 结构化：使用 Plan Mode（见 计划模式）
建议输出至少包含：

- 需求复述（确认理解一致）
- 风险点（权限、兼容、数据迁移、性能）
- 分阶段步骤（每步可验证）
- 回滚/降级思路（如果适用）
如果团队希望把 Plan 固化成一条“必须走的流程”，可以用项目级 `/plan` 自定义命令来强制“先计划后执行”（见 [自定义命令](https://claudecn.com/docs/claude-code/advanced/custom-commands/)）。

## 2) TDD：测试先行，覆盖边界与错误路径

你不一定要从第一天就要求“覆盖率 80%”，但建议至少把这两条变成团队共识：

- 新功能/修 Bug：先写能复现的测试，再实现代码
- 关键路径：必须覆盖错误处理与边界条件（空值、超限、权限不足、服务不可用）
从社区实践里提炼的测试结构：

- 单元测试：验证函数/组件行为
- 集成测试：验证 API/数据库/服务交互
- E2E（Playwright）：验证关键用户旅程，并产出截图/trace 等证据
E2E 的更完整落地做法（旅程清单、稳定性、产物管理、flaky 隔离）见：[E2E 测试工作流：Playwright 关键旅程与产物管理](https://claudecn.com/docs/claude-code/workflows/e2e-testing/)。

如果你希望把“缺口补测”变成固定动作，可以做一个 `/test-coverage` 类命令：跑覆盖率 → 找低覆盖文件 → 只补未覆盖分支 → 复跑测试确认。

## 3) Build Fix：构建失败就“增量修一个”

构建或类型检查失败时，最稳的节奏是：

- 先跑一次构建拿到完整报错
- 按文件/严重程度分组
- 一次只修一个错误
- 每修一次就复跑构建/检查，确认没有引入连锁问题
社区仓库的思路是把它固化成 `/build-fix`：**“修一个、跑一次、验证解决”**，避免一次性改太多导致回溯困难。

进一步的“最小 diff 排障”实践与常见修法见：[构建排障：最小 diff 的增量修复](https://claudecn.com/docs/claude-code/workflows/build-troubleshooting/)。

## 4) Review：安全与质量审查必须“按严重度阻断”

团队落地时建议把审查输出固定成四档（示例）：

- CRITICAL：必须修复（密钥泄漏、鉴权绕过、注入风险、路径遍历等）
- HIGH：应当修复（严重可靠性问题、重要边界缺失）
- MEDIUM：建议修复（性能风险、可维护性问题）
- LOW：提示（风格、可读性、命名）
审查清单建议覆盖：

- Secrets：是否硬编码密钥/令牌
- 输入校验：是否对外部输入做 schema/白名单校验
- 注入：SQL/命令注入/模板注入
- XSS/CSRF：是否有可执行注入面、是否有 CSRF 防护
- 授权：敏感操作是否先做权限检查
- 错误信息：是否泄露敏感细节
你可以把“审查”做成固定命令（例如 `/code-review`），并在团队里明确：存在 CRITICAL/HIGH 就不合并。

可直接复用的审查流程模板见：[代码审查工作流：分级输出与合并门禁](https://claudecn.com/docs/claude-code/workflows/code-review/)。

涉及输入/鉴权/敏感数据时建议额外做一次安全审查：见 [安全审查工作流：从 Secrets 到 OWASP](https://claudecn.com/docs/claude-code/workflows/security-review/)。

如果你要做“删代码/去重/清理依赖”，建议单独走一条可回滚流程：见 [安全清理死代码：工具分析 + 删除日志](https://claudecn.com/docs/claude-code/workflows/refactor-clean/)。

## 5) 一条推荐的团队闭环（可直接照做）

- /plan 或 Plan Mode：先确认方案与验收
- /tdd：测试先行，实现最小可用
- /build-fix：构建失败就增量修复
- /code-review：按严重度输出审查报告并阻断高风险
- git 合并前：跑最小验证集（测试 + 构建 + 关键 E2E）
如果你希望把这些流程“工程化”（目录结构、共享配置、最小文件清单），看：[团队 Starter Kit](https://claudecn.com/docs/claude-code/advanced/starter-kit/)。

## 参考

无
