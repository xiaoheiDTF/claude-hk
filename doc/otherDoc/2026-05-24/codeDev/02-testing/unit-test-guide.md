# Go单元测试最佳实践

> **创建时间：** 2026-05-24  
> **所属模块：** codeDev / 测试  
> **简述：** 详解Go语言单元测试的概念、编写方法、表格驱动测试、Mock技术以及 testify 断言库的使用。

---

## 一、什么是单元测试？

**单元测试（Unit Test）** 是对软件中最小可测试单元（通常是函数或方法）进行验证的测试。

### 单元测试的核心特征

| 特征 | 说明 |
|------|------|
| **隔离性** | 只测试一个函数/方法，不依赖外部系统（数据库、API、文件系统） |
| **快速性** | 应在毫秒级完成，整个项目单元测试应在秒级完成 |
| **确定性** | 相同输入永远产生相同输出，不受环境、时间、网络影响 |
| **自动化** | 无需人工干预，CI中自动运行 |

### 单元测试 vs 集成测试

| 维度 | 单元测试 | 集成测试 |
|------|----------|----------|
| 测试范围 | 单个函数/模块 | 多个模块协作 |
| 外部依赖 | Mock/Stub | 真实依赖（数据库、Redis等） |
| 执行速度 | 极快（毫秒） | 较慢（秒级） |
| 失败定位 | 精确到函数 | 定位范围较大 |
| 文件命名 | `*_test.go` | `*_test.go` + `//go:build integration` |

---

## 二、Go测试基础

### 2.1 文件命名与位置

- 测试文件：`xxx_test.go`
- 与被测代码放在**同一目录**
- 测试包名：通常使用 `xxx`（白盒测试）或 `xxx_test`（黑盒测试）

```
calculator/
├── calculator.go      # 被测代码
└── calculator_test.go # 测试代码
```

### 2.2 基本测试函数

```go
// 被测代码：calculator.go
package calculator

func Add(a, b int) int {
    return a + b
}

func Divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}
```

```go
// 测试代码：calculator_test.go
package calculator

import "testing"

// 测试函数必须以 Test 开头，参数为 *testing.T
func TestAdd(t *testing.T) {
    result := Add(2, 3)
    if result != 5 {
        t.Errorf("Add(2, 3) = %d; want 5", result)
    }
}

func TestDivide(t *testing.T) {
    result, err := Divide(10, 2)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != 5 {
        t.Errorf("Divide(10, 2) = %d; want 5", result)
    }
}

func TestDivideByZero(t *testing.T) {
    _, err := Divide(10, 0)
    if err == nil {
        t.Error("expected error for division by zero, got nil")
    }
}
```

### 2.3 运行测试

```bash
# 运行当前目录测试
go test

# 运行所有子目录测试
go test ./...

# 运行特定测试函数
go test -run TestAdd

# 显示详细输出
go test -v

# 跳过集成测试（使用 -short 标志）
go test -short ./...

# 运行基准测试
go test -bench=.
```

---

## 三、表格驱动测试（Table-Driven Tests）

Go社区强烈推荐的模式，一个测试函数覆盖多组输入输出。

```go
func TestAddTableDriven(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 2, 3, 5},
        {"negative numbers", -2, -3, -5},
        {"mixed signs", -2, 3, 1},
        {"with zero", 0, 5, 5},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, result, tt.expected)
            }
        })
    }
}
```

**优势：**
- 新增测试用例只需添加一行数据
- 测试逻辑与数据分离
- 每个子测试独立运行，一个失败不影响其他

---

## 四、Testify 断言库

Go标准库的 `t.Errorf` 比较原始，推荐使用 `stretchr/testify`。

**安装：**
```bash
go get github.com/stretchr/testify
```

### 4.1 基本断言

```go
import "github.com/stretchr/testify/assert"

func TestWithTestify(t *testing.T) {
    assert.Equal(t, 5, Add(2, 3), "Add should return sum")
    assert.NotNil(t, someObj)
    assert.True(t, isValid)
    assert.NoError(t, err)
    assert.Contains(t, []string{"a", "b"}, "a")
}
```

### 4.2 require（失败即终止）

```go
import "github.com/stretchr/testify/require"

func TestWithRequire(t *testing.T) {
    result, err := SomeOperation()
    require.NoError(t, err)        // err != nil 时立即终止
    require.NotNil(t, result)       // nil 时立即终止
    assert.Equal(t, 42, result.ID)  // 继续执行
}
```

**`assert` vs `require`：**
- `assert`：失败记录但不终止，继续后续断言
- `require`：失败立即终止当前测试函数

---

## 五、Mock与依赖注入

单元测试要求隔离外部依赖，Go通过**接口（interface）**实现Mock。

### 5.1 定义接口

```go
// user_service.go
package service

type UserRepository interface {
    GetUser(id int) (*User, error)
    SaveUser(user *User) error
}

type UserService struct {
    repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
    return &UserService{repo: repo}
}

func (s *UserService) GetUserName(id int) (string, error) {
    user, err := s.repo.GetUser(id)
    if err != nil {
        return "", err
    }
    return user.Name, nil
}
```

### 5.2 手写Mock

```go
// mock_repository.go (测试文件中)
type mockUserRepository struct {
    getUserFunc func(id int) (*User, error)
}

func (m *mockUserRepository) GetUser(id int) (*User, error) {
    return m.getUserFunc(id)
}

func (m *mockUserRepository) SaveUser(user *User) error {
    return nil
}
```

### 5.3 使用Mock进行测试

```go
func TestUserService_GetUserName(t *testing.T) {
    tests := []struct {
        name       string
        mockFunc   func(id int) (*User, error)
        wantName   string
        wantErr    bool
    }{
        {
            name: "user found",
            mockFunc: func(id int) (*User, error) {
                return &User{ID: 1, Name: "Alice"}, nil
            },
            wantName: "Alice",
            wantErr:  false,
        },
        {
            name: "user not found",
            mockFunc: func(id int) (*User, error) {
                return nil, errors.New("user not found")
            },
            wantName: "",
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := &mockUserRepository{getUserFunc: tt.mockFunc}
            svc := NewUserService(mockRepo)
            
            name, err := svc.GetUserName(1)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.wantName, name)
            }
        })
    }
}
```

### 5.4 使用 Mock 生成工具

对于大型项目，手写Mock繁琐，可使用：

| 工具 | 命令 | 特点 |
|------|------|------|
| `mockgen` (gomock) | `mockgen -source=repo.go -destination=mock_repo.go` | Google出品，功能强大 |
| `mockery` | `mockery --all` | 自动生成，基于 testify |
| `minimock` | `minimock -i UserRepository` | 生成代码简洁 |

**gomock 示例：**
```bash
go install github.com/golang/mock/mockgen@latest
mockgen -source=user_service.go -destination=mocks/mock_user_repository.go -package=mocks
```

```go
import "github.com/golang/mock/gomock"

func TestWithGomock(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    mockRepo := mocks.NewMockUserRepository(ctrl)
    mockRepo.EXPECT().GetUser(1).Return(&User{Name: "Alice"}, nil)
    
    svc := NewUserService(mockRepo)
    name, _ := svc.GetUserName(1)
    assert.Equal(t, "Alice", name)
}
```

---

## 六、子测试与并行测试

### 6.1 子测试

```go
func TestComplexLogic(t *testing.T) {
    t.Run("subtest A", func(t *testing.T) {
        // 测试A逻辑
    })
    t.Run("subtest B", func(t *testing.T) {
        // 测试B逻辑
    })
}
```

### 6.2 并行测试

```go
func TestParallel(t *testing.T) {
    tests := []struct{ name string }{...}
    
    for _, tt := range tests {
        tt := tt // 捕获循环变量
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // 标记可并行执行
            // 测试逻辑
        })
    }
}
```

---

## 七、测试辅助函数与Setup/Teardown

### 7.1 TestMain

```go
func TestMain(m *testing.M) {
    // 全局Setup
    setup()
    
    code := m.Run()
    
    // 全局Teardown
    teardown()
    
    os.Exit(code)
}
```

### 7.2 辅助函数

```go
func mustCreateTempFile(t *testing.T, content string) string {
    t.Helper() // 标记为辅助函数，报错时跳过此函数栈帧
    
    f, err := os.CreateTemp("", "test-*")
    require.NoError(t, err)
    t.Cleanup(func() { os.Remove(f.Name()) })
    
    _, err = f.WriteString(content)
    require.NoError(t, err)
    f.Close()
    
    return f.Name()
}
```

---

## 八、单元测试 checklist

- [ ] 每个导出函数都有对应的测试
- [ ] 使用表格驱动测试覆盖边界条件
- [ ] 正常路径、错误路径、边界条件都测试到
- [ ] 外部依赖通过接口Mock
- [ ] 测试名清晰描述测试场景
- [ ] 使用 `t.Helper()` 标记辅助函数
- [ ] 使用 `t.Parallel()` 加速测试执行
- [ ] 避免测试之间的状态共享
