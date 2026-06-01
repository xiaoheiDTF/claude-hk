# BDD + TDD: GET /api/logs 系统日志

> 接口: `GET /api/logs`
> 日期: 2026-06-01
> 状态: ❌ 未实现

---

## 一、接口定义

| 属性 | 值 |
|------|-----|
| 端点 | `/api/logs` |
| 方法 | GET |
| 功能 | 查询后端服务日志 |
| 查询参数 | `level` (可选), `date` (可选), `limit` (可选) |
| 响应 | `LogsResponse` |

---

## 二、BDD 测试用例

```gherkin
Feature: 查询系统日志
  作为运维人员
  我需要查看后端服务的运行日志
  以便排查问题和监控系统状态

  Background:
    Given 后端服务已启动
    And 日志目录存在以下日志文件:
      | 文件名         | 内容行数 |
      | 2026-06-01.log | 100      |
      | 2026-05-31.log | 200      |

  @positive
  Scenario: 获取当天日志
    When 发送 GET 请求到 /api/logs
    Then 响应状态码应为 200
    And 响应体应包含日志条目列表
    And 每个日志条目应包含:
      | timestamp | level | source | message |

  @positive
  Scenario: 按日期查询日志
    When 发送 GET 请求到 /api/logs?date=2026-05-31
    Then 响应状态码应为 200
    And 响应体应包含 2026-05-31 的日志

  @positive
  Scenario: 按级别过滤日志
    Given 当天日志包含 INFO、DEBUG、WARN、ERROR 级别的记录
    When 发送 GET 请求到 /api/logs?level=ERROR
    Then 响应状态码应为 200
    And 响应体中所有日志条目的 level 应为 "ERROR"

  @positive
  Scenario: 限制返回条数
    When 发送 GET 请求到 /api/logs?limit=10
    Then 响应状态码应为 200
    And 响应体应最多包含 10 条日志

  @positive
  Scenario: 指定日期无日志
    When 发送 GET 请求到 /api/logs?date=2026-01-01
    Then 响应状态码应为 200
    And 响应体应包含 {"logs": []}

  @negative
  Scenario: 使用非 GET 方法请求
    When 发送 POST 请求到 /api/logs
    Then 响应状态码应为 405
    And 响应体应包含 "method_not_allowed"
```

---

## 三、TDD 开发步骤

### Step 1: 定义响应类型

```go
// internal/backend/api/response.go

// LogsResponse 是日志查询的响应体。
type LogsResponse struct {
	Logs []LogItem `json:"logs"`
}

// LogItem 是单条日志记录。
type LogItem struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Source    string `json:"source"`
	Message   string `json:"message"`
}
```

### Step 2: 定义日志过滤条件

```go
// internal/backend/store/store.go

// LogFilter 是日志查询的过滤条件。
type LogFilter struct {
	Level *string // INFO/DEBUG/WARN/ERROR
	Date  *string // YYYY-MM-DD
	Limit int     // 最大返回条数，默认 100
}
```

### Step 3: 实现日志读取服务

```go
// internal/backend/service/log_service.go

package service

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type LogService struct {
	logDir string
}

func NewLogService(logDir string) *LogService {
	return &LogService{logDir: logDir}
}

// LogEntry 表示解析后的日志条目。
type LogEntry struct {
	Timestamp string
	Level     string
	Source    string
	Message   string
}

// QueryLogs 查询日志。
func (svc *LogService) QueryLogs(ctx context.Context, level, date string, limit int) ([]LogEntry, error) {
	logger.Debug("svc.log", "QueryLogs: level=%s date=%s limit=%d", level, date, limit)

	if limit <= 0 {
		limit = 100
	}

	// 确定日志文件路径
	logDate := date
	if logDate == "" {
		logDate = time.Now().Format("2006-01-02")
	}
	logFile := filepath.Join(svc.logDir, logDate+".log")

	// 检查文件是否存在
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return []LogEntry{}, nil
	}

	// 读取并过滤日志
	entries, err := svc.parseLogFile(logFile, level, limit)
	if err != nil {
		return nil, fmt.Errorf("parse log file: %w", err)
	}

	return entries, nil
}

// 日志格式: [2026-06-01 14:32:15] [INFO] [source] message
var logPattern = regexp.MustCompile(`\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\] \[(\w+)\] \[(\w+)\] (.*)`)

func (svc *LogService) parseLogFile(path, levelFilter string, limit int) ([]LogEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() && len(entries) < limit {
		line := scanner.Text()
		matches := logPattern.FindStringSubmatch(line)
		if len(matches) != 5 {
			continue
		}

		entry := LogEntry{
			Timestamp: matches[1],
			Level:     matches[2],
			Source:    matches[3],
			Message:   matches[4],
		}

		// 级别过滤
		if levelFilter != "" && !strings.EqualFold(entry.Level, levelFilter) {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}
```

### Step 4: 实现 Handler 层

```go
// internal/backend/api/log_handler.go

package api

import (
	"net/http"
	"strconv"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type LogHandler struct {
	svc *service.LogService
}

func NewLogHandler(svc *service.LogService) *LogHandler {
	return &LogHandler{svc: svc}
}

func (h *LogHandler) Query(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	level := r.URL.Query().Get("level")
	date := r.URL.Query().Get("date")
	limitStr := r.URL.Query().Get("limit")

	limit := 100
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	logger.Debug("api.log", "GET /api/logs level=%s date=%s limit=%d", level, date, limit)

	entries, err := h.svc.QueryLogs(r.Context(), level, date, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to query logs")
		return
	}

	items := make([]LogItem, len(entries))
	for i, e := range entries {
		items[i] = LogItem{
			Timestamp: e.Timestamp,
			Level:     e.Level,
			Source:    e.Source,
			Message:   e.Message,
		}
	}

	writeJSON(w, http.StatusOK, LogsResponse{Logs: items})
}
```

### Step 5: 注册路由

```go
// internal/backend/api/router.go

type Handlers struct {
	Issue   *IssueHandler
	Session *SessionHandler
	Proxy   *ProxyHandler
	Machine *MachineHandler
	Project *ProjectHandler
	Log     *LogHandler // 新增
}

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	// ... 已有路由
	mux.HandleFunc("/api/logs", h.Log.Query) // 新增
	return mux
}
```

---

## 四、Todo List

- [ ] Step 1: 定义 LogsResponse 和 LogItem 类型
- [ ] Step 2: 定义 LogFilter 过滤条件
- [ ] Step 3: 实现 LogService 和 parseLogFile 函数
- [ ] Step 4: 实现 LogHandler
- [ ] Step 5: 在 router.go 中注册 /api/logs 路由
- [ ] Step 6: 配置日志目录路径（从环境变量或配置读取）
- [ ] Step 7: 编写单元测试 log_api_test.go
- [ ] Step 8: 运行 BDD 测试验证
- [ ] Step 9: 更新 API 文档

---

*文档结束*
