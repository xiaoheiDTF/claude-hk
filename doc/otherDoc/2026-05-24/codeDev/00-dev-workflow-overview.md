# 软件开发工作流完整指南

> **创建时间：** 2026-05-24  
> **所属模块：** codeDev / 概述  
> **简述：** 从写代码到部署的完整研发流程总览，覆盖Linter、测试、覆盖率、PR、CI/CD、Code Review、合并与部署九大环节。

---

## 一、流程全景图

```
┌─────────────┐     ┌──────────────┐     ┌──────────────────────┐
│  1. 写代码   │ ──▶ │ 2. Linter检查 │ ──▶ │ 3. 单元测试 + 集成测试 │
└─────────────┘     └──────────────┘     └──────────────────────┘
                                                  │
                                                  ▼
┌─────────────┐     ┌──────────────────┐   ┌──────────────┐
│ 9. 部署上线  │ ◀── │ 8. 合并到主干     │ ◀─│ 7. 代码审查   │
└─────────────┘     └──────────────────┘   └──────────────┘
       ▲                                              │
       │          ┌──────────────────┐                │
       └──────────│ 6. CI/CD自动检查  │◀───────────────┘
                  └──────────────────┘
                           │
                           ▼
                  ┌──────────────────┐
                  │ 5. 提交PR        │
                  │   (Conventional  │
                  │    Commits规范)  │
                  └──────────────────┘
                           │
                           ▼
                  ┌──────────────────┐
                  │ 4. 覆盖率 ≥ 95%  │
                  └──────────────────┘
```

---

## 二、各环节职责说明

| 步骤 | 环节 | 核心目标 | 负责人 |
|------|------|----------|--------|
| 1 | 写代码 | 完成功能开发或Bug修复 | 开发者 |
| 2 | Linter检查 | 统一代码风格，提前发现低级错误 | 开发者（本地）+ 工具 |
| 3 | 单元测试 + 集成测试 | 验证代码正确性与模块协作 | 开发者 |
| 4 | 覆盖率 ≥ 95% | 确保关键逻辑被充分测试 | 开发者 + 工具 |
| 5 | 提交PR | 按规范提交变更，触发自动化流程 | 开发者 |
| 6 | CI/CD自动检查 | 在标准化环境中验证所有检查项 | CI流水线 |
| 7 | 代码审查 | 人工审核代码质量、设计、安全性 | Reviewer |
| 8 | 合并到主干 | 将经过验证的代码合入主分支 | Maintainer |
| 9 | 部署上线 | 自动化或半自动化发布到生产环境 | CI/CD |

---

## 三、流程执行原则

### 3.1 本地"门禁"原则

开发者在提交代码前，必须在本地完成以下检查：
- [ ] `gofmt` / `golangci-lint` 通过
- [ ] `go test ./...` 单元测试全部通过
- [ ] `go test -tags=integration ./...` 集成测试全部通过
- [ ] `go test -cover` 覆盖率 ≥ 95%
- [ ] Commit message 符合 Conventional Commits 规范

> **为什么要在本地先做？**  
> 避免将明显的问题推送到远程，浪费CI资源和他人的Review时间。

### 3.2 红色即停原则

CI/CD流水线中的任何一环失败，PR都不能合并：
- Linter检查失败 → 禁止合并
- 单元测试失败 → 禁止合并
- 集成测试失败 → 禁止合并
- 覆盖率不达标 → 禁止合并
- 安全扫描发现高危漏洞 → 禁止合并

### 3.3 小而美的PR原则

- 单次PR的代码变更量控制在 **300行以内**（核心业务逻辑）
- 一个PR只解决 **一个问题**（一个Bug、一个功能点）
- PR描述清晰，关联对应的Issue

---

## 四、Go项目推荐工具链

| 环节 | 推荐工具 | 说明 |
|------|----------|------|
| Linter | `golangci-lint` | 聚合40+ linter的一站式工具 |
| 单元测试 | Go内置 `testing` + `testify` | 标准库 + 断言库 |
| Mock | `gomock` / `mockery` / 手写接口 | 依赖注入 + 接口模拟 |
| 覆盖率 | `go test -cover` + `gocov` | 内置支持 + 可视化 |
| 提交规范 | `commitlint` + `husky` | 校验提交信息格式 |
| CI/CD | GitHub Actions | 与GitHub深度集成 |
| 代码审查 | GitHub Pull Request | 内联评论、Approve机制 |

---

## 五、快速参考：本地检查命令

```bash
# 1. Linter检查
golangci-lint run ./...

# 2. 运行单元测试（排除集成测试）
go test -short ./...

# 3. 运行全部测试（含集成测试）
go test ./...

# 4. 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# 5. 查看HTML覆盖率报告
go tool cover -html=coverage.out -o coverage.html

# 6. 检查覆盖率是否≥95%
go test -cover ./... | awk '{if($5+0 < 95) exit 1}'
```

---

## 六、延伸阅读

本系列其他文档：
- [01-linter-check.md](./01-linter-check.md) — Linter检查详解
- [02-testing/unit-test-guide.md](./02-testing/unit-test-guide.md) — 单元测试最佳实践
- [02-testing/integration-test-guide.md](./02-testing/integration-test-guide.md) — 集成测试策略
- [02-testing/coverage-95-guide.md](./02-testing/coverage-95-guide.md) — 如何达成95%覆盖率
- [03-git-conventional-commits.md](./03-git-conventional-commits.md) — 约定式提交规范
- [04-cicd-github-actions.md](./04-cicd-github-actions.md) — CI/CD流水线配置
- [05-code-review-guide.md](./05-code-review-guide.md) — 代码审查指南
