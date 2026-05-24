# CI/CD 流水线配置指南（GitHub Actions）

> **创建时间：** 2026-05-24  
> **所属模块：** codeDev / CI/CD  
> **简述：** 详解CI/CD概念、GitHub Actions工作流配置、Go项目完整的CI流水线模板，以及自动部署策略。

---

## 一、什么是 CI/CD？

| 缩写 | 全称 | 含义 |
|------|------|------|
| **CI** | Continuous Integration（持续集成） | 代码变更自动构建、测试 |
| **CD** | Continuous Delivery（持续交付） | 代码自动部署到预发布/生产环境 |
| **CD** | Continuous Deployment（持续部署） | 代码自动发布到生产环境 |

### CI/CD 解决的问题

- ✅ 人工构建容易出错 → **自动化**
- ✅ "在我机器上能跑" → **标准化环境**
- ✅ 手动测试耗时 → **自动化测试**
- ✅ 发布流程不透明 → **可追溯、可回滚**

---

## 二、GitHub Actions 基础

### 2.1 核心概念

| 概念 | 说明 |
|------|------|
| **Workflow** | 工作流，定义在 `.github/workflows/*.yml` |
| **Job** | 任务，同一Workflow可包含多个Job |
| **Step** | 步骤，Job内的执行单元 |
| **Action** | 可复用的动作（如 `actions/checkout`） |
| **Runner** | 执行环境（GitHub托管或自托管） |

### 2.2 基本结构

```yaml
name: CI Pipeline

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go build ./...
```

---

## 三、Go项目完整CI流水线

### 3.1 完整工作流

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main, develop]
    paths-ignore:
      - '**.md'
      - 'doc/**'
  pull_request:
    branches: [main]
    paths-ignore:
      - '**.md'

jobs:
  # ========== Job 1: Linter检查 ==========
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v1.58.0
          args: --timeout=5m --config=.golangci.yml

  # ========== Job 2: 单元测试 + 覆盖率 ==========
  unit-test:
    name: Unit Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true

      - name: Run unit tests
        run: go test -race -short -coverprofile=unit.out ./...

      - name: Upload unit coverage
        uses: actions/upload-artifact@v4
        with:
          name: unit-coverage
          path: unit.out

  # ========== Job 3: 集成测试 ==========
  integration-test:
    name: Integration Tests
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true

      - name: Run integration tests
        env:
          TEST_DB_HOST: localhost
          TEST_DB_PORT: 5432
          TEST_REDIS_HOST: localhost
          TEST_REDIS_PORT: 6379
        run: go test -race -tags=integration -coverprofile=integration.out ./...

      - name: Upload integration coverage
        uses: actions/upload-artifact@v4
        with:
          name: integration-coverage
          path: integration.out

  # ========== Job 4: 覆盖率合并与检查 ==========
  coverage:
    name: Coverage Check
    needs: [unit-test, integration-test]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Download unit coverage
        uses: actions/download-artifact@v4
        with:
          name: unit-coverage

      - name: Download integration coverage
        uses: actions/download-artifact@v4
        with:
          name: integration-coverage

      - name: Merge coverage
        run: |
          go install github.com/wadey/gocovmerge@latest
          gocovmerge unit.out integration.out > coverage.out

      - name: Check coverage threshold
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
          echo "Total coverage: ${COVERAGE}%"
          if (( $(echo "${COVERAGE} < 95.0" | bc -l) )); then
            echo "Coverage ${COVERAGE}% is below 95% threshold!"
            exit 1
          fi

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v4
        with:
          file: ./coverage.out
          fail_ci_if_error: true

  # ========== Job 5: 构建验证 ==========
  build:
    name: Build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true

      - name: Build binary
        run: go build -o app ./cmd/server

      - name: Build Docker image
        run: docker build -t app:test .
```

### 3.2 提交信息校验

```yaml
# .github/workflows/pr-title-check.yml
name: PR Title Check

on:
  pull_request:
    types: [opened, edited, synchronize]

jobs:
  check-title:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Check PR title
        uses: amannn/action-semantic-pull-request@v5
        with:
          types: |
            feat
            fix
            docs
            style
            refactor
            perf
            test
            chore
            ci
            build
            revert
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## 四、CD 部署配置

### 4.1 Docker 镜像构建与推送

```yaml
# .github/workflows/cd.yml
name: CD

on:
  push:
    tags:
      - 'v*'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ghcr.io/${{ github.repository }}
          tags: |
            type=ref,event=tag
            type=sha

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

### 4.2 多环境部署

```yaml
  deploy-staging:
    name: Deploy to Staging
    needs: [lint, test, build]
    if: github.ref == 'refs/heads/develop'
    runs-on: ubuntu-latest
    environment: staging
    steps:
      - name: Deploy to Staging
        run: |
          echo "Deploying to staging..."
          # 调用部署脚本或API

  deploy-production:
    name: Deploy to Production
    needs: [lint, test, build]
    if: startsWith(github.ref, 'refs/tags/v')
    runs-on: ubuntu-latest
    environment: production
    steps:
      - name: Deploy to Production
        run: |
          echo "Deploying to production..."
```

---

## 五、高级配置

### 5.1 矩阵构建（多Go版本/多OS）

```yaml
  test-matrix:
    strategy:
      matrix:
        go-version: ['1.21', '1.22']
        os: [ubuntu-latest, windows-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}
      - run: go test ./...
```

### 5.2 并发控制

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

### 5.3 缓存优化

```yaml
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true  # 自动缓存go module
```

### 5.4 安全扫描

```yaml
      - name: Run Gosec Security Scanner
        uses: securego/gosec@master
        with:
          args: '-no-fail -fmt sarif -out results.sarif ./...'
```

---

## 六、完整的PR检查流程

当开发者提交PR时，CI自动执行：

```
PR 创建/更新
    │
    ▼
┌──────────────┐
│ 1. PR标题校验 │ ← 必须符合 Conventional Commits
└──────────────┘
    │
    ▼
┌──────────────┐
│ 2. Linter检查 │ ← golangci-lint
└──────────────┘
    │
    ▼
┌──────────────┐
│ 3. 单元测试   │ ← go test -short ./...
└──────────────┘
    │
    ▼
┌──────────────┐
│ 4. 集成测试   │ ← go test -tags=integration ./...
└──────────────┘
    │
    ▼
┌──────────────┐
│ 5. 覆盖率检查 │ ← 必须 ≥95%
└──────────────┘
    │
    ▼
┌──────────────┐
│ 6. 构建验证   │ ← go build + Docker build
└──────────────┘
    │
    ▼
┌──────────────┐
│ 7. 安全扫描   │ ← gosec
└──────────────┘
    │
    ▼
   ✅ 全部通过 → 可合并
   ❌ 任一失败 → 禁止合并
```

---

## 七、GitHub Branch Protection 配置

在仓库设置中启用分支保护规则：

1. **Settings → Branches → Add rule**
2. 选择 `main` 分支
3. 启用以下选项：
   - ✅ **Require a pull request before merging**
   - ✅ **Require status checks to pass before merging**
     - 添加必需检查：`Lint`, `Unit Tests`, `Coverage Check`, `Build`
   - ✅ **Require conversation resolution before merging**
   - ✅ **Require signed commits**（可选）
   - ✅ **Include administrators**

---

## 八、Checklist

- [ ] 创建 `.github/workflows/ci.yml`
- [ ] 配置 golangci-lint 检查
- [ ] 配置单元测试和集成测试分离
- [ ] 配置覆盖率 ≥95% 的门禁
- [ ] 配置 PR 标题格式校验
- [ ] 配置多环境部署（staging / production）
- [ ] 启用分支保护规则
- [ ] 配置 secrets（如 Docker Registry Token）
- [ ] 设置通知（失败时通知Slack/钉钉）
