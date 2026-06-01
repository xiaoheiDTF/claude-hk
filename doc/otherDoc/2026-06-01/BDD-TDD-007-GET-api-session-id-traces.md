# BDD + TDD: GET /api/session/{id}/traces Trace 文件列表

> 接口: `GET /api/session/{id}/traces`
> 日期: 2026-06-01
> 状态: ❌ 未实现

---

## 一、接口定义

| 属性 | 值 |
|------|-----|
| 端点 | `/api/session/{id}/traces` |
| 方法 | GET |
| 功能 | 获取指定会话关联的所有 trace 文件信息 |
| 路径参数 | `id` — session_id |
| 响应 | `SessionTracesResponse` |

---

## 二、BDD 测试用例

```gherkin
Feature: 获取会话 Trace 文件列表
  作为监控用户
  我需要查看某个会话的所有 trace 文件
  以便追溯和下载对话记录

  Background:
    Given 后端服务已启动
    And 数据库已初始化
    And 存在会话 "session-abc123"
    And 该会话 local_trace_path 为 ".traces/2026-06-01/trace_143052.jsonl"
    And trace 文件存在且大小为 2.3MB，共 156 行

  @positive
  Scenario: 获取会话 Trace 文件列表
    When 发送 GET 请求到 /api/session/session-abc123/traces
    Then 响应状态码应为 200
    And 响应体应包含
      """
      {
        "session_id": "session-abc123",
        "traces": [
          {
            "path": ".traces/2026-06-01/trace_143052.jsonl",
            "size_bytes": 2411724,
            "line_count": 156,
            "date": "2026-06-01",
            "filename": "trace_143052.jsonl"
          }
        ]
      }
      """

  @positive
  Scenario: 会话无 Trace 文件
    Given 存在会话 "session-no-trace"
    And 该会话 local_trace_path 为空
    When 发送 GET 请求到 /api/session/session-no-trace/traces
    Then 响应状态码应为 200
    And 响应体应包含 {"traces": []}

  @negative
  Scenario: 获取不存在的会话
    When 发送 GET 请求到 /api/session/session-not-exist/traces
    Then 响应状态码应为 404
    And 响应体应包含错误码 "not_found"

  @negative
  Scenario: 使用非 GET 方法请求
    When 发送 POST 请求到 /api/session/session-abc123/traces
    Then 响应状态码应为 405
    And 响应体应包含 "method_not_allowed"
```

---

## 三、TDD 开发步骤

### Step 1: 定义响应类型

```go
// internal/backend/api/response.go

// SessionTracesResponse 是会话 Trace 文件列表的响应体。
type SessionTracesResponse struct {
	SessionID string      `json:"session_id"`
	Traces    []TraceItem `json:"traces"`
}

// TraceItem 是单个 Trace 文件的信息。
type TraceItem struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	LineCount int    `json:"line_count"`
	Date      string `json:"date"`
	Filename  string `json:"filename"`
}
```

### Step 2: 实现 Trace 文件扫描服务

```go
// internal/backend/service/trace_service.go

package service

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type TraceService struct {
	sessionStore store.SessionStore
}

func NewTraceService(s store.SessionStore) *TraceService {
	return &TraceService{sessionStore: s}
}

// TraceFileInfo 表示 Trace 文件元数据。
type TraceFileInfo struct {
	Path      string
	SizeBytes int64
	LineCount int
	Date      string
	Filename  string
}

// GetSessionTraces 获取指定会话的所有 trace 文件信息。
func (svc *TraceService) GetSessionTraces(ctx context.Context, sessionID string) ([]TraceFileInfo, error) {
	logger.Debug("svc.trace", "GetSessionTraces: session=%s", sessionID)

	sess, err := svc.sessionStore.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, store.ErrSessionNotFound
	}

	if sess.LocalTracePath == "" {
		return []TraceFileInfo{}, nil
	}

	// 获取 trace 文件信息
	info, err := getTraceFileInfo(sess.LocalTracePath)
	if err != nil {
		logger.Warn("svc.trace", "get trace info failed: %v", err)
		return []TraceFileInfo{}, nil
	}

	return []TraceFileInfo{*info}, nil
}

// getTraceFileInfo 获取单个 trace 文件的元数据。
func getTraceFileInfo(path string) (*TraceFileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// 计算行数
	lineCount, err := countLines(path)
	if err != nil {
		lineCount = 0
	}

	dir := filepath.Dir(path)
	date := filepath.Base(dir)
	filename := filepath.Base(path)

	return &TraceFileInfo{
		Path:      path,
		SizeBytes: stat.Size(),
		LineCount: lineCount,
		Date:      date,
		Filename:  filename,
	}, nil
}

// countLines 计算文件行数。
func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}
```

### Step 3: 实现 Handler 层

```go
// internal/backend/api/session_handler.go

// GetTraces 处理获取会话 Trace 文件列表的请求。
func (h *SessionHandler) GetTraces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	path := r.URL.Path
	prefix := "/api/session/"
	suffix := "/traces"
	sessionID := path[len(prefix) : len(path)-len(suffix)]
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session_id is required")
		return
	}

	logger.Debug("api.session", "GET /api/session/%s/traces", sessionID)

	// 检查会话是否存在
	sess, err := h.svc.Get(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get session")
		return
	}
	if sess == nil {
		writeError(w, http.StatusNotFound, "not_found", "session not found")
		return
	}

	// 获取 Trace 文件列表
	traces, err := h.traceSvc.GetSessionTraces(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get traces")
		return
	}

	items := make([]TraceItem, len(traces))
	for i, t := range traces {
		items[i] = TraceItem{
			Path:      t.Path,
			SizeBytes: t.SizeBytes,
			LineCount: t.LineCount,
			Date:      t.Date,
			Filename:  t.Filename,
		}
	}

	writeJSON(w, http.StatusOK, SessionTracesResponse{
		SessionID: sessionID,
		Traces:    items,
	})
}
```

### Step 4: 修改 SessionHandler 注入 TraceService

```go
// internal/backend/api/session_handler.go

type SessionHandler struct {
	svc      *service.SessionService
	issueSvc *service.IssueService
	tokenSvc *service.TokenService
	traceSvc *service.TraceService // 新增
}

func NewSessionHandler(svc *service.SessionService, issueSvc *service.IssueService, tokenSvc *service.TokenService, traceSvc *service.TraceService) *SessionHandler {
	return &SessionHandler{svc: svc, issueSvc: issueSvc, tokenSvc: tokenSvc, traceSvc: traceSvc}
}
```

---

## 四、Todo List

- [ ] Step 1: 定义 SessionTracesResponse 和 TraceItem 类型
- [ ] Step 2: 实现 TraceService 和 getTraceFileInfo 函数
- [ ] Step 3: 修改 SessionHandler 注入 TraceService
- [ ] Step 4: 实现 SessionHandler.GetTraces
- [ ] Step 5: 在 router.go 中注册 /api/session/{id}/traces 路由
- [ ] Step 6: 编写单元测试 trace_api_test.go
- [ ] Step 7: 运行 BDD 测试验证
- [ ] Step 8: 更新 API 文档

---

*文档结束*
