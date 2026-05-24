# 代码覆盖率实战：如何达到 ≥95%

> **创建时间：** 2026-05-24  
> **所属模块：** codeDev / 测试  
> **简述：** 详解Go代码覆盖率的概念、工具使用、提升策略，以及如何在CI中强制要求95%覆盖率。

---

## 一、什么是代码覆盖率？

**代码覆盖率（Code Coverage）** 衡量测试代码对被测代码的覆盖程度，通常以百分比表示。

### Go支持的覆盖率类型

| 类型 | 说明 | 命令 |
|------|------|------|
| **语句覆盖率** | 执行的语句占总语句的比例 | `go test -cover` |
| **分支覆盖率** | 执行的分支（if/switch）占总分支的比例 | 需额外工具 |
| **函数覆盖率** | 被调用的函数占总函数的比例 | 报告中可见 |

> Go内置工具默认统计**语句覆盖率**。

---

## 二、基础使用

### 2.1 生成覆盖率报告

```bash
# 查看覆盖率百分比
go test -cover ./...

# 生成覆盖率文件
go test -coverprofile=coverage.out ./...

# 查看函数级别覆盖率
go tool cover -func=coverage.out

# 生成HTML可视化报告
go tool cover -html=coverage.out -o coverage.html
```

### 2.2 排除不需要测试的代码

```go
// 整个文件排除覆盖率统计
//go:build ignore_coverage

// 单行排除
_ = someUntestableFunc() // coverage:ignore
```

更可靠的方式：在生成报告时排除

```bash
# 排除 generated 文件和 mock 文件
go test -coverprofile=coverage.out ./...
grep -v "_mock.go\|_gen.go" coverage.out > coverage.filtered.out
```

---

## 三、覆盖率可视化工具

### 3.1 gocov + gocov-html

```bash
go install github.com/axw/gocov/gocov@latest
go install github.com/matm/gocov-html@latest

# 使用
gocov test ./... | gocov-html > coverage.html
```

### 3.2 覆盖率徽章

在 README 中展示覆盖率徽章：

```markdown
![Coverage](https://img.shields.io/badge/coverage-96.5%25-brightgreen)
```

---

## 四、如何达到95%覆盖率？

### 4.1 识别未覆盖代码

```bash
# 查看哪些代码没有被覆盖
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep "0.0%"
```

### 4.2 提升策略

#### 策略1：补齐遗漏的错误处理分支

未覆盖代码最常见的位置是 **error != nil** 的分支：

```go
// 被测代码
func ProcessData(input string) (string, error) {
    data, err := parseInput(input)
    if err != nil {
        return "", err  // ❌ 经常遗漏测试
    }
    result, err := transform(data)
    if err != nil {
        return "", err  // ❌ 经常遗漏测试
    }
    return result, nil
}
```

**测试补齐：**
```go
func TestProcessData_ErrorPaths(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"invalid input", "bad", true},
        {"transform error", "trigger_transform_error", true},
        {"success", "valid", false},
    }
    // ...
}
```

#### 策略2：Mock外部依赖的异常情况

```go
// 测试数据库连接失败、网络超时等异常情况
mockRepo.On("GetUser", 1).Return(nil, errors.New("connection timeout"))
```

#### 策略3：测试边界条件

| 边界类型 | 示例 |
|----------|------|
| 空值 | `nil`, `""`, `0`, `[]` |
| 极限值 | `math.MaxInt64`, 超长字符串 |
| 刚好越界 | 切片索引 `-1` 和 `len` |
| 特殊字符 | Unicode, 换行符, null byte |

#### 策略4：使用模糊测试（Fuzzing）

Go 1.18+ 支持模糊测试，自动生成大量随机输入：

```go
func FuzzParseInput(f *testing.F) {
    f.Add("valid_input")
    f.Add("")
    
    f.Fuzz(func(t *testing.T, input string) {
        _, err := ParseInput(input)
        // 不应该panic
    })
}
```

#### 策略5：排除不可测试代码

某些代码本质上是不可测试的（如 `main()` 中的启动逻辑），应该将其精简：

```go
// main.go - 保持极简
func main() {
    if err := run(); err != nil {
        log.Fatal(err)
    }
}

// 核心逻辑移到可测试的函数
func run() error {
    // 所有逻辑在这里，可以被测试
    return nil
}
```

---

## 五、增量覆盖率（Diff Coverage）

团队实践中，更关注的是**本次变更的代码覆盖率**而非全量覆盖率。

### 5.1 使用 gocov-diff

```bash
go test -coverprofile=base.out ./...
# ... 修改代码 ...
go test -coverprofile=new.out ./...

# 比较差异
gocov-diff base.out new.out
```

### 5.2 CI中强制增量覆盖率

```yaml
- name: Check Diff Coverage
  run: |
    go test -coverprofile=coverage.out ./...
    # 要求新增代码覆盖率达到95%
    # 使用 codecov 或 sonarqube 实现
```

### 5.3 Codecov 集成

```yaml
- name: Upload coverage
  uses: codecov/codecov-action@v4
  with:
    file: ./coverage.out
    fail_ci_if_error: true
```

---

## 六、CI中强制覆盖率 ≥95%

### Makefile

```makefile
.PHONY: test coverage-check

test:
	go test -race -coverprofile=coverage.out ./...

coverage-check: test
	@echo "Checking coverage..."
	@go tool cover -func=coverage.out | awk 'END {print "Total coverage: " $$3; if ($$3+0 < 95.0) {print "Coverage below 95%!"; exit 1}}'
```

### GitHub Actions

```yaml
name: Coverage Check

on: [push, pull_request]

jobs:
  coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Run tests with coverage
        run: go test -race -coverprofile=coverage.out ./...
      
      - name: Check coverage threshold
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
          echo "Total coverage: ${COVERAGE}%"
          if (( $(echo "${COVERAGE} < 95.0" | bc -l) )); then
            echo "Coverage ${COVERAGE}% is below 95% threshold!"
            exit 1
          fi
```

---

## 七、覆盖率反模式（避免过度追求）

### ❌ 为覆盖而覆盖

```go
// 不要为了覆盖这行而写无意义的测试
if debug {
    log.Println("debug info") // 不需要测试
}
```

### ❌ 测试私有实现细节

```go
// 不要测试私有函数的内部状态
// 应该测试公共API的行为
```

### ❌ 忽略集成测试的贡献

覆盖率统计应同时考虑单元测试和集成测试：

```bash
# 合并单元测试和集成测试的覆盖率
go test -coverprofile=unit.out ./...
go test -coverprofile=integration.out -tags=integration ./...

# gocovmerge 合并报告
gocovmerge unit.out integration.out > total.out
```

---

## 八、覆盖率 checklist

- [ ] 核心业务逻辑覆盖率 ≥95%
- [ ] 所有错误处理分支都有测试
- [ ] 边界条件覆盖完整
- [ ] Mock了外部依赖的异常情况
- [ ] 使用 `go test -race` 检测数据竞争
- [ ] CI中配置覆盖率门禁
- [ ] 关注增量覆盖率而非仅全量覆盖率
- [ ] 不为无意义代码（如 debug log）硬凑测试
