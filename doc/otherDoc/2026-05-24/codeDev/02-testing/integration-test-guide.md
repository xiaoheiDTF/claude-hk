# Go集成测试策略与实践

> **创建时间：** 2026-05-24  
> **所属模块：** codeDev / 测试  
**简述：** 介绍集成测试的概念、与单元测试的区别、Go中集成测试的编写方法、Build Tags隔离策略以及测试容器的使用。

---

## 一、什么是集成测试？

**集成测试（Integration Test）** 验证多个模块、组件或服务之间的协作是否正确的测试。

### 集成测试的核心特征

| 特征 | 说明 |
|------|------|
| **协作验证** | 关注模块间的接口和数据流 |
| **真实依赖** | 使用真实数据库、缓存、消息队列等外部依赖 |
| **环境要求** | 需要预置测试环境（Docker容器、测试数据库） |
| **执行较慢** | 通常秒级甚至分钟级 |

### 何时需要集成测试？

- 数据库访问层（DAO/Repository）的CRUD操作
- HTTP API端到端测试
- 消息队列的生产和消费
- 缓存读写一致性
- 第三方API调用
- 文件系统操作

---

## 二、单元测试 vs 集成测试 对比

| 维度 | 单元测试 | 集成测试 |
|------|----------|----------|
| **目标** | 验证单个函数正确性 | 验证模块协作正确性 |
| **依赖** | Mock/Stub 全部外部依赖 | 使用真实外部依赖 |
| **速度** | 毫秒级 | 秒级~分钟级 |
| **稳定性** | 极高 | 受环境因素影响 |
| **定位问题** | 精确到函数 | 定位到模块交互 |
| **CI运行** | 每次提交必跑 | 可配置为定时或PR时运行 |
| **代码标记** | 普通 `*_test.go` | `//go:build integration` |

---

## 三、Go集成测试的编写方法

### 3.1 使用 Build Tags 隔离

Go通过 **Build Tags** 将集成测试与单元测试分离，避免每次运行单元测试时触发耗时的集成测试。

```go
// user_repository_integration_test.go
//go:build integration
// +build integration

package repository

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestUserRepository_CreateAndGet(t *testing.T) {
    // 使用真实数据库连接
    db := setupTestDB(t)
    
    repo := NewUserRepository(db)
    
    user := &User{Name: "Alice", Email: "alice@example.com"}
    err := repo.Create(user)
    require.NoError(t, err)
    require.NotZero(t, user.ID)
    
    fetched, err := repo.GetByID(user.ID)
    require.NoError(t, err)
    require.Equal(t, user.Name, fetched.Name)
}
```

### 3.2 运行方式

```bash
# 只运行单元测试（默认）
go test ./...

# 只运行单元测试（显式）
go test -short ./...

# 运行集成测试（必须带 tag）
go test -tags=integration ./...

# 运行所有测试
go test -tags=integration ./...

# 在CI中运行集成测试
go test -v -tags=integration -race ./...
```

---

## 四、测试数据库策略

### 4.1 方案一：内存SQLite（轻量级）

适用于不涉及数据库特定特性的场景：

```go
import (
    "testing"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
    require.NoError(t, err)
    
    // 自动迁移表结构
    err = db.AutoMigrate(&User{}, &Order{})
    require.NoError(t, err)
    
    t.Cleanup(func() {
        sqlDB, _ := db.DB()
        sqlDB.Close()
    })
    
    return db
}
```

### 4.2 方案二：Testcontainers（推荐）

使用真实数据库（PostgreSQL、MySQL、Redis等）：

```go
import (
    "context"
    "testing"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupPostgresContainer(t *testing.T) *gorm.DB {
    ctx := context.Background()
    
    container, err := postgres.Run(ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    require.NoError(t, err)
    
    t.Cleanup(func() {
        container.Terminate(ctx)
    })
    
    connStr, _ := container.ConnectionString(ctx)
    db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
    require.NoError(t, err)
    
    db.AutoMigrate(&User{})
    
    return db
}
```

**安装：**
```bash
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
```

### 4.3 方案三：固定测试数据库

使用独立的测试数据库实例：

```go
func setupTestDB(t *testing.T) *gorm.DB {
    // 连接到固定的测试数据库
    dsn := "host=localhost user=test password=test dbname=test_db port=5432"
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    require.NoError(t, err)
    
    // 每个测试前清理数据
    db.Exec("TRUNCATE users, orders CASCADE")
    
    t.Cleanup(func() {
        db.Exec("TRUNCATE users, orders CASCADE")
    })
    
    return db
}
```

---

## 五、HTTP API 集成测试

### 5.1 使用 httptest

Go标准库 `net/http/httptest` 可启动临时HTTP服务器：

```go
//go:build integration
package handler

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestCreateUserHandler(t *testing.T) {
    router := setupTestRouter() // 初始化路由和依赖
    
    w := httptest.NewRecorder()
    body := strings.NewReader(`{"name":"Alice","email":"alice@example.com"}`)
    req, _ := http.NewRequest("POST", "/api/users", body)
    req.Header.Set("Content-Type", "application/json")
    
    router.ServeHTTP(w, req)
    
    assert.Equal(t, http.StatusCreated, w.Code)
    assert.Contains(t, w.Body.String(), "Alice")
}
```

### 5.2 端到端API测试

```go
func TestAPIEndToEnd(t *testing.T) {
    // 启动完整服务（或使用 httptest）
    server := httptest.NewServer(setupTestRouter())
    defer server.Close()
    
    client := &http.Client{}
    
    // 创建用户
    resp, err := client.Post(
        server.URL+"/api/users",
        "application/json",
        strings.NewReader(`{"name":"Bob"}`),
    )
    require.NoError(t, err)
    assert.Equal(t, 201, resp.StatusCode)
    
    // 查询用户
    resp, err = client.Get(server.URL + "/api/users/1")
    require.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}
```

---

## 六、集成测试的最佳实践

### 6.1 目录与命名约定

```
internal/
├── service/
│   ├── user_service.go
│   └── user_service_test.go          # 单元测试
├── repository/
│   ├── user_repository.go
│   └── user_repository_test.go       # 单元测试
└── integration/
    └── api_integration_test.go       # 集成测试（使用 build tag）
```

### 6.2 隔离策略

| 策略 | 实现方式 |
|------|----------|
| **Schema隔离** | 每个测试套件使用独立的数据库Schema |
| **事务隔离** | 每个测试在事务中执行，结束后回滚 |
| **数据清理** | 测试前后 TRUNCATE 相关表 |
| **容器隔离** | 每个测试启动独立的数据库容器 |

### 6.3 事务回滚模式

```go
func TestWithTransaction(t *testing.T) {
    db := setupTestDB(t)
    
    tx := db.Begin()
    defer tx.Rollback() // 测试结束后回滚
    
    repo := NewUserRepository(tx)
    
    // 执行测试...
    user := &User{Name: "Test"}
    err := repo.Create(user)
    require.NoError(t, err)
    
    // 断言...
    assert.NotZero(t, user.ID)
}
```

---

## 七、CI/CD 中集成测试配置

```yaml
name: Integration Tests

on:
  pull_request:
    branches: [ main ]

jobs:
  integration:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
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
        image: redis:7
        ports:
          - 6379:6379

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Run Integration Tests
        env:
          TEST_DB_HOST: localhost
          TEST_REDIS_HOST: localhost
        run: go test -v -tags=integration ./...
```

---

## 八、集成测试 checklist

- [ ] 使用 `//go:build integration` 标记集成测试文件
- [ ] 每个集成测试都有独立的测试数据准备和清理
- [ ] 优先使用 Testcontainers 保证环境一致性
- [ ] 集成测试失败时提供足够的上下文信息
- [ ] CI中配置好测试依赖的服务（数据库、缓存等）
- [ ] 集成测试不应依赖于执行顺序
- [ ] 为耗时操作设置合理的超时时间
