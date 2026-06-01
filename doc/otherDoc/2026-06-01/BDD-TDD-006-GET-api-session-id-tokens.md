# BDD + TDD: GET /api/session/{id}/tokens Token 统计

> 接口: `GET /api/session/{id}/tokens`
> 日期: 2026-06-01
> 状态: ❌ 未实现

---

## 一、接口定义

| 属性 | 值 |
|------|-----|
| 端点 | `/api/session/{id}/tokens` |
| 方法 | GET |
| 功能 | 获取指定会话的 Token 使用统计 |
| 路径参数 | `id` — session_id |
| 响应 | `SessionTokensResponse` |

---

## 二、BDD 测试用例

```gherkin
Feature: 获取会话 Token 统计
  作为监控用户
  我需要查看某个会话的 Token 消耗情况
  以便了解 API 调用成本

  Background:
    Given 后端服务已启动
    And 数据库已初始化
    And 存在会话 "session-abc123"
    And 该会话的 trace 文件存在以下记录:
      | timestamp            | input_tokens | output_tokens | cache_read | cache_create |
      | 2026-06-01T14:32:00Z | 1024         | 256           | 0          | 0            |
      | 2026-06-01T14:33:00Z | 2048         | 512           | 1024       | 256          |
      | 2026-06-01T14:34:00Z | 512          | 128           | 0          | 0            |

  @positive
  Scenario: 获取会话 Token 统计
    When 发送 GET 请求到 /api/session/session-abc123/tokens
    Then 响应状态码应为 200
    And 响应体应包含
      """
      {
        "session_id": "session-abc123",
        "api_calls": 3,
        "input_tokens": 3584,
        "output_tokens": 896,
        "cache_read": 1024,
        "cache_create": 256,
        "total_tokens": 4480
      }
      """

  @positive
  Scenario: 会话无 Token 记录
    Given 存在会话 "session-empty"
    And 该会话无 trace 文件
    When 发送 GET 请求到 /api/session/session-empty/tokens
    Then 响应状态码应为 200
    And 响应体应包含
      """
      {
        "session_id": "session-empty",
        "api_calls": 0,
        "input_tokens": 0,
        "output_tokens": 0,
        "cache_read": 0,
        "cache_create": 0,
        "total_tokens": 0
      }
      """

  @negative
  Scenario: 获取不存在的会话
    When 发送 GET 请求到 /api/session/session-not-exist/tokens
    Then 响应状态码应为 404
    And 响应体应包含错误码 "not_found"

  @negative
  Scenario: 使用非 GET 方法请求
    When 发送 POST 请求到 /api/session/session-abc123/tokens
    Then 响应状态码应为 405
    And 响应体应包含 "method_not_allowed"
```

---

## 三、TDD 开发步骤

### Step 1: 定义响应类型

```go
// internal/backend/api/response.go

// SessionTokensResponse 是会话 Token 统计的响应体。
type SessionTokensResponse struct {
	SessionID    string `json:"session_id"`
	APICalls     int    `json:"api_calls"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	CacheRead    int    `json:"cache_read"`
	CacheCreate  int    `json:"cache_create"`
	TotalTokens  int    `json:"total_tokens"`
}
```

### Step 2: 定义 Token 统计结构

```go
// internal/backend/domain/token.go

package domain

// TokenStats 表示 Token 使用统计。
type TokenStats struct {
	APICalls     int
	InputTokens  int
	OutputTokens int
	CacheRead    int
	CacheCreate  int
}

// Total 计算总 Token 数。
func (t TokenStats) Total() int {
	return t.InputTokens + t.OutputTokens + t.CacheRead + t.CacheCreate
}
```

### Step 3: 实现 Trace 文件解析

```go
// internal/backend/service/token_service.go

package service

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/domain"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/logger"
)

type TokenService struct {
	sessionStore store.SessionStore
}

func NewTokenService(s store.SessionStore) *TokenService {
	return &TokenService{sessionStore: s}
}

// GetSessionTokens 获取指定会话的 Token 统计。
func (svc *TokenService) GetSessionTokens(ctx context.Context, sessionID string) (*domain.TokenStats, error) {
	logger.Debug("svc.token", "GetSessionTokens: session=%s", sessionID)

	// 获取会话信息
	sess, err := svc.sessionStore.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, store.ErrSessionNotFound
	}

	// 如果无 trace 路径，返回零值
	if sess.LocalTracePath == "" {
		return &domain.TokenStats{}, nil
	}

	// 解析 trace 文件
	stats, err := parseTraceFile(sess.LocalTracePath)
	if err != nil {
		logger.Warn("svc.token", "parse trace failed: %v", err)
		return &domain.TokenStats{}, nil
	}

	return stats, nil
}

// parseTraceFile 解析 JSONL trace 文件，汇总 Token 统计。
func parseTraceFile(path string) (*domain.TokenStats, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var stats domain.TokenStats
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var record struct {
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
				CacheRead    int `json:"cache_read,omitempty"`
				CacheCreate  int `json:"cache_create,omitempty"`
			} `json:"usage"`
		}

		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue // 跳过解析失败的行
		}

		stats.APICalls++
		stats.InputTokens += record.Usage.InputTokens
		stats.OutputTokens += record.Usage.OutputTokens
		stats.CacheRead += record.Usage.CacheRead
		stats.CacheCreate += record.Usage.CacheCreate
	}

	return &stats, scanner.Err()
}
```

### Step 4: 实现 Handler 层

```go
// internal/backend/api/session_handler.go

// GetTokens 处理获取会话 Token 统计的请求。
func (h *SessionHandler) GetTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}

	// 从 URL 路径提取 session_id
	path := r.URL.Path
	prefix := "/api/session/"
	suffix := "/tokens"
	sessionID := path[len(prefix) : len(path)-len(suffix)]
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "session_id is required")
		return
	}

	logger.Debug("api.session", "GET /api/session/%s/tokens", sessionID)

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

	// 获取 Token 统计
	stats, err := h.tokenSvc.GetSessionTokens(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get tokens")
		return
	}

	writeJSON(w, http.StatusOK, SessionTokensResponse{
		SessionID:    sessionID,
		APICalls:     stats.APICalls,
		InputTokens:  stats.InputTokens,
		OutputTokens: stats.OutputTokens,
		CacheRead:    stats.CacheRead,
		CacheCreate:  stats.CacheCreate,
		TotalTokens:  stats.Total(),
	})
}
```

### Step 5: 修改 SessionHandler 注入 TokenService

```go
// internal/backend/api/session_handler.go

type SessionHandler struct {
	svc      *service.SessionService
	issueSvc *service.IssueService
	tokenSvc *service.TokenService // 新增
}

func NewSessionHandler(svc *service.SessionService, issueSvc *service.IssueService, tokenSvc *service.TokenService) *SessionHandler {
	return &SessionHandler{svc: svc, issueSvc: issueSvc, tokenSvc: tokenSvc}
}
```

### Step 6: 注册路由

```go
// internal/backend/api/router.go

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	// ... 已有路由
	
	// Session 子路由
	mux.HandleFunc("/api/session/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/issues"):
			h.Session.GetIssues(w, r)
		case strings.HasSuffix(path, "/tokens"):
			h.Session.GetTokens(w, r)
		case strings.HasSuffix(path, "/traces"):
			h.Session.GetTraces(w, r)
		default:
			h.Session.Get(w, r)
		}
	})
	
	return mux
}
```

---

## 四、Todo List

- [ ] Step 1: 定义 SessionTokensResponse 和 TokenStats 类型
- [ ] Step 2: 实现 TokenService 和 parseTraceFile 函数
- [ ] Step 3: 修改 SessionHandler 注入 TokenService
- [ ] Step 4: 实现 SessionHandler.GetTokens
- [ ] Step 5: 在 router.go 中注册 /api/session/{id}/tokens 路由
- [ ] Step 6: 编写单元测试 token_api_test.go
- [ ] Step 7: 运行 BDD 测试验证
- [ ] Step 8: 更新 API 文档

---

*文档结束*
